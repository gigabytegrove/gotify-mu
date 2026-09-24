import React, {CSSProperties} from 'react';
import {
    AppBar,
    Avatar,
    Box,
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
import {Link} from 'react-router';
import * as config from '../config';

interface IProps {
    loggedIn: boolean;
    name: string;
    admin: boolean;
    version: string;
    logout: VoidFunction;
    style: CSSProperties;
    setNavOpen: (open: boolean) => void;
}

const Header = ({version, name, loggedIn, admin, logout, style, setNavOpen}: IProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);

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
                backgroundColor: 'background.paper',
            }}>
            <Toolbar sx={{minHeight: 64, gap: 1.5}}>
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
                    sx={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 1.25,
                        minWidth: 0,
                        color: 'inherit',
                        textDecoration: 'none',
                    }}>
                    <Box
                        component="img"
                        src={config.get('url') + 'static/gotify-mu-logo.png'}
                        alt="Gotify MU"
                        sx={{width: 44, height: 32, objectFit: 'contain', borderRadius: 1}}
                    />
                    <Box sx={{display: {xs: 'none', sm: 'block'}}}>
                        <Typography variant="h6" sx={{lineHeight: 1.1}}>
                            Gotify MU
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                            Multi-user notifications
                        </Typography>
                    </Box>
                </Box>

                <Box sx={{flex: 1}} />

                <Tooltip title="Build version">
                    <Chip
                        component="a"
                        clickable
                        size="small"
                        variant="outlined"
                        label={`@${version}`}
                        href={
                            version.startsWith('master-')
                                ? `https://github.com/gigabytegrove/gotify-mu/commit/${version.replace('master-', '')}`
                                : 'https://github.com/gigabytegrove/gotify-mu/releases'
                        }
                        target="_blank"
                        rel="noreferrer"
                    />
                </Tooltip>

                {loggedIn && (
                    <>
                        <IconButton
                            id="user-menu-button"
                            aria-label="Account menu"
                            onClick={(event) => setAnchorEl(event.currentTarget)}>
                            <Avatar sx={{width: 34, height: 34}}>
                                {name.slice(0, 1).toUpperCase() || <AccountCircle />}
                            </Avatar>
                        </IconButton>
                        <Menu
                            id="user-menu"
                            anchorEl={anchorEl}
                            open={Boolean(anchorEl)}
                            onClose={() => setAnchorEl(null)}
                            anchorOrigin={{vertical: 'bottom', horizontal: 'right'}}
                            transformOrigin={{vertical: 'top', horizontal: 'right'}}>
                            <Box sx={{px: 2, py: 1.25}}>
                                <Stack direction="row" spacing={1} alignItems="center">
                                    <Typography fontWeight={700}>{name}</Typography>
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
            </Toolbar>
        </AppBar>
    );
};

export default Header;
