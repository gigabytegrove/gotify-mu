import React, {useState} from 'react';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import {observer} from 'mobx-react-lite';
import {useStores} from '../stores';
import * as config from '../config';
import CircularProgress from '@mui/material/CircularProgress';
import Key from '@mui/icons-material/Key';
import {Box, Divider} from '@mui/material';

const ElevateDuration = 60 * 60;

const ElevationForm = observer(() => {
    const {elevateStore, currentUser} = useStores();
    const [password, setPassword] = useState('');
    const [mfaCode, setMfaCode] = useState('');
    const [error, setError] = useState('');

    const localAuthEnabled = config.get('localAuth');
    const oidcEnabled = config.get('oidc');
    const ldapEnabled = config.get('ldap');
    const ldapIdpName = config.get('ldapIdpName');
    const provider = currentUser.user.authProvider || 'local';
    const usePassword = provider === 'local' ? localAuthEnabled : provider === 'ldap' && ldapEnabled;
    const oidcPending = elevateStore.oidcElevatePending;
    const oidcIdpName = config.get('oidcIdpName');

    const handleLocalElevate = async () => {
        try {
            if (provider === 'ldap') {
                await elevateStore.directoryElevate(password, ElevateDuration);
            } else {
                await elevateStore.localElevate(password, ElevateDuration, mfaCode);
            }
        } catch {
            setError('Elevation failed. Check your password.');
        }
    };

    if (oidcPending) {
        return (
            <Box sx={{textAlign: 'center', my: 2}}>
                <CircularProgress sx={{mb: 2}} />
                <Typography sx={{mb: 1}}>Waiting for {oidcIdpName} sign-in...</Typography>
                <Typography variant="body2" color="textSecondary" sx={{mb: 1}}>
                    Complete sign-in in the new tab, then close it to continue.
                </Typography>
                <Button
                    className="elevation-oidc-cancel"
                    variant="outlined"
                    fullWidth
                    onClick={() => elevateStore.cleanupOidcElevate()}>
                    Cancel {oidcIdpName} Login
                </Button>
            </Box>
        );
    }

    return (
        <>
            <Typography>This action requires re-authentication.</Typography>
            {usePassword && (
                <form
                    onSubmit={(e) => {
                        e.preventDefault();
                        handleLocalElevate();
                    }}>
                    <TextField
                        autoFocus
                        margin="dense"
                        type="password"
                        label={provider === 'ldap' ? ldapIdpName + ' Password' : 'Password'}
                        className="elevation-password"
                        value={password}
                        onChange={(e) => {
                            setPassword(e.target.value);
                            setError('');
                        }}
                        fullWidth
                        error={!!error}
                        helperText={error}
                    />
                    {provider === 'local' && currentUser.user.mfaEnabled && (
                        <TextField
                            margin="dense"
                            label="Verification code"
                            className="elevation-mfa-code"
                            value={mfaCode}
                            onChange={(e) => {
                                setMfaCode(e.target.value);
                                setError('');
                            }}
                            autoComplete="one-time-code"
                            helperText="Authenticator code or recovery code."
                            fullWidth
                        />
                    )}
                    <Button
                        type="submit"
                        className="elevation-submit"
                        disabled={
                            password.length === 0 ||
                            (provider === 'local' &&
                                Boolean(currentUser.user.mfaEnabled) &&
                                mfaCode.length === 0)
                        }
                        color="primary"
                        variant="contained"
                        fullWidth>
                        {provider === 'ldap' ? 'Confirm with ' + ldapIdpName : 'Elevate with Password'}
                    </Button>
                </form>
            )}

            {Boolean(currentUser.user.passkeyCount) && (
                <>
                    {usePassword && <Divider sx={{my: 2}}>or</Divider>}
                    <Button
                        className="elevation-passkey"
                        variant="outlined"
                        startIcon={<Key />}
                        fullWidth
                        onClick={async () => {
                            try {
                                await elevateStore.passkeyElevate();
                            } catch {
                                setError('Passkey verification was not completed.');
                            }
                        }}>
                        Confirm with Passkey
                    </Button>
                </>
            )}

            {oidcEnabled && provider === 'oidc' && (
                <>
                    {usePassword && <Divider sx={{my: 2}}>or</Divider>}
                    <Button
                        className="elevation-oidc"
                        variant="contained"
                        color="primary"
                        fullWidth
                        onClick={() => elevateStore.oidcElevate(ElevateDuration)}>
                        Elevate via {oidcIdpName}
                    </Button>
                </>
            )}
        </>
    );
});

export default ElevationForm;
