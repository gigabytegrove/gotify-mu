import React from 'react';
import axios from 'axios';
import {
    Box,
    Button,
    Chip,
    Stack,
    Typography,
} from '@mui/material';
import Refresh from '@mui/icons-material/Refresh';
import Download from '@mui/icons-material/Download';
import Logout from '@mui/icons-material/Logout';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import * as config from '../config';
import {useStores} from '../stores';
import {
    IAdminSession,
    IAutomationRun,
    IIntegrationStatus,
    ISystemStats,
} from '../types';

const api = (path: string) => config.get('url') + path;
const formatBytes = (value?: number) => {
    if (!value || value <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let amount = value;
    let index = 0;
    while (amount >= 1024 && index < units.length - 1) {
        amount /= 1024;
        index++;
    }
    return amount.toFixed(index === 0 ? 0 : 1) + ' ' + units[index];
};

const Operations = () => {
    const {snackManager} = useStores();
    const [stats, setStats] = React.useState<ISystemStats>();
    const [sessions, setSessions] = React.useState<IAdminSession[]>([]);
    const [integrations, setIntegrations] = React.useState<IIntegrationStatus[]>([]);
    const [runs, setRuns] = React.useState<IAutomationRun[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [revoke, setRevoke] = React.useState<IAdminSession>();

    const refresh = React.useCallback(async () => {
        setLoading(true);
        try {
            const [statsResult, sessionsResult, integrationResult, runsResult] = await Promise.all([
                axios.get<ISystemStats>(api('operations/stats')),
                axios.get<IAdminSession[]>(api('operations/sessions')),
                axios.get<IIntegrationStatus[]>(api('integration/status')),
                axios.get<IAutomationRun[]>(api('automation/run?limit=100')),
            ]);
            setStats(statsResult.data);
            setSessions(sessionsResult.data);
            setIntegrations(integrationResult.data);
            setRuns(runsResult.data);
        } finally {
            setLoading(false);
        }
    }, []);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    const downloadDiagnostics = async () => {
        const response = await axios.get(api('operations/diagnostics'), {responseType: 'blob'});
        const href = URL.createObjectURL(response.data);
        const link = document.createElement('a');
        link.href = href;
        link.download = 'gotify-mu-diagnostics.json';
        link.click();
        URL.revokeObjectURL(href);
    };

    return (
        <DefaultPage
            title="Operations"
            description="Server health, sessions, integration status, automation history, and diagnostics."
            rightControl={
                <Stack direction="row" spacing={1}>
                    <Button startIcon={<Download />} onClick={() => void downloadDiagnostics()}>
                        Diagnostics
                    </Button>
                    <Button startIcon={<Refresh />} disabled={loading} onClick={() => void refresh()}>
                        Refresh
                    </Button>
                </Stack>
            }>
            <SurfaceCard title="System" subtitle="Current server totals and storage usage.">
                {stats ? (
                    <Box
                        sx={{
                            display: 'grid',
                            gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
                            gap: 1,
                        }}>
                        {[
                            ['Users', stats.users],
                            ['Channels', stats.channels],
                            ['Messages', stats.messages],
                            ['Clients', stats.clients],
                            ['Groups', stats.groups],
                            ['Webhooks', stats.webhooks],
                            ['MQTT', stats.mqttConnections],
                            ['Home Assistant', stats.homeAssistantConnections],
                            ['Schedules', stats.schedules],
                            ['Escalations', stats.escalationRules],
                            ['Audit Events', stats.auditEvents],
                            ['Automation Runs', stats.automationRuns],
                        ].map(([label, value]) => (
                            <Box key={String(label)} sx={{p: 1.5, border: 1, borderColor: 'divider', borderRadius: 2}}>
                                <Typography variant="caption" color="text.secondary">
                                    {label}
                                </Typography>
                                <Typography variant="h6">{value}</Typography>
                            </Box>
                        ))}
                        <Box sx={{p: 1.5, border: 1, borderColor: 'divider', borderRadius: 2}}>
                            <Typography variant="caption" color="text.secondary">Database</Typography>
                            <Typography variant="h6">{formatBytes(stats.databaseBytes)}</Typography>
                        </Box>
                        <Box sx={{p: 1.5, border: 1, borderColor: 'divider', borderRadius: 2}}>
                            <Typography variant="caption" color="text.secondary">Data</Typography>
                            <Typography variant="h6">{formatBytes(stats.dataBytes)}</Typography>
                        </Box>
                    </Box>
                ) : (
                    <Typography color="text.secondary">Loading system statistics…</Typography>
                )}
            </SurfaceCard>

            <SurfaceCard title="Active Sessions" subtitle="Signed-in clients that can currently access the server.">
                <Stack spacing={1}>
                    {sessions.length === 0 && <Typography color="text.secondary">No active sessions.</Typography>}
                    {sessions.map((session) => (
                        <Stack
                            key={session.id}
                            direction={{xs: 'column', sm: 'row'}}
                            spacing={1}
                            sx={{
                                p: 1.5,
                                border: 1,
                                borderColor: 'divider',
                                borderRadius: 2,
                                justifyContent: 'space-between',
                                alignItems: {sm: 'center'},
                            }}>
                            <Box>
                                <Typography sx={{fontWeight: 700}}>
                                    {session.displayName || session.username}
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    {session.name} · Last used{' '}
                                    {session.lastUsed ? new Date(session.lastUsed).toLocaleString() : 'not yet'}
                                </Typography>
                            </Box>
                            <Button
                                size="small"
                                color="error"
                                startIcon={<Logout />}
                                onClick={() => setRevoke(session)}>
                                Revoke
                            </Button>
                        </Stack>
                    ))}
                </Stack>
            </SurfaceCard>

            <SurfaceCard title="Integration Health" subtitle="Current runtime state reported by native integrations.">
                <Stack spacing={1}>
                    {integrations.length === 0 && (
                        <Typography color="text.secondary">No integration status has been reported yet.</Typography>
                    )}
                    {integrations.map((item) => (
                        <Stack
                            key={item.kind + '-' + item.objectId}
                            direction={{xs: 'column', sm: 'row'}}
                            spacing={1}
                            sx={{p: 1.5, border: 1, borderColor: 'divider', borderRadius: 2, justifyContent: 'space-between'}}>
                            <Box>
                                <Typography sx={{fontWeight: 700}}>
                                    {item.kind} #{item.objectId}
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    {item.lastError ||
                                        (item.lastActivityAt
                                            ? 'Last activity ' + new Date(item.lastActivityAt).toLocaleString()
                                            : 'No recent activity')}
                                </Typography>
                            </Box>
                            <Chip
                                size="small"
                                label={item.state}
                                color={item.state === 'connected' || item.state === 'healthy' ? 'success' : item.state === 'error' ? 'error' : 'default'}
                            />
                        </Stack>
                    ))}
                </Stack>
            </SurfaceCard>

            <SurfaceCard title="Automation History" subtitle="Recent Scheduled Notification, Digest, and Escalation runs.">
                <Stack spacing={1}>
                    {runs.length === 0 && <Typography color="text.secondary">No automation runs recorded yet.</Typography>}
                    {runs.map((run) => (
                        <Stack
                            key={run.id}
                            direction={{xs: 'column', sm: 'row'}}
                            spacing={1}
                            sx={{p: 1.5, border: 1, borderColor: 'divider', borderRadius: 2, justifyContent: 'space-between'}}>
                            <Box>
                                <Typography sx={{fontWeight: 700}}>
                                    {run.kind} #{run.objectId}
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    {new Date(run.startedAt).toLocaleString()}
                                    {run.error ? ' · ' + run.error : ''}
                                </Typography>
                            </Box>
                            <Chip
                                size="small"
                                label={run.status}
                                color={run.status === 'completed' ? 'success' : run.status === 'failed' ? 'error' : 'default'}
                            />
                        </Stack>
                    ))}
                </Stack>
            </SurfaceCard>

            {revoke && (
                <ConfirmDialog
                    title="Revoke session?"
                    text={'This immediately signs out ' + (revoke.displayName || revoke.username) + ' from ' + revoke.name + '.'}
                    requireElevated
                    fClose={() => setRevoke(undefined)}
                    fOnSubmit={() => {
                        void axios.delete(api('operations/sessions/' + revoke.id)).then(async () => {
                            setRevoke(undefined);
                            await refresh();
                            snackManager.snack('Session revoked');
                        });
                    }}
                />
            )}
        </DefaultPage>
    );
};

export default Operations;
