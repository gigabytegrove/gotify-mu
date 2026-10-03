import React, {CSSProperties} from 'react';
import {
    AppBar,
    Avatar,
    Box,
    Button,
    Chip,
    IconButton,
    ListItemIcon,
    ListItemText,
    Menu,
    MenuItem,
    Stack,
    Toolbar,
    Tooltip,
    Typography,
} from '@mui/material';
import AccountCircle from '@mui/icons-material/AccountCircle';
import ExitToApp from '@mui/icons-material/ExitToApp';
import MenuIcon from '@mui/icons-material/Menu';
import Settings from '@mui/icons-material/Settings';
import Security from '@mui/icons-material/Security';
import InstallDesktop from '@mui/icons-material/InstallDesktop';
import KeyboardArrowDown from '@mui/icons-material/KeyboardArrowDown';
import FiberManualRecord from '@mui/icons-material/FiberManualRecord';
import {Link, useLocation} from 'react-router';
import * as config from '../config';
import {navigationWidth} from './Navigation';

interface BeforeInstallPromptEvent extends Event {
    prompt: () => Promise<void>;
    userChoice: Promise<{outcome: 'accepted' | 'dismissed'; platform: string}>;
}

interface IProps {
    loggedIn: boolean;
    name: string;
    admin: boolean;
    version: string;
    logout: VoidFunction;
    style: CSSProperties;
    setNavOpen: (open: boolean) => void;
}

const routeTitle = (pathname: string): string => {
    if (pathname === '/') return 'Dashboard';
    if (pathname.startsWith('/messages')) return 'Messages';
    if (pathname.startsWith('/channels')) return 'Channels';
    if (pathname.startsWith('/users')) return 'Users';
    if (pathname.startsWith('/groups')) return 'Groups';
    if (pathname.startsWith('/integrations')) return 'Integrations';
    if (pathname.startsWith('/automation')) return 'Automation';
    if (pathname.startsWith('/system')) return 'Security & Operations';
    if (pathname.startsWith('/audit')) return 'Audit Log';
    if (pathname.startsWith('/clients')) return 'Clients';
    if (pathname.startsWith('/plugins')) return 'Plugins';
    if (pathname.startsWith('/settings')) return 'Settings';
    if (pathname.startsWith('/login')) return 'Sign in';
    return 'Workspace';
};

