import React from 'react';
import axios from 'axios';
import {
    Box,
    Button,
    Checkbox,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControlLabel,
    Stack,
    Typography,
} from '@mui/material';
import Refresh from '@mui/icons-material/Refresh';
import Download from '@mui/icons-material/Download';
import Logout from '@mui/icons-material/Logout';
import Add from '@mui/icons-material/Add';
import ContentCopy from '@mui/icons-material/ContentCopy';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import * as config from '../config';
import {useStores} from '../stores';
import {
    IAdminSession,
    IAutomationRun,
    IIntegrationStatus,
    IServiceAccount,
    IServiceAccountCreated,
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
    const {snackManager, appStore} = useStores();
    const [stats, setStats] = React.useState<ISystemStats>();
    const [sessions, setSessions] = React.useState<IAdminSession[]>([]);
    const [integrations, setIntegrations] = React.useState<IIntegrationStatus[]>([]);
    const [runs, setRuns] = React.useState<IAutomationRun[]>([]);
    const [serviceAccounts, setServiceAccounts] = React.useState<IServiceAccount[]>([]);
    const [serviceAccountOpen, setServiceAccountOpen] = React.useState(false);
    const [serviceToken, setServiceToken] = React.useState<IServiceAccountCreated>();
    const [loading, setLoading] = React.useState(true);
    const [revoke, setRevoke] = React.useState<IAdminSession>();
    const [restoreResult, setRestoreResult] = React.useState<string>();

    const refresh = React.useCallback(async () => {
        setLoading(true);
        try {
            await appStore.refresh();
            const [statsResult, sessionsResult, integrationResult, runsResult, serviceResult] =
                await Promise.all([
                    axios.get<ISystemStats>(api('operations/stats')),
                    axios.get<IAdminSession[]>(api('operations/sessions')),
                    axios.get<IIntegrationStatus[]>(api('integration/status')),
                    axios.get<IAutomationRun[]>(api('automation/run?limit=100')),
                    axios.get<IServiceAccount[]>(api('service-account')),
                ]);
            setStats(statsResult.data);
            setSessions(sessionsResult.data);
            setIntegrations(integrationResult.data);
            setRuns(runsResult.data);
            setServiceAccounts(serviceResult.data);
        } finally {
            setLoading(false);
        }
    }, [appStore]);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    const downloadBackup = async () => {
        const response = await axios.get(api('operations/backup'), {responseType: 'blob'});
        const href = URL.createObjectURL(response.data);
        const link = document.createElement('a');
        link.href = href;
        link.download = 'gotify-mu-backup-' + new Date().toISOString().slice(0, 10) + '.zip';
        link.click();
        URL.revokeObjectURL(href);
    };

    const stageRestore = async (file: File) => {
        const form = new FormData();
        form.append('backup', file);
        const response = await axios.post<{
            message: string;
            backupVersion?: string;
        }>(api('operations/restore'), form);
        setRestoreResult(response.data.message);
        snackManager.snack('Backup staged for restore');
    };

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
                    <Button startIcon={<Download />} onClick={() => void downloadBackup()}>
                        Backup
                    </Button>
                    <Button startIcon={<Refresh />} disabled={loading} onClick={() => void refresh()}>
                        Refresh
                    </Button>
                </Stack>
            }>
            <SurfaceCard
                title="Backup & Restore"
                subtitle="Download a consistent server backup or stage a verified backup for restoration on the next restart.">
                <Stack spacing={1.5}>
                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                        <Button variant="contained" onClick={() => void downloadBackup()}>
                            Download Backup
                        </Button>
                        <Button component="label" variant="outlined">
                            Stage Restore
                            <input
                                hidden
                                type="file"
                                accept=".zip,application/zip"
                                onChange={(event) => {
                                    const selected = event.target.files?.[0];
                                    if (selected) void stageRestore(selected);
                                    event.target.value = '';
                                }}
                            />
                        </Button>
                    </Stack>
                    <Typography variant="body2" color="text.secondary">
                        Restore archives are validated before staging. The restore is applied before
                        the database opens on the next Gotify MU restart, and the current data is
                        preserved automatically as an emergency pre-restore copy.
                    </Typography>
                    {restoreResult && (
                        <Typography variant="body2" color="warning.main">
                            {restoreResult}
                        </Typography>
                    )}
                </Stack>
            </SurfaceCard>

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

            <SurfaceCard
                title="Service Accounts"
                subtitle="Scoped non-interactive credentials for external systems and automation."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setServiceAccountOpen(true)}>
                        Add Service Account
                    </Button>
                }>
                <Stack spacing={1}>
                    {serviceAccounts.length === 0 && (
                        <Typography color="text.secondary">No service accounts configured.</Typography>
                    )}
                    {serviceAccounts.map((account) => (
                        <Stack
                            key={account.id}
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
                                <Typography sx={{fontWeight: 700}}>{account.name}</Typography>
                                <Typography variant="body2" color="text.secondary">
                                    {account.scopes} · Channels {account.channelIds}
                                    {account.expiresAt
                                        ? ' · Expires ' + new Date(account.expiresAt).toLocaleString()
                                        : ' · No expiration'}
                                </Typography>
                            </Box>
                            <Button
                                size="small"
                                color="error"
                                onClick={() => {
                                    void axios.delete(api('service-account/' + account.id)).then(async () => {
                                        await refresh();
                                        snackManager.snack('Service account revoked');
                                    });
                                }}>
                                Revoke
                            </Button>
                        </Stack>
                    ))}
                </Stack>
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

            {serviceAccountOpen && (
                <ServiceAccountDialog
                    channels={appStore.getItems()}
                    onClose={() => setServiceAccountOpen(false)}
                    onCreated={async (created) => {
                        setServiceAccountOpen(false);
                        setServiceToken(created);
                        await refresh();
                    }}
                />
            )}
            {serviceToken && (
                <Dialog open onClose={() => setServiceToken(undefined)} fullWidth maxWidth="sm">
                    <DialogTitle>Save Service Account Token</DialogTitle>
                    <DialogContent>
                        <Stack spacing={2} sx={{pt: 1}}>
                            <Typography color="warning.main">
                                This token is shown once. Store it before closing this window.
                            </Typography>
                            <TextField
                                label="Token"
                                value={serviceToken.token}
                                fullWidth
                                slotProps={{input: {readOnly: true}}}
                            />
                            <Button
                                startIcon={<ContentCopy />}
                                onClick={() => {
                                    void navigator.clipboard.writeText(serviceToken.token);
                                    snackManager.snack('Service account token copied');
                                }}>
                                Copy Token
                            </Button>
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="contained" onClick={() => setServiceToken(undefined)}>
                            I Saved This Token
                        </Button>
                    </DialogActions>
                </Dialog>
            )}
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

