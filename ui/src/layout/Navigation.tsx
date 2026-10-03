import React from 'react';
import {
    Avatar,
    Box,
    Button,
    Chip,
    Divider,
    Drawer,
    IconButton,
    List,
    ListItemAvatar,
    ListItemButton,
    ListItemIcon,
    ListItemText,
    Stack,
    Typography,
} from '@mui/material';
import Close from '@mui/icons-material/Close';
import Dashboard from '@mui/icons-material/Dashboard';
import Inbox from '@mui/icons-material/Inbox';
import Forum from '@mui/icons-material/Forum';
import People from '@mui/icons-material/People';
import GroupWork from '@mui/icons-material/GroupWork';
import FactCheck from '@mui/icons-material/FactCheck';
import DevicesOther from '@mui/icons-material/DevicesOther';
import Extension from '@mui/icons-material/Extension';
import Settings from '@mui/icons-material/Settings';
import Hub from '@mui/icons-material/Hub';
import AutoMode from '@mui/icons-material/AutoMode';
import AdminPanelSettings from '@mui/icons-material/AdminPanelSettings';
import Public from '@mui/icons-material/Public';
import NotificationsOff from '@mui/icons-material/NotificationsOff';
import NotificationsActive from '@mui/icons-material/NotificationsActive';
import {Link, useLocation} from 'react-router';
import {observer} from 'mobx-react-lite';
import {mayAllowPermission, requestPermission} from '../snack/browserNotification';
import {useStores} from '../stores';
import * as config from '../config';

export const navigationWidth = 288;

interface IProps {
    loggedIn: boolean;
    navOpen: boolean;
    setNavOpen: (open: boolean) => void;
}

interface NavItem {
    label: string;
    to: string;
    icon: React.ReactNode;
    exact?: boolean;
    adminOnly?: boolean;
}

