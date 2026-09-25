package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math/big"
)

func NewWebAuthnChallenge() (string,error) {
	raw:=make([]byte,32)
	if _,err:=rand.Read(raw);err!=nil{return "",err}
	return base64.RawURLEncoding.EncodeToString(raw),nil
}

type cborDecoder struct{ data []byte; offset int }

func (d *cborDecoder) readN(n int)([]byte,error){
	if n<0||d.offset+n>len(d.data){return nil,errors.New("invalid CBOR length")}
	out:=d.data[d.offset:d.offset+n];d.offset+=n;return out,nil
}
func (d *cborDecoder) readUint(add byte)(uint64,error){
	switch {
	case add<24:return uint64(add),nil
	case add==24:b,err:=d.readN(1);if err!=nil{return 0,err};return uint64(b[0]),nil
	case add==25:b,err:=d.readN(2);if err!=nil{return 0,err};return uint64(binary.BigEndian.Uint16(b)),nil
	case add==26:b,err:=d.readN(4);if err!=nil{return 0,err};return uint64(binary.BigEndian.Uint32(b)),nil
	case add==27:b,err:=d.readN(8);if err!=nil{return 0,err};return binary.BigEndian.Uint64(b),nil
	default:return 0,errors.New("unsupported CBOR integer")
	}
}
func (d *cborDecoder) item()(any,error){
	head,err:=d.readN(1);if err!=nil{return nil,err}
	major:=head[0]>>5;add:=head[0]&0x1f
	switch major {
	case 0:
		n,err:=d.readUint(add);return int64(n),err
	case 1:
		n,err:=d.readUint(add);if err!=nil{return nil,err};return -1-int64(n),nil
	case 2:
		n,err:=d.readUint(add);if err!=nil{return nil,err};return d.readN(int(n))
	case 3:
		n,err:=d.readUint(add);if err!=nil{return nil,err};b,err:=d.readN(int(n));return string(b),err
	case 4:
		n,err:=d.readUint(add);if err!=nil{return nil,err};out:=make([]any,0,n)
		for i:=uint64(0);i<n;i++{v,err:=d.item();if err!=nil{return nil,err};out=append(out,v)}
		return out,nil
	case 5:
		n,err:=d.readUint(add);if err!=nil{return nil,err};out:=map[any]any{}
		for i:=uint64(0);i<n;i++{k,err:=d.item();if err!=nil{return nil,err};v,err:=d.item();if err!=nil{return nil,err};out[k]=v}
		return out,nil
	case 6:
		if _,err:=d.readUint(add);err!=nil{return nil,err};return d.item()
	case 7:
		switch add{case 20:return false,nil;case 21:return true,nil;case 22:return nil,nil;default:return nil,errors.New("unsupported CBOR simple value")}
	default:return nil,errors.New("unsupported CBOR value")
	}
}

type WebAuthnRegistration struct {
	CredentialID []byte
	PublicKeyX []byte
	PublicKeyY []byte
	SignCount uint32
	Flags byte
	RPIDHash []byte
}

