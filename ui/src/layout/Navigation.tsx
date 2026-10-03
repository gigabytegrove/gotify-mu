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

    const coreItems: NavItem[] = [
        {label: 'Dashboard', to: '/', icon: <Dashboard />, exact: true},
        {label: 'Messages', to: '/messages', icon: <Inbox />},
        {label: 'Channels', to: '/channels', icon: <Forum />},
    ];
    const adminItems: NavItem[] = [
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
    ];
    const toolItems: NavItem[] = [
        {label: 'Clients', to: '/clients', icon: <DevicesOther />},
        {label: 'Plugins', to: '/plugins', icon: <Extension />},
        {label: 'Settings', to: '/settings', icon: <Settings />},
    ];

    const selected = (item: NavItem) =>
        item.exact ? location.pathname === item.to : location.pathname.startsWith(item.to);

    const sectionLabel = (label: string) => (
        <Typography
            variant="overline"
            sx={{
                display: 'block',
                px: 1.5,
                mb: 0.65,
                color: 'rgba(226,235,247,.44)',
                fontSize: '0.64rem',
            }}>
            {label}
        </Typography>
    );

    const renderItems = (items: NavItem[]) => (
        <List disablePadding>
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
                            position: 'relative',
                            minHeight: 40,
                            borderRadius: 2,
                            my: 0.25,
                            px: 1.25,
                            color: 'rgba(231,238,248,.78)',
                            '& .MuiListItemIcon-root': {
                                color: 'rgba(182,197,217,.66)',
                            },
                            '&:hover': {
                                bgcolor: 'rgba(255,255,255,.055)',
                                color: '#FFFFFF',
                            },
                            '&.Mui-selected': {
                                bgcolor: 'rgba(110,140,255,.15)',
                                color: '#FFFFFF',
                                '&:before': {
                                    content: '""',
                                    position: 'absolute',
                                    left: 0,
                                    top: 9,
                                    bottom: 9,
                                    width: 3,
                                    borderRadius: 4,
                                    bgcolor: '#6E8CFF',
                                },
                                '& .MuiListItemIcon-root': {color: '#8DA5FF'},
                                '& .MuiListItemText-primary': {fontWeight: 720},
                                '&:hover': {bgcolor: 'rgba(110,140,255,.18)'},
                            },
                        }}>
                        <ListItemIcon sx={{minWidth: 38, '& svg': {fontSize: 20}}}>
                            {item.icon}
                        </ListItemIcon>
                        <ListItemText
                            primary={item.label}
                            slotProps={{primary: {fontSize: '0.86rem'}}}
                        />
                    </ListItemButton>
                ))}
        </List>
    );

    const renderChannelSection = (
        label: string,
        sectionApps: typeof apps,
        icon: React.ReactNode
    ) => (
        <Box sx={{mb: 1.6}}>
            <Stack
                direction="row"
                sx={{px: 1.35, mb: 0.55, alignItems: 'center', justifyContent: 'space-between'}}>
                <Stack direction="row" spacing={0.7} sx={{alignItems: 'center'}}>
                    <Box sx={{display: 'flex', color: 'rgba(182,197,217,.58)'}}>{icon}</Box>
                    <Typography
                        variant="overline"
                        sx={{color: 'rgba(226,235,247,.44)', fontSize: '0.62rem'}}>
                        {label}
                    </Typography>
                </Stack>
                <Chip
                    size="small"
                    variant="outlined"
                    label={sectionApps.length}
                    sx={{
                        color: 'rgba(231,238,248,.62)',
                        borderColor: 'rgba(255,255,255,.1)',
                        bgcolor: 'rgba(255,255,255,.025)',
                    }}
                />
            </Stack>
            <List disablePadding>
                {loggedIn && sectionApps.length === 0 && (
                    <ListItemButton disabled sx={{borderRadius: 2, color: 'rgba(255,255,255,.35)'}}>
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
                                    minHeight: 40,
                                    borderRadius: 2,
                                    my: 0.2,
                                    px: 1.15,
                                    color: 'rgba(231,238,248,.78)',
                                    '&:hover': {bgcolor: 'rgba(255,255,255,.055)'},
                                    '&.Mui-selected': {
                                        bgcolor: 'rgba(255,255,255,.075)',
                                        color: '#FFFFFF',
                                    },
                                }}>
                                <ListItemAvatar sx={{minWidth: 39}}>
                                    <Avatar
                                        src={config.get('url') + app.image}
                                        variant="rounded"
                                        sx={{width: 29, height: 29, borderRadius: 1.5}}
                                    />
                                </ListItemAvatar>
                                <ListItemText
                                    primary={<Typography noWrap fontSize="0.84rem">{app.name}</Typography>}
                                    secondary={
                                        app.receiveNotifications === false
                                            ? 'Muted'
                                            : undefined
                                    }
                                    slotProps={{
                                        secondary: {
                                            noWrap: true,
                                            sx: {color: 'rgba(182,197,217,.48)', fontSize: '0.69rem'},
                                        },
                                    }}
                                />
                                <Stack direction="row" spacing={0.4} sx={{alignItems: 'center'}}>
                                    {app.receiveNotifications === false && (
                                        <NotificationsOff
                                            sx={{fontSize: 14, color: 'rgba(182,197,217,.42)'}}
                                        />
                                    )}
                                    {app.autoAssign && (
                                        <Public
                                            sx={{fontSize: 14, color: 'rgba(182,197,217,.5)'}}
                                        />
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
                bgcolor: '#0A1220',
                color: '#FFFFFF',
            }}>
            <Box sx={{display: {xs: 'flex', sm: 'none'}, justifyContent: 'flex-end', p: 1}}>
                <IconButton
                    aria-label="Close navigation"
                    onClick={() => setNavOpen(false)}
                    sx={{color: 'rgba(255,255,255,.72)'}}>
                    <Close />
                </IconButton>
            </Box>

            <Box sx={{px: 1.5, pt: {xs: 0.5, sm: 2}, pb: 1}}>
                {sectionLabel('Workspace')}
                {renderItems(coreItems)}
            </Box>

            <Divider sx={{borderColor: 'rgba(255,255,255,.07)', mx: 1.5}} />

            <Box sx={{px: 1.5, py: 1.5, flex: 1, minHeight: 120, overflowY: 'auto'}}>
                <Stack
                    direction="row"
                    sx={{px: 1.35, mb: 1, alignItems: 'center', justifyContent: 'space-between'}}>
                    <Typography
                        variant="overline"
                        sx={{color: 'rgba(226,235,247,.44)', fontSize: '0.64rem'}}>
                        Conversations
                    </Typography>
                    <Chip
                        size="small"
                        label={apps.length}
                        sx={{
                            color: '#AFC0D7',
                            bgcolor: 'rgba(255,255,255,.045)',
                            border: '1px solid rgba(255,255,255,.08)',
                        }}
                    />
                </Stack>
                {renderChannelSection('Chats', chatApps, <Forum sx={{fontSize: 15}} />)}
                {renderChannelSection(
                    'Notifications',
                    notificationApps,
                    <NotificationsActive sx={{fontSize: 15}} />
                )}
            </Box>

            {currentUser.user.admin && (
                <>
                    <Divider sx={{borderColor: 'rgba(255,255,255,.07)', mx: 1.5}} />
                    <Box sx={{px: 1.5, py: 1.25}}>
                        {sectionLabel('Administration')}
                        {renderItems(adminItems)}
                    </Box>
                </>
            )}

            <Divider sx={{borderColor: 'rgba(255,255,255,.07)', mx: 1.5}} />

            <Box sx={{px: 1.5, py: 1.25}}>
                {sectionLabel('System')}
                {renderItems(toolItems)}
            </Box>

            {showRequestNotification && (
                <Box sx={{p: 1.5, pt: 0.25}}>
                    <Button
                        fullWidth
                        variant="outlined"
                        onClick={() => {
                            requestPermission();
                            setShowRequestNotification(false);
                        }}
                        sx={{
                            color: '#DCE6F4',
                            borderColor: 'rgba(255,255,255,.14)',
                            '&:hover': {
                                borderColor: 'rgba(255,255,255,.24)',
                                bgcolor: 'rgba(255,255,255,.05)',
                            },
                        }}>
                        Enable Notifications
                    </Button>
                </Box>
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
                    '& .MuiDrawer-paper': {width: navigationWidth, border: 0},
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
                        border: 0,
                    },
                }}>
                {drawerContent}
            </Drawer>
        </>
    );
});

export default Navigation;
