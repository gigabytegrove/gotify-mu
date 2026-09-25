package passkey

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

type ClientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
}

type RegistrationResult struct {
	CredentialID string
	PublicKeyX   string
	PublicKeyY   string
	SignCount    uint32
}

type AuthenticationResult struct {
	SignCount uint32
}

func RandomValue(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func VerifyRegistration(
	clientDataJSON, attestationObject []byte,
	expectedChallenge, expectedOrigin, rpID string,
) (*RegistrationResult, error) {
	clientData, clientHash, err := validateClientData(
		clientDataJSON, "webauthn.create", expectedChallenge, expectedOrigin,
	)
	if err != nil { return nil, err }
	_ = clientData
	_ = clientHash

	decoded, _, err := decodeCBOR(attestationObject, 0)
	if err != nil { return nil, fmt.Errorf("decode attestation: %w", err) }
	object, ok := decoded.(map[any]any)
	if !ok { return nil, errors.New("attestation object is invalid") }
	format, _ := object["fmt"].(string)
	if format != "none" {
		return nil, errors.New("only privacy-preserving none attestation is accepted")
	}
	authData, ok := object["authData"].([]byte)
	if !ok { return nil, errors.New("attestation is missing authenticator data") }
	rpHash, flags, signCount, rest, err := parseAuthenticatorData(authData)
	if err != nil { return nil, err }
	if !bytes.Equal(rpHash, hashRPID(rpID)) {
		return nil, errors.New("passkey RP ID does not match this server")
	}
	if flags&0x01 == 0 { return nil, errors.New("user presence was not verified") }
	if flags&0x40 == 0 { return nil, errors.New("attested credential data is missing") }
	if len(rest) < 18 { return nil, errors.New("attested credential data is truncated") }
	credentialLength := int(binary.BigEndian.Uint16(rest[16:18]))
	if len(rest) < 18+credentialLength {
		return nil, errors.New("credential id is truncated")
	}
	credentialID := rest[18 : 18+credentialLength]
	keyData := rest[18+credentialLength:]
	keyValue, _, err := decodeCBOR(keyData, 0)
	if err != nil { return nil, fmt.Errorf("decode credential public key: %w", err) }
	keyMap, ok := keyValue.(map[any]any)
	if !ok { return nil, errors.New("credential public key is invalid") }

	kty, _ := integerValue(keyMap[int64(1)])
	alg, _ := integerValue(keyMap[int64(3)])
	crv, _ := integerValue(keyMap[int64(-1)])
	x, xOK := keyMap[int64(-2)].([]byte)
	y, yOK := keyMap[int64(-3)].([]byte)
	if kty != 2 || alg != -7 || crv != 1 || !xOK || !yOK || len(x) != 32 || len(y) != 32 {
		return nil, errors.New("passkey must use an ES256 P-256 public key")
	}
	if !elliptic.P256().IsOnCurve(new(big.Int).SetBytes(x), new(big.Int).SetBytes(y)) {
		return nil, errors.New("passkey public key is not on the P-256 curve")
	}
	return &RegistrationResult{
		CredentialID: base64.RawURLEncoding.EncodeToString(credentialID),
		PublicKeyX: base64.RawURLEncoding.EncodeToString(x),
		PublicKeyY: base64.RawURLEncoding.EncodeToString(y),
		SignCount: signCount,
	}, nil
}

func VerifyAuthentication(
	clientDataJSON, authenticatorData, signature []byte,
	expectedChallenge, expectedOrigin, rpID, publicX, publicY string,
	previousSignCount uint32,
) (*AuthenticationResult, error) {
	_, clientHash, err := validateClientData(
		clientDataJSON, "webauthn.get", expectedChallenge, expectedOrigin,
	)
	if err != nil { return nil, err }
	rpHash, flags, signCount, _, err := parseAuthenticatorData(authenticatorData)
	if err != nil { return nil, err }
	if !bytes.Equal(rpHash, hashRPID(rpID)) {
		return nil, errors.New("passkey RP ID does not match this server")
	}
	if flags&0x01 == 0 { return nil, errors.New("user presence was not verified") }

	xBytes, err := base64.RawURLEncoding.DecodeString(publicX)
	if err != nil { return nil, errors.New("stored passkey public key is invalid") }
	yBytes, err := base64.RawURLEncoding.DecodeString(publicY)
	if err != nil { return nil, errors.New("stored passkey public key is invalid") }
	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X: new(big.Int).SetBytes(xBytes),
		Y: new(big.Int).SetBytes(yBytes),
	}
	if !publicKey.Curve.IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, errors.New("stored passkey public key is invalid")
	}
	signed := append(append([]byte(nil), authenticatorData...), clientHash...)
	digest := sha256.Sum256(signed)
	if !ecdsa.VerifyASN1(publicKey, digest[:], signature) {
		return nil, errors.New("passkey signature is invalid")
	}
	if previousSignCount > 0 && signCount > 0 && signCount <= previousSignCount {
		return nil, errors.New("passkey counter did not advance")
	}
	return &AuthenticationResult{SignCount: signCount}, nil
}

