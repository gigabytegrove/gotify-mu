import axios from 'axios';
import * as config from './config';
import {ICurrentUser} from './types';

export const decodeBase64Url = (value: string): ArrayBuffer => {
    const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
    const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4);
    const binary = atob(padded);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    return bytes.buffer;
};

export const encodeBase64Url = (value: ArrayBuffer): string => {
    const bytes = new Uint8Array(value);
    let binary = '';
    bytes.forEach((byte) => (binary += String.fromCharCode(byte)));
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
};

interface RequestOptions {
    challenge: string;
    rpId: string;
    timeout: number;
    userVerification: UserVerificationRequirement;
    allowCredentials: Array<{type: PublicKeyCredentialType; id: string}>;
}

const getAssertion = async (options: RequestOptions) => {
    if (!navigator.credentials) throw new Error('Passkeys are not supported by this browser.');
    const credential = (await navigator.credentials.get({
        publicKey: {
            challenge: decodeBase64Url(options.challenge),
            rpId: options.rpId,
            timeout: options.timeout,
            userVerification: options.userVerification,
            allowCredentials: options.allowCredentials.map((item) => ({
                type: item.type,
                id: decodeBase64Url(item.id),
            })),
        },
    })) as PublicKeyCredential | null;
    if (!credential) throw new Error('Passkey verification was cancelled.');
    const response = credential.response as AuthenticatorAssertionResponse;
    return {
        credentialId: encodeBase64Url(credential.rawId),
        clientDataJSON: encodeBase64Url(response.clientDataJSON),
        authenticatorData: encodeBase64Url(response.authenticatorData),
        signature: encodeBase64Url(response.signature),
    };
};

export const loginWithPasskey = async (
    username: string,
    clientName: string
): Promise<ICurrentUser> => {
    const options = await axios
        .post<RequestOptions>(config.get('url') + 'auth/passkey/login/options', {
            username,
            clientName,
        })
        .then((response) => response.data);
    const assertion = await getAssertion(options);
    return axios
        .post<ICurrentUser>(config.get('url') + 'auth/passkey/login/verify', assertion)
        .then((response) => response.data);
};

interface RegistrationOptions {
    challenge: string;
    rp: PublicKeyCredentialRpEntity;
    user: {id: string; name: string; displayName: string};
    pubKeyCredParams: PublicKeyCredentialParameters[];
    timeout: number;
    attestation: AttestationConveyancePreference;
    authenticatorSelection: AuthenticatorSelectionCriteria;
    excludeCredentials: Array<{type: PublicKeyCredentialType; id: string}>;
}

export const createPasskey = async (name: string): Promise<void> => {
    if (!navigator.credentials) throw new Error('Passkeys are not supported by this browser.');
    const options = await axios
        .post<RegistrationOptions>(config.get('url') + 'current/user/passkeys/options')
        .then((response) => response.data);
    const credential = (await navigator.credentials.create({
        publicKey: {
            ...options,
            challenge: decodeBase64Url(options.challenge),
            user: {...options.user, id: decodeBase64Url(options.user.id)},
            excludeCredentials: options.excludeCredentials.map((item) => ({
                type: item.type,
                id: decodeBase64Url(item.id),
            })),
        },
    })) as PublicKeyCredential | null;
    if (!credential) throw new Error('Passkey creation was cancelled.');
    const response = credential.response as AuthenticatorAttestationResponse;
    await axios.post(config.get('url') + 'current/user/passkeys', {
        name,
        credentialId: encodeBase64Url(credential.rawId),
        clientDataJSON: encodeBase64Url(response.clientDataJSON),
        attestationObject: encodeBase64Url(response.attestationObject),
    });
};

export const elevateWithPasskey = async (): Promise<void> => {
    const options = await axios
        .post<RequestOptions>(config.get('url') + 'current/user/passkeys/elevate/options')
        .then((response) => response.data);
    const assertion = await getAssertion(options);
    await axios.post(config.get('url') + 'current/user/passkeys/elevate/verify', assertion);
};
