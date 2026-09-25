import React from 'react';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import Divider from '@mui/material/Divider';
import Box from '@mui/material/Box';
import Avatar from '@mui/material/Avatar';
import NotificationsActive from '@mui/icons-material/NotificationsActive';
import NotificationsOff from '@mui/icons-material/NotificationsOff';
import Public from '@mui/icons-material/Public';
import People from '@mui/icons-material/People';
import DevicesOther from '@mui/icons-material/DevicesOther';
import Extension from '@mui/icons-material/Extension';
import Security from '@mui/icons-material/Security';
import ArrowForward from '@mui/icons-material/ArrowForward';
import Inbox from '@mui/icons-material/Inbox';
import Settings from '@mui/icons-material/Settings';
import Forum from '@mui/icons-material/Forum';
import {Link} from 'react-router';
import {observer} from 'mobx-react-lite';
import DefaultPage from '../common/DefaultPage';
import StatCard from '../common/StatCard';
import SurfaceCard from '../common/SurfaceCard';
import {useStores} from '../stores';
import * as config from '../config';

const Dashboard = observer(() => {
    const {appStore, userStore, clientStore, pluginStore, currentUser} = useStores();
    const admin = currentUser.user.admin;

    React.useEffect(() => {
        void appStore.refresh();
        void clientStore.refresh();
        void pluginStore.refresh();
        if (admin) void userStore.refresh();
    }, [admin, appStore, clientStore, pluginStore, userStore]);

    const apps = appStore.getItems();
    const globals = apps.filter((app) => app.autoAssign).length;
    const muted = apps.filter((app) => app.receiveNotifications === false).length;
    const clients = clientStore.getItems();
    const plugins = pluginStore.getItems();
    const enabledPlugins = plugins.filter((plugin) => plugin.enabled).length;
    const users = admin ? userStore.getItems() : [];
    const version = config.get('version');

    const recentApps = [...apps]
        .sort((a, b) => {
            if (!a.lastUsed && !b.lastUsed) return 0;
            if (!a.lastUsed) return 1;
            if (!b.lastUsed) return -1;
            return Date.parse(b.lastUsed) - Date.parse(a.lastUsed);
        })
        .slice(0, 6);

    return (
        <DefaultPage
            title="Dashboard"
            description="Server health, access, and notification activity at a glance."
            rightControl={
                <Button
                    component={Link}
                    to="/messages"
                    variant="contained"
                    startIcon={<Inbox />}>
                    Open Messages
                </Button>
            }>
            <Grid container spacing={1.5}>
                <Grid size={{xs: 12, sm: 6, lg: 3}}>
                    <StatCard
                        label="Channels"
                        value={apps.length}
                        helper={`${globals} global · ${muted} muted`}
                        icon={<NotificationsActive />}
                    />
                </Grid>
                {admin && (
                    <Grid size={{xs: 12, sm: 6, lg: 3}}>
                        <StatCard
                            label="Users"
                            value={users.length}
                            helper="Local accounts"
                            icon={<People />}
                        />
                    </Grid>
                )}
                <Grid size={{xs: 12, sm: 6, lg: 3}}>
                    <StatCard
                        label="Clients"
                        value={clients.length}
                        helper="Authorized credentials"
                        icon={<DevicesOther />}
                    />
                </Grid>
                <Grid size={{xs: 12, sm: 6, lg: 3}}>
                    <StatCard
                        label="Plugins"
                        value={plugins.length}
                        helper={`${enabledPlugins} enabled`}
                        icon={<Extension />}
                    />
                </Grid>
            </Grid>

            <SurfaceCard title="Quick Actions" subtitle="Jump straight into common server tasks.">
                <Stack direction="row" spacing={1} useFlexGap sx={{flexWrap: 'wrap'}}>
                    <Button
                        component={Link}
                        to="/channels"
                        variant="outlined"
                        startIcon={<Forum />}>
                        Manage Channels
                    </Button>
                    <Button
                        component={Link}
                        to="/messages"
                        variant="outlined"
                        startIcon={<Inbox />}>
                        View Messages
                    </Button>
                    {admin && (
                        <Button
                            component={Link}
                            to="/users"
                            variant="outlined"
                            startIcon={<People />}>
                            Manage Users
                        </Button>
                    )}
                    <Button
                        component={Link}
                        to="/settings"
                        variant="outlined"
                        startIcon={<Settings />}>
                        Settings
                    </Button>
                </Stack>
            </SurfaceCard>

            <Grid container spacing={1.5}>
                <Grid size={{xs: 12, lg: 7}}>
                    <SurfaceCard
                        title="Recent Channels"
                        subtitle="Channels ordered by their most recent activity."
                        action={
                            <Button component={Link} to="/channels" size="small">
                                View all
                            </Button>
                        }>
                        <Stack spacing={0.5}>
                            {recentApps.length === 0 && (
                                <Typography color="text.secondary" sx={{py: 2}}>
                                    No Channels are available yet.
                                </Typography>
                            )}
                            {recentApps.map((app) => (
                                <Stack
                                    key={app.id}
                                    direction="row"
                                    spacing={1.25}
                                    sx={{
                                        alignItems: 'center',
                                        py: 0.8,
                                        px: 0.75,
                                        borderRadius: 1.5,
                                        '&:hover': {bgcolor: 'action.hover'},
                                    }}>
                                    <Avatar
                                        src={config.get('url') + app.image}
                                        variant="rounded"
                                        sx={{width: 36, height: 36}}
                                    />
                                    <Box sx={{minWidth: 0, flex: 1}}>
                                        <Stack
                                            direction="row"
                                            spacing={0.5}
                                            useFlexGap
                                            sx={{alignItems: 'center', flexWrap: 'wrap'}}>
                                            <Typography sx={{fontWeight: 700}} noWrap>
                                                {app.name}
                                            </Typography>
                                            {app.autoAssign && (
                                                <Chip
                                                    size="small"
                                                    icon={<Public fontSize="small" />}
                                                    label="Global"
                                                />
                                            )}
                                            {app.receiveNotifications === false && (
                                                <Chip
                                                    size="small"
                                                    variant="outlined"
                                                    icon={<NotificationsOff fontSize="small" />}
                                                    label="Muted"
                                                />
                                            )}
                                        </Stack>
                                        <Typography
                                            variant="body2"
                                            color="text.secondary"
                                            noWrap>
                                            {app.description || 'No description'}
                                        </Typography>
                                    </Box>
                                    <Button
                                        size="small"
                                        component={Link}
                                        to={`/channels/${app.id}`}
                                        endIcon={<ArrowForward fontSize="small" />}>
                                        Open
                                    </Button>
                                </Stack>
                            ))}
                        </Stack>
                    </SurfaceCard>
                </Grid>

                <Grid size={{xs: 12, lg: 5}}>
                    <SurfaceCard
                        title="Server & Security"
                        subtitle="Runtime identity and authentication state."
                        action={
                            <Chip
                                size="small"
                                color={currentUser.connectionErrorMessage ? 'warning' : 'success'}
                                label={currentUser.connectionErrorMessage ? 'Attention' : 'Connected'}
                            />
                        }>
                        <Stack spacing={1}>
                            <InfoRow label="Version" value={`@${version.version}`} />
                            <Divider />
                            <InfoRow label="Commit" value={version.commit || 'unknown'} mono />
                            <Divider />
                            <InfoRow label="Build date" value={version.buildDate || 'unknown'} />
                            <Divider />
                            <InfoRow label="Signed in as" value={currentUser.user.name} />
                            <Divider />
                            <Stack
                                direction="row"
                                spacing={1}
                                sx={{alignItems: 'center', justifyContent: 'space-between'}}>
                                <Typography variant="body2" color="text.secondary">
                                    Role
                                </Typography>
                                <Chip
                                    size="small"
                                    label={admin ? 'Administrator' : 'User'}
                                    icon={admin ? <Security fontSize="small" /> : undefined}
                                />
                            </Stack>
                            <Divider />
                            <Stack
                                direction="row"
                                spacing={1}
                                sx={{alignItems: 'center', justifyContent: 'space-between'}}>
                                <Typography variant="body2" color="text.secondary">
                                    Authentication
                                </Typography>
                                <Stack direction="row" spacing={0.5} useFlexGap sx={{flexWrap: 'wrap'}}>
                                    {config.get('localAuth') && (
                                        <Chip size="small" variant="outlined" label="Local" />
                                    )}
                                    {config.get('oidc') && (
                                        <Chip size="small" variant="outlined" label="OIDC" />
                                    )}
                                </Stack>
                            </Stack>
                        </Stack>
                    </SurfaceCard>
                </Grid>
            </Grid>
        </DefaultPage>
    );
});

const InfoRow = ({
    label,
    value,
    mono = false,
}: {
    label: string;
    value: string;
    mono?: boolean;
}) => (
    <Stack
        direction="row"
        spacing={2}
        sx={{alignItems: 'center', justifyContent: 'space-between'}}>
        <Typography variant="body2" color="text.secondary">
            {label}
        </Typography>
        <Typography
            variant="body2"
            sx={{
                fontWeight: 650,
                fontFamily: mono ? 'monospace' : undefined,
                maxWidth: '65%',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
            }}
            title={value}>
            {value}
        </Typography>
    </Stack>
);

export default Dashboard;