const ServiceAccountDialog = ({
    channels,
    onClose,
    onCreated,
}: {
    channels: Array<{id: number; name: string}>;
    onClose: VoidFunction;
    onCreated: (created: IServiceAccountCreated) => Promise<void>;
}) => {
    const [name, setName] = React.useState('');
    const [scopes, setScopes] = React.useState<string[]>(['message:write']);
    const [channelIds, setChannelIds] = React.useState<number[]>([]);
    const [expiresAt, setExpiresAt] = React.useState('');
    const [saving, setSaving] = React.useState(false);

    const toggleScope = (scope: string) =>
        setScopes((current) =>
            current.includes(scope)
                ? current.filter((value) => value !== scope)
                : [...current, scope]
        );

    const toggleChannel = (id: number) =>
        setChannelIds((current) =>
            current.includes(id) ? current.filter((value) => value !== id) : [...current, id]
        );

    const save = async () => {
        setSaving(true);
        try {
            const response = await axios.post<IServiceAccountCreated>(api('service-account'), {
                name,
                scopes,
                channelIds,
                expiresAt: expiresAt ? new Date(expiresAt).toISOString() : null,
            });
            await onCreated(response.data);
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>Add Service Account</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    <TextField
                        label="Name"
                        value={name}
                        onChange={(event) => setName(event.target.value)}
                        required
                    />
                    <Box>
                        <Typography sx={{fontWeight: 700, mb: 0.5}}>API access</Typography>
                        {[
                            ['channels:read', 'Read Channel information'],
                            ['message:read', 'Read Channel messages'],
                            ['message:write', 'Send Channel messages'],
                        ].map(([scope, label]) => (
                            <FormControlLabel
                                key={scope}
                                control={
                                    <Checkbox
                                        checked={scopes.includes(scope)}
                                        onChange={() => toggleScope(scope)}
                                    />
                                }
                                label={label}
                            />
                        ))}
                    </Box>
                    <Box>
                        <Typography sx={{fontWeight: 700, mb: 0.5}}>Allowed Channels</Typography>
                        <Stack>
                            {channels.map((channel) => (
                                <FormControlLabel
                                    key={channel.id}
                                    control={
                                        <Checkbox
                                            checked={channelIds.includes(channel.id)}
                                            onChange={() => toggleChannel(channel.id)}
                                        />
                                    }
                                    label={channel.name}
                                />
                            ))}
                        </Stack>
                    </Box>
                    <TextField
                        type="datetime-local"
                        label="Expiration"
                        value={expiresAt}
                        onChange={(event) => setExpiresAt(event.target.value)}
                        helperText="Optional. Leave blank for no expiration."
                        slotProps={{inputLabel: {shrink: true}}}
                    />
                    <Typography variant="body2" color="text.secondary">
                        Service account tokens use dedicated /service/v1 endpoints and cannot be
                        used as interactive browser sessions.
                    </Typography>
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
                <Button
                    variant="contained"
                    disabled={saving || !name.trim() || scopes.length === 0 || channelIds.length === 0}
                    onClick={() => void save()}>
                    Create
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default Operations;