func validateClientData(
	raw []byte,
	expectedType, expectedChallenge, expectedOrigin string,
) (*ClientData, []byte, error) {
	var client ClientData
	if err := json.Unmarshal(raw, &client); err != nil {
		return nil, nil, errors.New("client data is invalid")
	}
	if client.Type != expectedType {
		return nil, nil, errors.New("unexpected WebAuthn operation")
	}
	if client.Challenge != expectedChallenge {
		return nil, nil, errors.New("passkey challenge does not match")
	}
	if normalizeOrigin(client.Origin) != normalizeOrigin(expectedOrigin) {
		return nil, nil, errors.New("passkey origin does not match this server")
	}
	hash := sha256.Sum256(raw)
	return &client, hash[:], nil
}

func parseAuthenticatorData(data []byte) ([]byte, byte, uint32, []byte, error) {
	if len(data) < 37 {
		return nil, 0, 0, nil, errors.New("authenticator data is truncated")
	}
	rpHash := data[:32]
	flags := data[32]
	signCount := binary.BigEndian.Uint32(data[33:37])
	return rpHash, flags, signCount, data[37:], nil
}

func hashRPID(rpID string) []byte {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(rpID))))
	return sum[:]
}

func integerValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case uint64:
		if typed > uint64(^uint64(0)>>1) { return 0, false }
		return int64(typed), true
	default:
		return 0, false
	}
}

func decodeCBOR(data []byte, offset int) (any, int, error) {
	if offset >= len(data) { return nil, offset, errors.New("unexpected end of CBOR") }
	initial := data[offset]
	offset++
	major := initial >> 5
	additional := initial & 0x1f
	length, next, err := cborLength(data, offset, additional)
	if err != nil { return nil, offset, err }
	offset = next

	switch major {
	case 0:
		return length, offset, nil
	case 1:
		return -1 - int64(length), offset, nil
	case 2:
		if length > uint64(len(data)-offset) { return nil, offset, errors.New("CBOR bytes truncated") }
		end := offset + int(length)
		value := append([]byte(nil), data[offset:end]...)
		return value, end, nil
	case 3:
		if length > uint64(len(data)-offset) { return nil, offset, errors.New("CBOR text truncated") }
		end := offset + int(length)
		return string(data[offset:end]), end, nil
	case 4:
		items := make([]any, 0, int(length))
		for i := uint64(0); i < length; i++ {
			item, nextOffset, err := decodeCBOR(data, offset)
			if err != nil { return nil, offset, err }
			items = append(items, item)
			offset = nextOffset
		}
		return items, offset, nil
	case 5:
		result := make(map[any]any, int(length))
		for i := uint64(0); i < length; i++ {
			key, nextOffset, err := decodeCBOR(data, offset)
			if err != nil { return nil, offset, err }
			offset = nextOffset
			value, nextOffset, err := decodeCBOR(data, offset)
			if err != nil { return nil, offset, err }
			offset = nextOffset
			switch key.(type) {
			case string, int64, uint64:
				result[key] = value
			default:
				return nil, offset, errors.New("unsupported CBOR map key")
			}
		}
		return result, offset, nil
	case 7:
		switch additional {
		case 20: return false, offset, nil
		case 21: return true, offset, nil
		case 22: return nil, offset, nil
		default: return nil, offset, errors.New("unsupported CBOR simple value")
		}
	default:
		return nil, offset, errors.New("unsupported CBOR type")
	}
}

func cborLength(data []byte, offset int, additional byte) (uint64, int, error) {
	switch {
	case additional < 24:
		return uint64(additional), offset, nil
	case additional == 24:
		if offset+1 > len(data) { return 0, offset, errors.New("CBOR length truncated") }
		return uint64(data[offset]), offset+1, nil
	case additional == 25:
		if offset+2 > len(data) { return 0, offset, errors.New("CBOR length truncated") }
		return uint64(binary.BigEndian.Uint16(data[offset:offset+2])), offset+2, nil
	case additional == 26:
		if offset+4 > len(data) { return 0, offset, errors.New("CBOR length truncated") }
		return uint64(binary.BigEndian.Uint32(data[offset:offset+4])), offset+4, nil
	case additional == 27:
		if offset+8 > len(data) { return 0, offset, errors.New("CBOR length truncated") }
		return binary.BigEndian.Uint64(data[offset:offset+8]), offset+8, nil
	default:
		return 0, offset, errors.New("indefinite CBOR lengths are not supported")
	}
}
