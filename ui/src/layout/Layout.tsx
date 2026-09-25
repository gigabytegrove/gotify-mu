import {
    Box,
    CssBaseline,
    Paper,
    StyledEngineProvider,
    ThemeProvider,
    useMediaQuery,
} from '@mui/material';
import * as React from 'react';
import {HashRouter, Navigate, Route, Routes} from 'react-router';
import Header from './Header';
import Navigation from './Navigation';
import ScrollUpButton from '../common/ScrollUpButton';
import ElevationForm from '../common/ElevationForm';
import * as config from '../config';
import Dashboard from '../dashboard/Dashboard';
import Applications from '../application/Applications';
import Clients from '../client/Clients';
import Plugins from '../plugin/Plugins';
import Login from '../user/Login';
import Messages from '../message/Messages';
import Settings from '../user/Settings';
import Users from '../user/Users';
import {observer} from 'mobx-react-lite';
import {ConnectionErrorBanner} from '../common/ConnectionErrorBanner';
import {useStores} from '../stores';
import {SnackbarProvider} from 'notistack';
import LoadingSpinner from '../common/LoadingSpinner';
import {createGotifyMuTheme, isThemeKey, ThemeKey} from './theme';
import DefaultPage from '../common/DefaultPage';

const localStorageThemeKey = 'gotify-theme';

const Layout = observer(() => {
    const {
        currentUser: {
            loggedIn,
            authenticating,
            user: {name, admin},
            logout,
            tryReconnect,
            connectionErrorMessage,
            refreshKey,
        },
    } = useStores();

    const [currentTheme, setCurrentTheme] = React.useState<ThemeKey>(() => {
        const stored = window.localStorage.getItem(localStorageThemeKey);
        return isThemeKey(stored) ? stored : 'system';
    });
    const prefersDark = useMediaQuery('(prefers-color-scheme: dark)');
    const paletteMode = currentTheme === 'system' ? (prefersDark ? 'dark' : 'light') : currentTheme;
    const theme = React.useMemo(() => createGotifyMuTheme(paletteMode), [paletteMode]);
    const {version} = config.get('version');
    const [navOpen, setNavOpen] = React.useState(false);

    const setTheme = (next: ThemeKey) => {
        setCurrentTheme(next);
        localStorage.setItem(localStorageThemeKey, next);
    };

    const authed = (children: React.ReactNode) => (
        <RequireAuth loggedIn={loggedIn} authenticating={authenticating}>
            {children}
        </RequireAuth>
    );

    const elevated = (children: React.ReactNode) => <RequireElevation>{children}</RequireElevation>;

    return (
        <StyledEngineProvider injectFirst>
            <ThemeProvider theme={theme}>
                <HashRouter>
                    <CssBaseline />
                    <div key={refreshKey}>
                        {connectionErrorMessage && (
                            <ConnectionErrorBanner
                                height={64}
                                retry={() => tryReconnect()}
                                message={connectionErrorMessage}
                            />
                        )}

                        <Header
                            admin={admin}
                            name={name}
                            style={{top: 0}}
                            version={version}
                            loggedIn={loggedIn}
                            logout={logout}
                            setNavOpen={setNavOpen}
                        />

                        <Box sx={{display: 'flex', minHeight: 'calc(100vh - 64px)'}}>
                            {loggedIn && (
                                <Navigation
                                    loggedIn={loggedIn}
                                    navOpen={navOpen}
                                    setNavOpen={setNavOpen}
                                />
                            )}

                            <Box
                                component="main"
                                sx={{
                                    flex: 1,
                                    minWidth: 0,
                                    px: {xs: 1.5, sm: 2.5, lg: 4},
                                    py: {xs: 2, sm: 3.5},
                                    overflowX: 'hidden',
                                }}>
                                <Routes>
                                    <Route path="/login" element={<Login />} />
                                    <Route path="/" element={authed(<Dashboard />)} />
                                    <Route path="/messages" element={authed(<Messages />)} />
                                    <Route path="/channels" element={authed(<Applications />)} />
                                    <Route path="/channels/:id" element={authed(<Messages />)} />
                                    <Route
                                        path="/messages/:id"
                                        element={authed(<Messages />)}
                                    />
                                    <Route
                                        path="/applications"
                                        element={<Navigate replace to="/channels" />}
                                    />
                                    <Route path="/clients" element={authed(<Clients />)} />
                                    <Route
                                        path="/users"
                                        element={authed(elevated(<Users />))}
                                    />
                                    <Route
                                        path="/settings"
                                        element={authed(
                                            <Settings
                                                themeMode={currentTheme}
                                                setTheme={setTheme}
                                            />
                                        )}
                                    />
                                    <Route path="/plugins" element={authed(<Plugins />)} />
                                    <Route
                                        path="/plugins/:id"
                                        element={authed(
                                            <Lazy
                                                component={() =>
                                                    import('../plugin/PluginDetailView')
                                                }
                                            />
                                        )}
                                    />
                                </Routes>
                            </Box>
                        </Box>

                        <ScrollUpButton />
                        <SnackbarProvider />
                    </div>
                </HashRouter>
            </ThemeProvider>
        </StyledEngineProvider>
    );
});

// eslint-disable-next-line
const Lazy = ({component}: {component: () => Promise<{default: React.ComponentType<any>}>}) => {
    const Component = React.lazy(component);

    return (
        <React.Suspense fallback={<LoadingSpinner />}>
            <Component />
        </React.Suspense>
    );
};

const RequireAuth: React.FC<
    React.PropsWithChildren<{loggedIn: boolean; authenticating: boolean}>
> = ({children, authenticating, loggedIn}) => {
    if (authenticating) {
        return <LoadingSpinner />;
    }
    if (!loggedIn) {
        return <Navigate replace={true} to="/login" />;
    }
    return <>{children}</>;
};

export const RequireElevation = observer(({children}: React.PropsWithChildren) => {
    const {elevateStore} = useStores();

    if (elevateStore.elevated) {
        return <>{children}</>;
    }

    return (
        <DefaultPage
            title="Authentication Required"
            description="Confirm your identity before accessing this administrative area."
            maxWidth={520}>
            <Paper variant="outlined" sx={{p: 2.5, borderRadius: 3}}>
                <ElevationForm />
            </Paper>
        </DefaultPage>
    );
});

export default Layout;