const Navigation = observer(({loggedIn, navOpen, setNavOpen}: IProps) => {
    const location = useLocation();
    const {appStore, currentUser} = useStores();
    const apps = appStore.getItems();
    const chatApps = apps.filter(
        (app) =>
            app.channelType === 'chat' || (app.channelType == null && Boolean(app.allowMemberPost))
    );
    const notificationApps = apps.filter(
        (app) =>
            !(
                app.channelType === 'chat' ||
                (app.channelType == null && Boolean(app.allowMemberPost))
            )
    );
    const [showRequestNotification, setShowRequestNotification] =
        React.useState(mayAllowPermission);

    const items: NavItem[] = [
        {label: 'Overview', to: '/', icon: <Dashboard />, exact: true},
        {label: 'Messages', to: '/messages', icon: <Inbox />},
        {label: 'Channels', to: '/channels', icon: <Forum />},
        {label: 'Users', to: '/users', icon: <People />, adminOnly: true},
        {label: 'Groups', to: '/groups', icon: <GroupWork />, adminOnly: true},
        {label: 'Integrations', to: '/integrations', icon: <Hub />, adminOnly: true},
        {label: 'Automation', to: '/automation', icon: <AutoMode />, adminOnly: true},
        {
            label: 'Security & Operations',
            to: '/system',
            icon: <AdminPanelSettings />,
            adminOnly: true,
        },
        {label: 'Audit Log', to: '/audit', icon: <FactCheck />, adminOnly: true},
        {label: 'Clients', to: '/clients', icon: <DevicesOther />},
        {label: 'Plugins', to: '/plugins', icon: <Extension />},
        {label: 'Settings', to: '/settings', icon: <Settings />},
    ];

    const selected = (item: NavItem) =>
        item.exact ? location.pathname === item.to : location.pathname.startsWith(item.to);

    const renderChannelSection = (
        label: string,
        sectionApps: typeof apps,
        icon: React.ReactNode
    ) => (
        <Box sx={{mb: 1.7}}>
            <Stack
                direction="row"
                sx={{px: 1, mb: 0.65, alignItems: 'center', justifyContent: 'space-between'}}>
                <Stack direction="row" spacing={0.7} sx={{alignItems: 'center'}}>
                    <Box sx={{display: 'flex', color: 'rgba(219,231,245,.64)'}}>{icon}</Box>
                    <Typography
                        variant="overline"
                        sx={{
                            color: 'rgba(219,231,245,.58)',
                            fontSize: '0.64rem',
                            letterSpacing: '.12em',
                        }}>
                        {label}
                    </Typography>
                </Stack>
                <Chip
                    size="small"
                    label={sectionApps.length}
                    sx={{
                        color: '#cbd9ea',
                        borderColor: 'rgba(255,255,255,.12)',
                        bgcolor: 'rgba(255,255,255,.04)',
                    }}
                    variant="outlined"
                />
            </Stack>
            <List disablePadding>
                {loggedIn && sectionApps.length === 0 && (
                    <ListItemButton
                        disabled
                        sx={{
                            borderRadius: 2,
                            color: 'rgba(219,231,245,.45)',
                            '&.Mui-disabled': {opacity: 0.6},
                        }}>
                        <ListItemText primary={`No ${label.toLowerCase()}`} />
                    </ListItemButton>
                )}
                {loggedIn &&
                    sectionApps.map((app) => {
                        const to = `/channels/${app.id}`;
                        return (
                            <ListItemButton
                                key={app.id}
                                className="item channel-shortcut"
                                component={Link}
                                to={to}
                                selected={location.pathname === to}
                                onClick={() => setNavOpen(false)}
                                sx={{
                                    borderRadius: 2.2,
                                    my: 0.18,
                                    py: 0.55,
                                    px: 0.8,
                                    color: '#dbe7f5',
                                    '&:hover': {bgcolor: 'rgba(255,255,255,.055)'},
                                    '&.Mui-selected': {
                                        bgcolor: 'rgba(37,99,235,.24)',
                                        boxShadow: 'inset 3px 0 0 #60A5FA',
                                    },
                                    '&.Mui-selected:hover': {bgcolor: 'rgba(37,99,235,.30)'},
                                }}>
                                <ListItemAvatar sx={{minWidth: 40}}>
                                    <Avatar
                                        src={config.get('url') + app.image}
                                        variant="rounded"
                                        sx={{
                                            width: 31,
                                            height: 31,
                                            bgcolor: 'rgba(255,255,255,.07)',
                                            border: '1px solid rgba(255,255,255,.08)',
                                        }}
                                    />
                                </ListItemAvatar>
                                <ListItemText
                                    primary={
                                        <Typography noWrap sx={{fontSize: '0.84rem', fontWeight: 650}}>
                                            {app.name}
                                        </Typography>
                                    }
                                    secondary={
                                        app.receiveNotifications === false
                                            ? 'Notifications muted'
                                            : undefined
                                    }
                                    slotProps={{
                                        secondary: {
                                            noWrap: true,
                                            sx: {color: 'rgba(219,231,245,.48)', fontSize: '0.71rem'},
                                        },
                                    }}
                                />
                                <Stack direction="row" spacing={0.45} sx={{alignItems: 'center'}}>
                                    {app.receiveNotifications === false && (
                                        <NotificationsOff sx={{fontSize: 14, color: '#75869c'}} />
                                    )}
                                    {app.autoAssign && (
                                        <Public sx={{fontSize: 14, color: '#75869c'}} />
                                    )}
                                </Stack>
                            </ListItemButton>
                        );
                    })}
            </List>
        </Box>
    );

    const drawerContent = (
        <Box
            sx={{
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                bgcolor: '#0B1220',
                color: '#dbe7f5',
            }}>
            <Box
                sx={{
                    display: {xs: 'flex', sm: 'none'},
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    px: 1.25,
                    py: 1.1,
                }}>
                <Typography sx={{fontWeight: 800}}>Navigation</Typography>
                <IconButton
                    aria-label="Close navigation"
                    onClick={() => setNavOpen(false)}
                    sx={{color: '#dbe7f5'}}>
                    <Close />
                </IconButton>
            </Box>

            <Box sx={{px: 1.4, pt: {xs: 0.5, sm: 1.8}, pb: 1.2}}>
                <Typography
                    variant="overline"
                    sx={{
                        px: 1,
                        color: 'rgba(219,231,245,.48)',
                        fontSize: '0.63rem',
                        letterSpacing: '.12em',
                    }}>
                    Workspace
                </Typography>
                <List disablePadding sx={{mt: 0.45}}>
                    {items
                        .filter((item) => !item.adminOnly || currentUser.user.admin)
                        .map((item) => (
                            <ListItemButton
                                key={item.to}
                                id={
                                    item.to === '/channels'
                                        ? 'navigate-apps'
                                        : item.to === '/users'
                                          ? 'navigate-users'
                                          : item.to === '/clients'
                                            ? 'navigate-clients'
                                            : item.to === '/plugins'
                                              ? 'navigate-plugins'
                                              : item.to === '/messages'
                                                ? 'navigate-messages'
                                                : undefined
                                }
                                className={item.to === '/messages' ? 'all' : undefined}
                                component={Link}
                                to={item.to}
                                selected={selected(item)}
                                disabled={!loggedIn}
                                onClick={() => setNavOpen(false)}
                                sx={{
                                    borderRadius: 2.2,
                                    my: 0.18,
                                    py: 0.72,
                                    px: 1,
                                    color: '#dbe7f5',
                                    '&:hover': {bgcolor: 'rgba(255,255,255,.055)'},
                                    '&.Mui-selected': {
                                        bgcolor: 'rgba(37,99,235,.24)',
                                        color: '#fff',
                                        boxShadow: 'inset 3px 0 0 #60A5FA',
                                        '& .MuiListItemIcon-root': {color: '#7DB3FF'},
                                        '& .MuiListItemText-primary': {fontWeight: 760},
                                    },
                                    '&.Mui-selected:hover': {bgcolor: 'rgba(37,99,235,.30)'},
                                }}>
                                <ListItemIcon sx={{minWidth: 38, color: '#8fa2b9'}}>
                                    {item.icon}
                                </ListItemIcon>
                                <ListItemText
                                    primary={item.label}
                                    slotProps={{primary: {sx: {fontSize: '0.87rem'}}}}
                                />
                            </ListItemButton>
                        ))}
                </List>
            </Box>

            <Divider sx={{borderColor: 'rgba(255,255,255,.075)'}} />

            <Box sx={{px: 1.4, py: 1.45, flex: 1, minHeight: 0, overflowY: 'auto'}}>
                <Stack
                    direction="row"
                    sx={{px: 1, mb: 1, alignItems: 'center', justifyContent: 'space-between'}}>
                    <Typography
                        variant="overline"
                        sx={{
                            color: 'rgba(219,231,245,.48)',
                            fontSize: '0.63rem',
                            letterSpacing: '.12em',
                        }}>
                        Live channels
                    </Typography>
                    <Chip
                        size="small"
                        label={apps.length}
                        sx={{
                            color: '#bfdbfe',
                            bgcolor: 'rgba(37,99,235,.18)',
                            borderColor: 'rgba(96,165,250,.28)',
                        }}
                        variant="outlined"
                    />
                </Stack>
                {renderChannelSection('Chats', chatApps, <Forum sx={{fontSize: 15}} />)}
                {renderChannelSection(
                    'Notifications',
                    notificationApps,
                    <NotificationsActive sx={{fontSize: 15}} />
                )}
            </Box>

            {showRequestNotification && (
                <>
                    <Divider sx={{borderColor: 'rgba(255,255,255,.075)'}} />
                    <Stack sx={{p: 1.4}}>
                        <Button
                            variant="outlined"
                            onClick={() => {
                                requestPermission();
                                setShowRequestNotification(false);
                            }}
                            sx={{
                                color: '#dbe7f5',
                                borderColor: 'rgba(255,255,255,.14)',
                                bgcolor: 'rgba(255,255,255,.025)',
                                '&:hover': {
                                    borderColor: '#60A5FA',
                                    bgcolor: 'rgba(37,99,235,.14)',
                                },
                            }}>
                            Enable Browser Notifications
                        </Button>
                    </Stack>
                </>
            )}
        </Box>
    );

    return (
        <>
            <Drawer
                open={navOpen}
                onClose={() => setNavOpen(false)}
                variant="temporary"
                sx={{
                    display: {xs: 'block', sm: 'none'},
                    '& .MuiDrawer-paper': {
                        width: navigationWidth,
                        borderRight: 0,
                    },
                }}>
                {drawerContent}
            </Drawer>
            <Drawer
                id="message-navigation"
                variant="permanent"
                open
                sx={{
                    display: {xs: 'none', sm: 'block'},
                    width: navigationWidth,
                    flexShrink: 0,
                    '& .MuiDrawer-paper': {
                        width: navigationWidth,
                        boxSizing: 'border-box',
                        position: 'relative',
                        height: '100%',
                        borderRight: '1px solid #19263a',
                    },
                }}>
                {drawerContent}
            </Drawer>
        </>
    );
});

export default Navigation;
