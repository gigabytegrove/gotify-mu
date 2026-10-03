import React from 'react';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import Divider from '@mui/material/Divider';
import Box from '@mui/material/Box';
import Avatar from '@mui/material/Avatar';
import Paper from '@mui/material/Paper';
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
import GroupWork from '@mui/icons-material/GroupWork';
import FactCheck from '@mui/icons-material/FactCheck';
import Hub from '@mui/icons-material/Hub';
import AutoMode from '@mui/icons-material/AutoMode';
import FiberManualRecord from '@mui/icons-material/FiberManualRecord';
import {Link} from 'react-router';
import {observer} from 'mobx-react-lite';
import DefaultPage from '../common/DefaultPage';
import StatCard from '../common/StatCard';
import SurfaceCard from '../common/SurfaceCard';
import {useStores} from '../stores';
import * as config from '../config';
import {UpdateAvailableBanner} from '../update/UpdateStatus';

const Dashboard = observer(() => {
    const {appStore, userStore, clientStore, pluginStore, groupStore, currentUser} = useStores();
    const admin = currentUser.user.admin;

    React.useEffect(() => {
        void appStore.refresh();
        void clientStore.refresh();
        void pluginStore.refresh();
        if (admin) {
            void userStore.refresh();
            void groupStore.refresh();
        }
    }, [admin, appStore, clientStore, pluginStore, userStore, groupStore]);

    const apps = appStore.getItems();
    const globals = apps.filter((app) => app.autoAssign).length;
    const muted = apps.filter((app) => app.receiveNotifications === false).length;
    const chats = apps.filter(
        (app) =>
            app.channelType === 'chat' || (app.channelType == null && Boolean(app.allowMemberPost))
    ).length;
    const clients = clientStore.getItems();
    const plugins = pluginStore.getItems();
    const enabledPlugins = plugins.filter((plugin) => plugin.enabled).length;
    const users = admin ? userStore.getItems() : [];
    const groups = admin ? groupStore.getItems() : [];
    const version = config.get('version');

    const recentApps = [...apps]
        .sort((a, b) => {
            if (!a.lastUsed && !b.lastUsed) return 0;
            if (!a.lastUsed) return 1;
            if (!b.lastUsed) return -1;
            return Date.parse(b.lastUsed) - Date.parse(a.lastUsed);
        })
        .slice(0, 6);

    const displayName = currentUser.user.displayName || currentUser.user.name;

    return (
        <DefaultPage
            title="Dashboard"
            description="Your live view of Monita channels, access, integrations, and server health."
            rightControl={
                <Button component={Link} to="/messages" variant="contained" startIcon={<Inbox />}>
                    Open Messages
                </Button>
            }>
            {admin && <UpdateAvailableBanner />}

            <Paper
                variant="outlined"
                sx={{
                    position: 'relative',
                    overflow: 'hidden',
                    borderRadius: 3,
                    bgcolor: '#101B31',
                    color: '#FFFFFF',
                    borderColor: '#1C2B48',
                    boxShadow: '0 16px 44px rgba(10,18,32,.16)',
                }}>
                <Stack
                    direction={{xs: 'column', md: 'row'}}
                    spacing={2.5}
                    sx={{
                        p: {xs: 2.5, sm: 3},
                        alignItems: {md: 'center'},
                        justifyContent: 'space-between',
                    }}>
                    <Box sx={{maxWidth: 690}}>
                        <Stack
                            direction="row"
                            spacing={0.75}
                            sx={{alignItems: 'center', mb: 1.1}}>
                            <FiberManualRecord sx={{fontSize: 10, color: '#27D17F'}} />
                            <Typography
                                variant="overline"
                                sx={{color: 'rgba(231,238,248,.7)', fontSize: '0.68rem'}}>
                                Workspace online
                            </Typography>
                        </Stack>
                        <Typography
                            variant="h4"
                            sx={{
                                fontSize: {xs: '1.65rem', sm: '2rem'},
                                color: '#FFFFFF',
                                lineHeight: 1.08,
                            }}>
                            Welcome back, {displayName}
                        </Typography>
                        <Typography
                            sx={{
                                mt: 1,
                                color: 'rgba(225,234,246,.68)',
                                maxWidth: 610,
                                fontSize: '0.92rem',
                            }}>
                            {apps.length === 0
                                ? 'Create your first Channel to start routing notifications and conversations through Monita.'
                                : `${apps.length} channel${apps.length === 1 ? '' : 's'} are available, including ${chats} chat${chats === 1 ? '' : 's'} and ${globals} global channel${globals === 1 ? '' : 's'}.`}
                        </Typography>
                    </Box>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={1}
                        sx={{flexShrink: 0}}>
                        <Button
                            component={Link}
                            to="/channels"
                            variant="contained"
                            startIcon={<Forum />}
                            sx={{
                                bgcolor: '#FFFFFF',
                                color: '#101B31',
                                '&:hover': {bgcolor: '#EEF3FA'},
                            }}>
                            Manage Channels
                        </Button>
                        {admin && (
                            <Button
                                component={Link}
                                to="/integrations"
                                variant="outlined"
                                startIcon={<Hub />}
                                sx={{
                                    color: '#E7EEF8',
                                    borderColor: 'rgba(255,255,255,.22)',
                                    '&:hover': {
                                        borderColor: 'rgba(255,255,255,.4)',
                                        bgcolor: 'rgba(255,255,255,.05)',
                                    },
                                }}>
                                Integrations
                            </Button>
                        )}
                    </Stack>
                </Stack>
            </Paper>

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
                            helper={`${groups.length} group${groups.length === 1 ? '' : 's'}`}
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

            <Grid container spacing={1.5}>
                <Grid size={{xs: 12, lg: 8}}>
                    <SurfaceCard
                        title="Recent Channels"
                        subtitle="Your most recently active notification and chat destinations."
                        action={
                            <Button component={Link} to="/channels" size="small" endIcon={<ArrowForward />}>
                                All Channels
                            </Button>
                        }>
                        <Stack spacing={0.4}>
                            {recentApps.length === 0 && (
                                <Box sx={{py: 4, textAlign: 'center'}}>
                                    <Typography sx={{fontWeight: 700}}>No Channels yet</Typography>
                                    <Typography color="text.secondary" variant="body2" sx={{mt: 0.4}}>
                                        Create a Channel to start receiving notifications or chatting.
                                    </Typography>
                                    <Button
                                        component={Link}
                                        to="/channels"
                                        size="small"
                                        variant="outlined"
                                        sx={{mt: 1.5}}>
                                        Create a Channel
                                    </Button>
                                </Box>
                            )}
                            {recentApps.map((app, index) => (
                                <React.Fragment key={app.id}>
                                    <Stack
                                        direction="row"
                                        spacing={1.4}
                                        sx={{
                                            alignItems: 'center',
                                            py: 1,
                                            px: 0.5,
                                            borderRadius: 2,
                                            '&:hover': {bgcolor: 'action.hover'},
                                        }}>
                                        <Avatar
                                            src={config.get('url') + app.image}
                                            variant="rounded"
                                            sx={{width: 40, height: 40, borderRadius: 2}}
                                        />
                                        <Box sx={{minWidth: 0, flex: 1}}>
                                            <Stack
                                                direction="row"
                                                spacing={0.6}
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
                                                noWrap
                                                sx={{mt: 0.15}}>
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
                                    {index < recentApps.length - 1 && <Divider />}
                                </React.Fragment>
                            ))}
                        </Stack>
                    </SurfaceCard>
                </Grid>

                <Grid size={{xs: 12, lg: 4}}>
                    <Stack spacing={1.5}>
                        <SurfaceCard
                            title="Server"
                            subtitle="Current instance status."
                            action={
                                <Chip
                                    size="small"
                                    color={currentUser.connectionErrorMessage ? 'warning' : 'success'}
                                    label={
                                        currentUser.connectionErrorMessage ? 'Attention' : 'Healthy'
                                    }
                                />
                            }>
                            <Stack spacing={1.25}>
                                <InfoRow label="Version" value={`@${version.version}`} />
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
                            </Stack>
                        </SurfaceCard>

                        <SurfaceCard title="Quick Actions" subtitle="Common workspace tasks.">
                            <Stack spacing={0.75}>
                                <QuickAction to="/messages" icon={<Inbox />} label="View Messages" />
                                <QuickAction to="/channels" icon={<Forum />} label="Manage Channels" />
                                {admin && (
                                    <>
                                        <QuickAction to="/users" icon={<People />} label="Manage Users" />
                                        <QuickAction
                                            to="/automation"
                                            icon={<AutoMode />}
                                            label="Automation"
                                        />
                                        <QuickAction
                                            to="/audit"
                                            icon={<FactCheck />}
                                            label="Audit Log"
                                        />
                                    </>
                                )}
                                <QuickAction to="/settings" icon={<Settings />} label="Settings" />
                            </Stack>
                        </SurfaceCard>
                    </Stack>
                </Grid>
            </Grid>
        </DefaultPage>
    );
});

const QuickAction = ({
    to,
    icon,
    label,
}: {
    to: string;
    icon: React.ReactNode;
    label: string;
}) => (
    <Button
        component={Link}
        to={to}
        variant="text"
        startIcon={icon}
        endIcon={<ArrowForward fontSize="small" />}
        sx={{
            justifyContent: 'flex-start',
            color: 'text.primary',
            px: 1,
            '& .MuiButton-endIcon': {ml: 'auto'},
        }}>
        {label}
    </Button>
);

const InfoRow = ({label, value}: {label: string; value: string}) => (
    <Stack direction="row" spacing={2} sx={{alignItems: 'center', justifyContent: 'space-between'}}>
        <Typography variant="body2" color="text.secondary">
            {label}
        </Typography>
        <Typography
            variant="body2"
            sx={{
                fontWeight: 680,
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
