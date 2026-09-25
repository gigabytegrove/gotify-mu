import React from 'react';
import axios from 'axios';
import {
    Alert,
    Box,
    Button,
    Chip,
    FormControlLabel,
    Stack,
    Switch,
    TextField,
    Typography,
} from '@mui/material';
import Refresh from '@mui/icons-material/Refresh';
import Security from '@mui/icons-material/Security';
import Devices from '@mui/icons-material/Devices';
import Assessment from '@mui/icons-material/Assessment';
import Download from '@mui/icons-material/Download';
import Upload from '@mui/icons-material/Upload';
import Backup from '@mui/icons-material/Backup';
import BugReport from '@mui/icons-material/BugReport';
import Delete from '@mui/icons-material/Delete';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import * as config from '../config';
import {useStores} from '../stores';
import {IAdminSession, IOperationsSummary, ISecurityPolicy} from '../types';

const api = (path: string) => config.get('url') + path;

const SystemAdministration = () => {
    const {snackManager} = useStores();
    const [policy, setPolicy] = React.useState<ISecurityPolicy>();
    const [operations, setOperations] = React.useState<IOperationsSummary>();
    const [sessions, setSessions] = React.useState<IAdminSession[]>([]);
    const [loading, setLoading] = React.useState(false);
    const [saving, setSaving] = React.useState(false);
    const [restoreStaged, setRestoreStaged] = React.useState(false);
    const [restoring, setRestoring] = React.useState(false);
    const restoreInput = React.useRef<HTMLInputElement | null>(null);

    const refresh = React.useCallback(async () => {
        setLoading(true);
        try {
            const [policyResponse, operationsResponse, sessionsResponse] = await Promise.all([
                axios.get<ISecurityPolicy>(api('admin/security-policy')),
                axios.get<IOperationsSummary>(api('admin/operations')),
                axios.get<IAdminSession[]>(api('admin/sessions')),
            ]);
            setPolicy(policyResponse.data);
            setOperations(operationsResponse.data);
            setSessions(sessionsResponse.data);
        } finally {
            setLoading(false);
        }
    }, []);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    const savePolicy = async () => {
        if (!policy) return;
        setSaving(true);
        try {
            const response = await axios.put<ISecurityPolicy>(
                api('admin/security-policy'),
                policy
            );
            setPolicy(response.data);
            snackManager.snack('Security policy saved');
        } finally {
            setSaving(false);
        }
    };

    const downloadAudit = (format: 'csv' | 'json') => {
        window.location.href = api('audit/export?format=' + format);
    };

    const stageRestore = async (file?: File) => {
        if (!file) return;
        setRestoring(true);
        try {
            const form = new FormData();
            form.append('backup', file);
            await axios.post(api('admin/restore'), form, {
                headers: {'Content-Type': 'multipart/form-data'},
            });
            setRestoreStaged(true);
            snackManager.snack('Backup staged. Restart Gotify MU to apply it.');
        } finally {
            setRestoring(false);
            if (restoreInput.current) restoreInput.current.value = '';
        }
    };

    return (
        <DefaultPage
            title="Security & Operations"
            description="Server-wide security policy, sessions, audit retention, and operational status."
            rightControl={
                <Button startIcon={<Refresh />} onClick={() => void refresh()} disabled={loading}>
                    Refresh
                </Button>
            }>
            <SurfaceCard
                title="Security Policy"
                subtitle="Apply consistent password, session, MFA, and audit-retention requirements."
                action={<Security color="action" />}>
                {!policy ? (
                    <Typography color="text.secondary">Loading security policy…</Typography>
                ) : (
                    <Stack spacing={2}>
                        <TextField
                            type="number"
                            label="Minimum password length"
                            value={policy.minimumPasswordLength}
                            onChange={(event) =>
                                setPolicy({
                                    ...policy,
                                    minimumPasswordLength: Number(event.target.value),
                                })
                            }
                            slotProps={{htmlInput: {min: 8, max: 72}}}
                        />
                        <TextField
                            type="number"
                            label="Session inactivity timeout"
                            value={policy.sessionInactivityMinutes}
                            onChange={(event) =>
                                setPolicy({
                                    ...policy,
                                    sessionInactivityMinutes: Number(event.target.value),
                                })
                            }
                            helperText="Minutes before an inactive browser/client session expires."
                            slotProps={{htmlInput: {min: 5, max: 525600}}}
                        />
                        <TextField
                            type="number"
                            label="Administrative re-authentication duration"
                            value={policy.elevationMinutes}
                            onChange={(event) =>
                                setPolicy({
                                    ...policy,
                                    elevationMinutes: Number(event.target.value),
                                })
                            }
                            helperText="Minutes before protected administrative actions require identity confirmation again."
                            slotProps={{htmlInput: {min: 1, max: 1440}}}
                        />
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={policy.requireMfaForAdmins}
                                    onChange={(event) =>
                                        setPolicy({
                                            ...policy,
                                            requireMfaForAdmins: event.target.checked,
                                        })
                                    }
                                />
                            }
                            label="Require MFA for local administrator accounts"
                        />
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={policy.requireMfaForAllLocalUsers}
                                    onChange={(event) =>
                                        setPolicy({
                                            ...policy,
                                            requireMfaForAllLocalUsers: event.target.checked,
                                        })
                                    }
                                />
                            }
                            label="Require MFA for all local password accounts"
                        />
                        <TextField
                            type="number"
                            label="Audit retention"
                            value={policy.auditRetentionDays}
                            onChange={(event) =>
                                setPolicy({
                                    ...policy,
                                    auditRetentionDays: Number(event.target.value),
                                })
                            }
                            helperText="Days to retain administrative/security audit history."
                            slotProps={{htmlInput: {min: 1, max: 3650}}}
                        />
                        <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                            <Button
                                variant="contained"
                                disabled={saving}
                                onClick={() => void savePolicy()}>
                                Save Security Policy
                            </Button>
                            <Button
                                variant="outlined"
                                disabled={saving}
                                onClick={async () => {
                                    await axios.post(api('audit/retention/apply'));
                                    await refresh();
                                    snackManager.snack('Audit retention applied');
                                }}>
                                Apply Retention Now
                            </Button>
                        </Stack>
                        {(policy.requireMfaForAdmins || policy.requireMfaForAllLocalUsers) && (
                            <Alert severity="info">
                                MFA requirements apply to local password accounts. External single
                                sign-on continues to use the security controls of its identity
                                provider.
                            </Alert>
                        )}
                    </Stack>
                )}
            </SurfaceCard>

            <SurfaceCard
                title="Operations"
                subtitle="Current server state and pending notification work."
                action={<Assessment color="action" />}>
                {!operations ? (
                    <Typography color="text.secondary">Loading operations status…</Typography>
                ) : (
                    <Box
                        sx={{
                            display: 'grid',
                            gridTemplateColumns: {
                                xs: 'repeat(2, minmax(0, 1fr))',
                                md: 'repeat(4, minmax(0, 1fr))',
                            },
                            gap: 1.25,
                        }}>
                        {[
                            ['Users', operations.users],
                            ['Channels', operations.channels],
                            ['Messages', operations.messages],
                            ['Sessions', operations.clients],
                            ['Plugins', operations.plugins],
                            ['Webhooks', operations.webhooks],
                            ['MQTT', operations.mqttConnections],
                            ['Home Assistant', operations.homeAssistantConnections],
                            ['Schedules', operations.schedules],
                            ['Pending Escalations', operations.pendingEscalations],
                            ['Pending Digests', operations.pendingDigests],
                            ['Deferred Alerts', operations.deferredNotifications],
                            ['Audit Events', operations.auditEvents],
                        ].map(([label, value]) => (
                            <Box
                                key={String(label)}
                                sx={{border: 1, borderColor: 'divider', borderRadius: 2, p: 1.5}}>
                                <Typography variant="caption" color="text.secondary">
                                    {label}
                                </Typography>
                                <Typography variant="h5">{value}</Typography>
                            </Box>
                        ))}
                        <Box sx={{gridColumn: '1 / -1'}}>
                            <Chip
                                size="small"
                                variant="outlined"
                                label={'Database: ' + operations.databaseDialect}
                            />
                        </Box>
                    </Box>
                )}
            </SurfaceCard>

            <SurfaceCard
                title="Active Sessions"
                subtitle="Review and revoke active client/browser sessions across all users."
                action={<Devices color="action" />}>
                {sessions.length === 0 ? (
                    <Typography color="text.secondary">No active sessions.</Typography>
                ) : (
                    <Stack spacing={1}>
                        {sessions.map((session) => (
                            <Box
                                key={session.id}
                                sx={{border: 1, borderColor: 'divider', borderRadius: 2, p: 1.5}}>
                                <Stack
                                    direction={{xs: 'column', sm: 'row'}}
                                    spacing={1}
                                    sx={{justifyContent: 'space-between', alignItems: {sm: 'center'}}}>
                                    <Box>
                                        <Typography sx={{fontWeight: 700}}>
                                            {session.username || 'User #' + session.userId}
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary">
                                            {session.name}
                                        </Typography>
                                        <Typography variant="caption" color="text.secondary">
                                            Last used:{' '}
                                            {session.lastUsed
                                                ? new Date(session.lastUsed).toLocaleString()
                                                : 'Never'}
                                        </Typography>
                                    </Box>
                                    <Button
                                        color="error"
                                        size="small"
                                        startIcon={<Delete />}
                                        onClick={async () => {
                                            await axios.delete(api('admin/sessions/' + session.id));
                                            await refresh();
                                            snackManager.snack('Session revoked');
                                        }}>
                                        Revoke
                                    </Button>
                                </Stack>
                            </Box>
                        ))}
                    </Stack>
                )}
            </SurfaceCard>

            <SurfaceCard
                title="Backup & Recovery"
                subtitle="Download a complete recovery bundle or stage a verified restore for the next restart."
                action={<Backup color="action" />}>
                <Stack spacing={1.5}>
                    {restoreStaged && (
                        <Alert
                            severity="warning"
                            action={
                                <Button
                                    color="inherit"
                                    size="small"
                                    onClick={async () => {
                                        await axios.delete(api('admin/restore'));
                                        setRestoreStaged(false);
                                        snackManager.snack('Staged restore cancelled');
                                    }}>
                                    Cancel Restore
                                </Button>
                            }>
                            A restore is staged. Restart Gotify MU to apply it before the database
                            opens. A pre-restore safety backup will be created automatically.
                        </Alert>
                    )}
                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                        <Button
                            variant="contained"
                            startIcon={<Download />}
                            component="a"
                            href={api('admin/backup')}>
                            Download Full Backup
                        </Button>
                        <input
                            hidden
                            type="file"
                            accept=".zip,application/zip"
                            ref={restoreInput}
                            onChange={(event) => void stageRestore(event.target.files?.[0])}
                        />
                        <Button
                            variant="outlined"
                            startIcon={<Upload />}
                            disabled={restoring}
                            onClick={() => restoreInput.current?.click()}>
                            {restoring ? 'Validating Backup…' : 'Stage Restore'}
                        </Button>
                        <Button
                            variant="outlined"
                            startIcon={<BugReport />}
                            component="a"
                            href={api('admin/diagnostics')}>
                            Download Diagnostics
                        </Button>
                    </Stack>
                    <Typography variant="body2" color="text.secondary">
                        Full backups include the database snapshot, encrypted-integration key,
                        uploaded images, attachments, plugins, and other persistent data. Restore
                        files are validated before they are accepted.
                    </Typography>
                </Stack>
            </SurfaceCard>

            <SurfaceCard
                title="Audit Export"
                subtitle="Export retained audit events for review or long-term storage."
                action={<Download color="action" />}>
                <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                    <Button
                        variant="outlined"
                        startIcon={<Download />}
                        onClick={() => downloadAudit('csv')}>
                        Export CSV
                    </Button>
                    <Button
                        variant="outlined"
                        startIcon={<Download />}
                        onClick={() => downloadAudit('json')}>
                        Export JSON
                    </Button>
                </Stack>
            </SurfaceCard>
        </DefaultPage>
    );
};

export default SystemAdministration;