func ParseWebAuthnAttestation(attestationObject []byte)(WebAuthnRegistration,error){
	var result WebAuthnRegistration
	decoder:=&cborDecoder{data:attestationObject}
	value,err:=decoder.item();if err!=nil{return result,err}
	root,ok:=value.(map[any]any);if !ok{return result,errors.New("invalid attestation object")}
	rawAuth,ok:=root["authData"].([]byte);if !ok{return result,errors.New("attestation is missing authenticator data")}
	if len(rawAuth)<55{return result,errors.New("authenticator data is too short")}
	result.RPIDHash=append([]byte(nil),rawAuth[:32]...)
	result.Flags=rawAuth[32]
	result.SignCount=binary.BigEndian.Uint32(rawAuth[33:37])
	if result.Flags&0x01==0{return result,errors.New("user presence was not verified")}
	if result.Flags&0x40==0{return result,errors.New("attested credential data is missing")}
	offset:=37+16
	if len(rawAuth)<offset+2{return result,errors.New("credential id length is missing")}
	credentialLength:=int(binary.BigEndian.Uint16(rawAuth[offset:offset+2]));offset+=2
	if credentialLength<=0||len(rawAuth)<offset+credentialLength{return result,errors.New("invalid credential id")}
	result.CredentialID=append([]byte(nil),rawAuth[offset:offset+credentialLength]...)
	offset+=credentialLength
	keyDecoder:=&cborDecoder{data:rawAuth[offset:]}
	keyValue,err:=keyDecoder.item();if err!=nil{return result,err}
	keyMap,ok:=keyValue.(map[any]any);if !ok{return result,errors.New("invalid credential public key")}
	getInt:=func(key int64)(int64,bool){v,ok:=keyMap[key];if !ok{return 0,false};n,ok:=v.(int64);return n,ok}
	kty,_:=getInt(1);alg,_:=getInt(3);crv,_:=getInt(-1)
	if kty!=2||alg!=-7||crv!=1{return result,errors.New("only ES256 P-256 passkeys are supported")}
	x,okX:=keyMap[int64(-2)].([]byte);y,okY:=keyMap[int64(-3)].([]byte)
	if !okX||!okY||len(x)!=32||len(y)!=32{return result,errors.New("invalid P-256 public key")}
	result.PublicKeyX=append([]byte(nil),x...);result.PublicKeyY=append([]byte(nil),y...)
	return result,nil
}

type webAuthnClientData struct {
	Type string `json:"type"`
	Challenge string `json:"challenge"`
	Origin string `json:"origin"`
	CrossOrigin bool `json:"crossOrigin"`
}

func VerifyWebAuthnClientData(raw []byte, expectedType, challenge, origin string) error {
	var data webAuthnClientData
	if err:=json.Unmarshal(raw,&data);err!=nil{return err}
	if data.Type!=expectedType{return errors.New("unexpected WebAuthn operation type")}
	if data.Challenge!=challenge{return errors.New("WebAuthn challenge does not match")}
	if data.Origin!=origin{return errors.New("WebAuthn origin does not match")}
	if data.CrossOrigin{return errors.New("cross-origin WebAuthn is not allowed")}
	return nil
}

func VerifyWebAuthnRPID(hash []byte, rpID string) error {
	expected:=sha256.Sum256([]byte(rpID))
	if len(hash)!=len(expected)||!bytesEqual(hash,expected[:]){return errors.New("WebAuthn relying-party id does not match")}
	return nil
}
func bytesEqual(a,b []byte)bool{
	if len(a)!=len(b){return false};var diff byte;for i:=range a{diff|=a[i]^b[i]};return diff==0
}

func VerifyWebAuthnAssertion(authenticatorData, clientDataJSON, signature, x, y []byte, rpID string, previousSignCount uint32)(uint32,error){
	if len(authenticatorData)<37{return 0,errors.New("authenticator data is too short")}
	if err:=VerifyWebAuthnRPID(authenticatorData[:32],rpID);err!=nil{return 0,err}
	if authenticatorData[32]&0x01==0{return 0,errors.New("user presence was not verified")}
	signCount:=binary.BigEndian.Uint32(authenticatorData[33:37])
	xInt:=new(big.Int).SetBytes(x);yInt:=new(big.Int).SetBytes(y)
	if !elliptic.P256().IsOnCurve(xInt,yInt){return 0,errors.New("stored passkey public key is invalid")}
	clientHash:=sha256.Sum256(clientDataJSON)
	signed:=append(append([]byte(nil),authenticatorData...),clientHash[:]...)
	digest:=sha256.Sum256(signed)
	pub:=ecdsa.PublicKey{Curve:elliptic.P256(),X:xInt,Y:yInt}
	if !ecdsa.VerifyASN1(&pub,digest[:],signature){return 0,errors.New("passkey signature verification failed")}
	if previousSignCount>0&&signCount>0&&signCount<=previousSignCount{return 0,errors.New("passkey signature counter did not advance")}
	return signCount,nil
}
