import AppBar from '@mui/material/AppBar';
import Button, {ButtonProps} from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import {Theme} from '@mui/material/styles';
import {makeStyles} from 'tss-react/mui';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import AccountCircle from '@mui/icons-material/AccountCircle';
import Chat from '@mui/icons-material/Chat';
import DevicesOther from '@mui/icons-material/DevicesOther';
import ExitToApp from '@mui/icons-material/ExitToApp';
import MenuIcon from '@mui/icons-material/Menu';
import Apps from '@mui/icons-material/Apps';
import SupervisorAccount from '@mui/icons-material/SupervisorAccount';
import SettingsIcon from '@mui/icons-material/Settings';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import React, {CSSProperties} from 'react';
import {Link} from 'react-router';
import {useMediaQuery} from '@mui/material';
import * as config from '../config';

const useStyles = makeStyles()((theme: Theme) => ({
    appBar: {
        zIndex: theme.zIndex.drawer + 1,
        [theme.breakpoints.down('sm')]: {
            paddingBottom: 10,
        },
    },
    toolbar: {
        justifyContent: 'space-between',
        [theme.breakpoints.down('sm')]: {
            flexWrap: 'wrap',
        },
    },
    menuButtons: {
        display: 'flex',
        [theme.breakpoints.down('md')]: {
            flex: 1,
        },
        justifyContent: 'center',
        [theme.breakpoints.down('sm')]: {
            flexBasis: '100%',
            marginTop: 5,
            order: 1,
            height: 50,
            justifyContent: 'space-between',
            alignItems: 'center',
        },
    },
    title: {
        [theme.breakpoints.up('md')]: {
            flex: 1,
        },
        display: 'flex',
        alignItems: 'center',
    },
    titleName: {
        paddingRight: 10,
    },
    logo: {
        width: 54,
        height: 36,
        objectFit: 'cover',
        borderRadius: 4,
        marginRight: 10,
    },
    link: {
        color: 'inherit',
        textDecoration: 'none',
    },
}));

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
    const {classes} = useStyles();
    return (
        <AppBar
            sx={{position: {xs: 'sticky', sm: 'fixed'}}}
            style={style}
            className={classes.appBar}>
            <Toolbar className={classes.toolbar}>
                <div className={classes.title}>
                    <Link to="/" className={classes.link} style={{display: 'flex', alignItems: 'center'}}>
                        <img
                            src={config.get('url') + 'static/gotify-mu-logo.png'}
                            alt="Gotify MU"
                            className={classes.logo}
                        />
                        <Typography variant="h5" className={classes.titleName} color="inherit">
                            Gotify MU
                        </Typography>
                    </Link>
                    <a
                        href={
                            version.startsWith('master-')
                                ? `https://github.com/gigabytegrove/gotify-mu/commit/${version.replace('master-', '')}`
                                : 'https://github.com/gigabytegrove/gotify-mu/releases'
                        }
                        className={classes.link}>
                        <Typography variant="button" color="inherit">
                            @{version}
                        </Typography>
                    </a>
                </div>
                {loggedIn && (
                    <Buttons admin={admin} name={name} logout={logout} setNavOpen={setNavOpen} />
                )}
            </Toolbar>
        </AppBar>
    );
};

const Buttons = ({
    name,
    admin,
    logout,
    setNavOpen,
}: {
    name: string;
    admin: boolean;
    logout: VoidFunction;
    setNavOpen: (open: boolean) => void;
}) => {
    const {classes} = useStyles();
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const userDropDown = Boolean(anchorEl);

    return (
        <div className={classes.menuButtons}>
            <ResponsiveButton
                sx={{display: {sm: 'none', xs: 'block'}}}
                icon={<MenuIcon />}
                onClick={() => setNavOpen(true)}
                label="menu"
                color="inherit"
            />
            {admin && (
                <Link className={classes.link} to="/users" id="navigate-users">
                    <ResponsiveButton icon={<SupervisorAccount />} label="users" color="inherit" />
                </Link>
            )}
            <Link className={classes.link} to="/applications" id="navigate-apps">
                <ResponsiveButton icon={<Chat />} label="channels" color="inherit" />
            </Link>
            <Link className={classes.link} to="/clients" id="navigate-clients">
                <ResponsiveButton icon={<DevicesOther />} label="clients" color="inherit" />
            </Link>
            <Link className={classes.link} to="/plugins" id="navigate-plugins">
                <ResponsiveButton icon={<Apps />} label="plugins" color="inherit" />
            </Link>
            <ResponsiveButton
                icon={<AccountCircle />}
                label={name}
                onClick={(e) => setAnchorEl(e.currentTarget)}
                id="user-menu-button"
                aria-controls={userDropDown ? 'user-menu' : undefined}
                aria-haspopup="true"
                aria-expanded={userDropDown ? 'true' : undefined}
                color="inherit"
            />
            <Menu
                id="user-menu"
                anchorEl={anchorEl}
                open={userDropDown}
                onClose={() => setAnchorEl(null)}
                anchorOrigin={{vertical: 'bottom', horizontal: 'right'}}
                transformOrigin={{vertical: 'top', horizontal: 'right'}}>
                <MenuItem component={Link} to="/settings" onClick={() => setAnchorEl(null)}>
                    <ListItemIcon>
                        <SettingsIcon fontSize="small" />
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
                    <ListItemText>Logout</ListItemText>
                </MenuItem>
            </Menu>
        </div>
    );
};

const ResponsiveButton: React.FC<{
    color: 'inherit';
    sx?: ButtonProps['sx'];
    label: string;
    id?: string;
    onClick?: (event: React.MouseEvent<HTMLElement>) => void;
    icon: React.ReactNode;
}> = ({icon, label, ...rest}) => {
    const matches = useMediaQuery('(max-width:1000px)');
    if (matches) {
        return (
            <IconButton {...rest} size="large">
                {icon}
            </IconButton>
        );
    }
    return (
        <Button startIcon={icon} {...rest}>
            {label}
        </Button>
    );
};

export default Header;
