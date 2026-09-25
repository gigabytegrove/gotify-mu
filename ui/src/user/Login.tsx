import React from 'react';
import {
    Box,
    Button,
    Chip,
    Divider,
    Stack,
    TextField,
    Typography,
} from '@mui/material';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import * as config from '../config';
import RegistrationDialog from './Register';
import {useStores} from '../stores';
import {observer} from 'mobx-react-lite';
import {useNavigate, useSearchParams} from 'react-router';
import LockOutlined from '@mui/icons-material/LockOutlined';

const Login = observer(() => {
    const [username, setUsername] = React.useState('');
    const [password, setPassword] = React.useState('');
    const [mfaCode, setMfaCode] = React.useState('');
    const [registerDialog, setRegisterDialog] = React.useState(false);
    const {currentUser} = useStores();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();

    const localAuthEnabled = config.get('localAuth');
    const oidcEnabled = config.get('oidc');
    const oidcIdpName = config.get('oidcIdpName');

    const oidcAutoRedirect =
        oidcEnabled &&
        config.get('oidcAutoRedirect') &&
        searchParams.get('redirect') !== 'false' &&
        !currentUser.connectionErrorMessage;

    const oidcLoginUrl =
        config.get('url') +
        'auth/oidc/login?name=' +
        encodeURIComponent(currentUser.createClientName());

    React.useEffect(() => {
        if (currentUser.loggedIn) {
            navigate('/');
            return;
        }
        if (!currentUser.authenticating && oidcAutoRedirect) {
            window.location.href = oidcLoginUrl;
        }
    }, [
        currentUser.loggedIn,
        currentUser.authenticating,
        navigate,
        oidcAutoRedirect,
        oidcLoginUrl,
    ]);

    const login = (event: React.FormEvent) => {
        event.preventDefault();
        void currentUser.login(username, password, mfaCode);
    };

    return (
        <DefaultPage title="Sign in" maxWidth={430}>
            <SurfaceCard>
                <Stack spacing={2}>
                    <Box sx={{textAlign: 'center', pt: 0.5}}>
                        <Box
                            component="img"
                            src={config.get('url') + 'static/gotify-mu-logo.png'}
                            alt="Gotify MU"
                            sx={{width: 150, maxWidth: '65%', mb: 0.75}}
                        />
                        <Typography variant="h5">Welcome to Gotify MU</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{mt: 0.4}}>
                            Multi-user notification delivery with shared Channels and centralized
                            administration.
                        </Typography>
                    </Box>

                    {localAuthEnabled && (
                        <Box component="form" id="login-form" onSubmit={login}>
                            <Stack spacing={2}>
                                <TextField
                                    autoFocus
                                    id="username"
                                    className="name"
                                    label="Username"
                                    name="username"
                                    autoComplete="username"
                                    value={username}
                                    onChange={(event) => setUsername(event.target.value)}
                                    fullWidth
                                />
                                <TextField
                                    id="password"
                                    type="password"
                                    className="password"
                                    label="Password"
                                    name="password"
                                    autoComplete="current-password"
                                    value={password}
                                    onChange={(event) => setPassword(event.target.value)}
                                    fullWidth
                                />
                                {currentUser.mfaRequired && (
                                    <TextField
                                        autoFocus
                                        id="mfa-code"
                                        label="Authenticator or recovery code"
                                        value={mfaCode}
                                        onChange={(event) => setMfaCode(event.target.value)}
                                        autoComplete="one-time-code"
                                        fullWidth
                                    />
                                )}
                                <Button
                                    type="submit"
                                    startIcon={<LockOutlined />}
                                    variant="contained"
                                    size="large"
                                    className="login"
                                    disabled={
                                        Boolean(currentUser.connectionErrorMessage) ||
                                        currentUser.authenticating
                                    }
                                    loading={currentUser.authenticating}
                                    fullWidth>
                                    Sign In
                                </Button>
                            </Stack>
                        </Box>
                    )}

                    {oidcEnabled && (
                        <>
                            {localAuthEnabled && <Divider>or</Divider>}
                            <Button
                                id="oidc-login"
                                component="a"
                                href={oidcLoginUrl}
                                variant="outlined"
                                size="large"
                                fullWidth>
                                Sign in with {oidcIdpName}
                            </Button>
                        </>
                    )}

                    {localAuthEnabled && config.get('register') && (
                        <Button
                            id="register"
                            onClick={() => setRegisterDialog(true)}
                            fullWidth>
                            Create an account
                        </Button>
                    )}

                    <Stack
                        direction="row"
                        spacing={0.75}
                        useFlexGap
                        sx={{pt: 0.5, justifyContent: 'center', flexWrap: 'wrap'}}>
                        <Chip
                            size="small"
                            variant="outlined"
                            label={`@${config.get('version').version}`}
                        />
                        {localAuthEnabled && (
                            <Chip size="small" variant="outlined" label="Local auth" />
                        )}
                        {oidcEnabled && (
                            <Chip size="small" variant="outlined" label={oidcIdpName} />
                        )}
                    </Stack>
                </Stack>
            </SurfaceCard>

            {registerDialog && (
                <RegistrationDialog
                    fClose={() => setRegisterDialog(false)}
                    fOnSubmit={currentUser.register}
                />
            )}
        </DefaultPage>
    );
});

export default Login;
