import React from 'react';
import {
    Box,
    Button,
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

const Login = observer(() => {
    const [username, setUsername] = React.useState('');
    const [password, setPassword] = React.useState('');
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
        void currentUser.login(username, password);
    };

    return (
        <DefaultPage title="Sign in" maxWidth={460}>
            <SurfaceCard>
                <Stack spacing={2.5}>
                    <Box sx={{textAlign: 'center'}}>
                        <Box
                            component="img"
                            src={config.get('url') + 'static/gotify-mu-logo.png'}
                            alt="Gotify MU"
                            sx={{width: 180, maxWidth: '70%', mb: 1}}
                        />
                        <Typography variant="h5">Welcome to Gotify MU</Typography>
                        <Typography color="text.secondary">
                            Sign in to access your Channels and notifications.
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
                                <Button
                                    type="submit"
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
