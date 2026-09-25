import React from 'react';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import Divider from '@mui/material/Divider';
import NotificationsActive from '@mui/icons-material/NotificationsActive';
import Public from '@mui/icons-material/Public';
import People from '@mui/icons-material/People';
import DevicesOther from '@mui/icons-material/DevicesOther';
import Extension from '@mui/icons-material/Extension';
import Security from '@mui/icons-material/Security';
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
    const clients = clientStore.getItems();
    const plugins = pluginStore.getItems();
    const users = admin ? userStore.getItems() : [];
    const version = config.get('version');

    return (
        <DefaultPage
            title="Dashboard"
            description="A single view of this Gotify MU server and its multi-user notification environment.">
            <Grid container spacing={2}>
                <Grid size={{xs: 12, sm: 6, lg: 3}}>
                    <StatCard
                        label="Channels"
                        value={apps.length}
                        helper={`${globals} global`}
                        icon={<NotificationsActive />}
                    />
                </Grid>
                {admin && (
                    <Grid size={{xs: 12, sm: 6, lg: 3}}>
                        <StatCard label="Users" value={users.length} icon={<People />} />
                    </Grid>
                )}
                <Grid size={{xs: 12, sm: 6, lg: 3}}>
                    <StatCard label="Clients" value={clients.length} icon={<DevicesOther />} />
                </Grid>
                <Grid size={{xs: 12, sm: 6, lg: 3}}>
                    <StatCard label="Plugins" value={plugins.length} icon={<Extension />} />
                </Grid>
            </Grid>

            <Grid container spacing={2}>
                <Grid size={{xs: 12, md: 7}}>
                    <SurfaceCard
                        title="Channels"
                        subtitle="Shared notification destinations available to your account."
                        action={
                            <Button component={Link} to="/channels">
                                Manage Channels
                            </Button>
                        }>
                        <Stack spacing={1.5}>
                            {apps.length === 0 && (
                                <Typography color="text.secondary">
                                    No Channels are available yet.
                                </Typography>
                            )}
                            {apps.slice(0, 6).map((app) => (
                                <Stack
                                    key={app.id}
                                    direction="row"
                                    spacing={2}
                                    sx={{alignItems: 'center', justifyContent: 'space-between'}}>
                                    <Stack sx={{minWidth: 0}}>
                                        <Typography sx={{fontWeight: 600}} noWrap>
                                            {app.name}
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" noWrap>
                                            {app.description || 'No description'}
                                        </Typography>
                                    </Stack>
                                    <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                        {app.autoAssign && (
                                            <Chip
                                                size="small"
                                                label="Global"
                                                icon={<Public fontSize="small" />}
                                            />
                                        )}
                                        <Button
                                            size="small"
                                            component={Link}
                                            to={`/channels/${app.id}`}>
                                            Open
                                        </Button>
                                    </Stack>
                                </Stack>
                            ))}
                        </Stack>
                    </SurfaceCard>
                </Grid>

                <Grid size={{xs: 12, md: 5}}>
                    <SurfaceCard title="Server" subtitle="Runtime identity and authentication status.">
                        <Stack spacing={1.5}>
                            <Stack direction="row" spacing={2} sx={{justifyContent: 'space-between'}}>
                                <Typography color="text.secondary">Version</Typography>
                                <Typography sx={{fontWeight: 600}}>@{version.version}</Typography>
                            </Stack>
                            <Divider />
                            <Stack direction="row" spacing={2} sx={{justifyContent: 'space-between'}}>
                                <Typography color="text.secondary">Signed in as</Typography>
                                <Typography sx={{fontWeight: 600}}>{currentUser.user.name}</Typography>
                            </Stack>
                            <Divider />
                            <Stack direction="row" spacing={2} sx={{justifyContent: 'space-between'}}>
                                <Typography color="text.secondary">Role</Typography>
                                <Chip
                                    size="small"
                                    label={admin ? 'Administrator' : 'User'}
                                    icon={admin ? <Security fontSize="small" /> : undefined}
                                />
                            </Stack>
                            <Divider />
                            <Stack direction="row" spacing={2} sx={{justifyContent: 'space-between'}}>
                                <Typography color="text.secondary">Local login</Typography>
                                <Chip
                                    size="small"
                                    label={config.get('localAuth') ? 'Enabled' : 'Disabled'}
                                />
                            </Stack>
                            <Stack direction="row" spacing={2} sx={{justifyContent: 'space-between'}}>
                                <Typography color="text.secondary">OIDC</Typography>
                                <Chip
                                    size="small"
                                    label={config.get('oidc') ? 'Enabled' : 'Disabled'}
                                />
                            </Stack>
                        </Stack>
                    </SurfaceCard>
                </Grid>
            </Grid>
        </DefaultPage>
    );
});

export default Dashboard;