const Header = ({version, name, loggedIn, admin, logout, style, setNavOpen}: IProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const [installPrompt, setInstallPrompt] = React.useState<BeforeInstallPromptEvent | null>(null);
    const location = useLocation();

    React.useEffect(() => {
        const onInstallPrompt = (event: Event) => {
            event.preventDefault();
            setInstallPrompt(event as BeforeInstallPromptEvent);
        };
        const onInstalled = () => setInstallPrompt(null);
        window.addEventListener('beforeinstallprompt', onInstallPrompt);
        window.addEventListener('appinstalled', onInstalled);
        return () => {
            window.removeEventListener('beforeinstallprompt', onInstallPrompt);
            window.removeEventListener('appinstalled', onInstalled);
        };
    }, []);

    const installMonita = async () => {
        if (!installPrompt) return;
        await installPrompt.prompt();
        await installPrompt.userChoice;
        setInstallPrompt(null);
    };

    return (
        <AppBar
            position="sticky"
            elevation={0}
            color="inherit"
            style={style}
            sx={{
                zIndex: (theme) => theme.zIndex.drawer + 1,
                borderBottom: 1,
                borderColor: 'divider',
                bgcolor: 'background.paper',
            }}>
            <Toolbar
                disableGutters
                sx={{
                    minHeight: 68,
                    px: {xs: 1.25, sm: 0},
                }}>
                <Stack
                    direction="row"
                    spacing={1.25}
                    sx={{
                        width: {sm: navigationWidth},
                        height: 68,
                        flexShrink: 0,
                        alignItems: 'center',
                        px: {xs: 0, sm: 2},
                        borderRight: {sm: 1},
                        borderColor: {sm: 'divider'},
                    }}>
                    {loggedIn && (
                        <IconButton
                            sx={{display: {xs: 'inline-flex', sm: 'none'}}}
                            aria-label="Open navigation"
                            onClick={() => setNavOpen(true)}>
                            <MenuIcon />
                        </IconButton>
                    )}
                    <Box
                        component={Link}
                        to="/"
                        aria-label="Monita home"
                        sx={{
                            display: 'flex',
                            alignItems: 'center',
                            minWidth: 0,
                            color: 'inherit',
                            textDecoration: 'none',
                        }}>
                        <Box
                            component="img"
                            src={config.get('url') + 'static/monita-icon.svg?v=1.3.5'}
                            alt=""
                            aria-hidden="true"
                            sx={{width: 36, height: 36, objectFit: 'contain', flexShrink: 0}}
                        />
                        <Box sx={{ml: 1.15, minWidth: 0}}>
                            <Typography
                                sx={{
                                    fontWeight: 800,
                                    lineHeight: 1,
                                    fontSize: '1.08rem',
                                    letterSpacing: '-0.025em',
                                }}>
                                Monita
                            </Typography>
                            <Typography
                                variant="caption"
                                color="text.secondary"
                                sx={{
                                    display: {xs: 'none', sm: 'block'},
                                    mt: 0.35,
                                    lineHeight: 1,
                                }}>
                                Control Center
                            </Typography>
                        </Box>
                    </Box>
                </Stack>

                <Stack
                    direction="row"
                    spacing={1.25}
                    sx={{
                        flex: 1,
                        minWidth: 0,
                        px: {xs: 0, sm: 2.25},
                        alignItems: 'center',
                    }}>
                    <Box sx={{display: {xs: 'none', md: 'block'}, minWidth: 0}}>
                        <Typography
                            variant="caption"
                            color="text.secondary"
                            sx={{display: 'block', lineHeight: 1}}>
                            Workspace
                        </Typography>
                        <Typography sx={{fontWeight: 720, mt: 0.35}} noWrap>
                            {routeTitle(location.pathname)}
                        </Typography>
                    </Box>

                    <Box sx={{flex: 1}} />

                    {loggedIn && (
                        <Chip
                            size="small"
                            icon={
                                <FiberManualRecord
                                    sx={{fontSize: '9px !important', color: '#19B36B !important'}}
                                />
                            }
                            label="Connected"
                            variant="outlined"
                            sx={{display: {xs: 'none', lg: 'inline-flex'}}}
                        />
                    )}

                    {installPrompt && (
                        <Tooltip title="Install Monita">
                            <Button
                                size="small"
                                variant="outlined"
                                startIcon={<InstallDesktop />}
                                onClick={() => void installMonita()}
                                sx={{display: {xs: 'none', lg: 'inline-flex'}}}>
                                Install
                            </Button>
                        </Tooltip>
                    )}

                    <Tooltip title="Build version">
                        <Chip
                            component="a"
                            clickable
                            size="small"
                            variant="outlined"
                            label={`@${version}`}
                            href={
                                version.startsWith('master-')
                                    ? `https://github.com/gigabytegrove/monita/commit/${version.replace('master-', '')}`
                                    : 'https://github.com/gigabytegrove/monita/releases'
                            }
                            target="_blank"
                            rel="noreferrer"
                            sx={{display: {xs: 'none', sm: 'inline-flex'}}}
                        />
                    </Tooltip>

                    {loggedIn && (
                        <>
                            <Button
                                id="user-menu-button"
                                aria-label="Account menu"
                                onClick={(event) => setAnchorEl(event.currentTarget)}
                                endIcon={<KeyboardArrowDown fontSize="small" />}
                                sx={{
                                    minWidth: 0,
                                    px: 0.75,
                                    color: 'text.primary',
                                    gap: 0.5,
                                }}>
                                <Avatar
                                    sx={{
                                        width: 32,
                                        height: 32,
                                        fontSize: '0.82rem',
                                        bgcolor: 'primary.main',
                                        color: 'primary.contrastText',
                                    }}>
                                    {name.slice(0, 1).toUpperCase() || <AccountCircle />}
                                </Avatar>
                                <Typography
                                    variant="body2"
                                    sx={{
                                        display: {xs: 'none', md: 'block'},
                                        fontWeight: 680,
                                        maxWidth: 150,
                                    }}
                                    noWrap>
                                    {name}
                                </Typography>
                            </Button>
                            <Menu
                                id="user-menu"
                                anchorEl={anchorEl}
                                open={Boolean(anchorEl)}
                                onClose={() => setAnchorEl(null)}
                                anchorOrigin={{vertical: 'bottom', horizontal: 'right'}}
                                transformOrigin={{vertical: 'top', horizontal: 'right'}}>
                                <Box sx={{px: 2, py: 1.25}}>
                                    <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                        <Typography sx={{fontWeight: 720}}>{name}</Typography>
                                        {admin && (
                                            <Chip
                                                icon={<Security fontSize="small" />}
                                                label="Admin"
                                                size="small"
                                            />
                                        )}
                                    </Stack>
                                </Box>
                                <MenuItem
                                    component={Link}
                                    to="/settings"
                                    onClick={() => setAnchorEl(null)}>
                                    <ListItemIcon>
                                        <Settings fontSize="small" />
                                    </ListItemIcon>
                                    <ListItemText>Settings</ListItemText>
                                </MenuItem>
                                <MenuItem
                                    id="logout"
                                    onClick={() => {
                                        setAnchorEl(null);
                                        logout();
                                    }}>
                                    <ListItemIcon>
                                        <ExitToApp fontSize="small" />
                                    </ListItemIcon>
                                    <ListItemText>Sign out</ListItemText>
                                </MenuItem>
                            </Menu>
                        </>
                    )}
                </Stack>
            </Toolbar>
        </AppBar>
    );
};

export default Header;
