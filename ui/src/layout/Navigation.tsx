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
import MonitorHeart from '@mui/icons-material/MonitorHeart';
import Public from '@mui/icons-material/Public';
import NotificationsOff from '@mui/icons-material/NotificationsOff';
import {Link, useLocation} from 'react-router';
import {observer} from 'mobx-react-lite';
import {mayAllowPermission, requestPermission} from '../snack/browserNotification';
import {useStores} from '../stores';
import * as config from '../config';

export const navigationWidth = 276;

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
    const [showRequestNotification, setShowRequestNotification] =
        React.useState(mayAllowPermission);

    const items: NavItem[] = [
        {label: 'Dashboard', to: '/', icon: <Dashboard />, exact: true},
        {label: 'Messages', to: '/messages', icon: <Inbox />},
        {label: 'Channels', to: '/channels', icon: <Forum />},
        {label: 'Users', to: '/users', icon: <People />, adminOnly: true},
        {label: 'Groups', to: '/groups', icon: <GroupWork />, adminOnly: true},
        {label: 'Integrations', to: '/integrations', icon: <Hub />, adminOnly: true},
        {label: 'Automation', to: '/automation', icon: <AutoMode />, adminOnly: true},
        {label: 'Operations', to: '/operations', icon: <MonitorHeart />, adminOnly: true},
        {label: 'Audit Log', to: '/audit', icon: <FactCheck />, adminOnly: true},
        {label: 'Clients', to: '/clients', icon: <DevicesOther />},
        {label: 'Plugins', to: '/plugins', icon: <Extension />},
        {label: 'Settings', to: '/settings', icon: <Settings />},
    ];

    const selected = (item: NavItem) =>
        item.exact ? location.pathname === item.to : location.pathname.startsWith(item.to);

    const drawerContent = (
        <Box sx={{height: '100%', display: 'flex', flexDirection: 'column'}}>
            <Box sx={{display: {xs: 'flex', sm: 'none'}, justifyContent: 'flex-end', p: 1}}>
                <IconButton aria-label="Close navigation" onClick={() => setNavOpen(false)}>
                    <Close />
                </IconButton>
            </Box>

            <Box sx={{px: 1.5, py: 2}}>
                <Typography
                    variant="overline"
                    color="text.secondary"
                    sx={{px: 1.5, letterSpacing: 1}}>
                    Workspace
                </Typography>
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
                                    borderRadius: 1.75,
                                    my: 0.15,
                                    py: 0.7,
                                    '&.Mui-selected': {
                                        bgcolor: 'action.selected',
                                        '& .MuiListItemIcon-root': {color: 'primary.main'},
                                        '& .MuiListItemText-primary': {fontWeight: 700},
                                    },
                                }}>
                                <ListItemIcon sx={{minWidth: 40}}>{item.icon}</ListItemIcon>
                                <ListItemText primary={item.label} />
                            </ListItemButton>
                        ))}
                </List>
            </Box>

            <Divider />

            <Box sx={{px: 1.5, py: 1.5, flex: 1, minHeight: 0, overflowY: 'auto'}}>
                <Stack
                    direction="row"
                    sx={{px: 1.25, mb: 0.5, alignItems: 'center', justifyContent: 'space-between'}}>
                    <Typography
                        variant="overline"
                        color="text.secondary"
                        sx={{letterSpacing: 1}}>
                        Your Channels
                    </Typography>
                    <Chip size="small" variant="outlined" label={apps.length} />
                </Stack>
                <List disablePadding>
                    {loggedIn && apps.length === 0 && (
                        <ListItemButton disabled sx={{borderRadius: 2}}>
                            <ListItemText
                                primary="No channels"
                                secondary="Create or join a Channel to see it here."
                            />
                        </ListItemButton>
                    )}
                    {loggedIn &&
                        apps.map((app) => {
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
                                        borderRadius: 1.75,
                                        my: 0.15,
                                        py: 0.55,
                                        '&.Mui-selected': {
                                            bgcolor: 'action.selected',
                                        },
                                    }}>
                                    <ListItemAvatar sx={{minWidth: 42}}>
                                        <Avatar
                                            src={config.get('url') + app.image}
                                            variant="rounded"
                                            sx={{width: 30, height: 30}}
                                        />
                                    </ListItemAvatar>
                                    <ListItemText
                                        primary={<Typography noWrap>{app.name}</Typography>}
                                    />
                                    <Stack direction="row" spacing={0.5} sx={{alignItems: 'center'}}>
                                        {app.receiveNotifications === false && (
                                            <NotificationsOff
                                                sx={{fontSize: 15, color: 'text.disabled'}}
                                            />
                                        )}
                                        {app.autoAssign && (
                                            <Public
                                                sx={{fontSize: 15, color: 'text.secondary'}}
                                            />
                                        )}
                                    </Stack>
                                </ListItemButton>
                            );
                        })}
                </List>
            </Box>

            {showRequestNotification && (
                <>
                    <Divider />
                    <Stack sx={{p: 1.5}}> 
                        <Button
                            variant="outlined"
                            onClick={() => {
                                requestPermission();
                                setShowRequestNotification(false);
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
                    '& .MuiDrawer-paper': {width: navigationWidth},
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
                        borderRightStyle: 'solid',
                    },
                }}>
                {drawerContent}
            </Drawer>
        </>
    );
});

export default Navigation;
