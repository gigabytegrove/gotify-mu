import React, {useState} from 'react';
import axios from 'axios';
import {
    Button,
    Chip,
    FormControl,
    InputLabel,
    MenuItem,
    Select,
    Stack,
    Switch,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import DarkMode from '@mui/icons-material/DarkMode';
import Security from '@mui/icons-material/Security';
import Key from '@mui/icons-material/Key';
import NotificationsNone from '@mui/icons-material/NotificationsNone';
import Schedule from '@mui/icons-material/Schedule';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ElevationForm from '../common/ElevationForm';
import {ThemeKey} from '../layout/theme';
import {useStores} from '../stores';
import * as config from '../config';
import {UpdateStatusCard} from '../update/UpdateStatus';
import {IDigestPolicy, IQuietHoursPolicy, ISecurityPolicy, IMFAStatus} from '../types';

interface IProps {
    themeMode: ThemeKey;
    setTheme: (theme: ThemeKey) => void;
}

const Settings = ({themeMode, setTheme}: IProps) => {
    const {currentUser} = useStores();

    return (
    <DefaultPage
        title="Settings"
        description="Account preferences and sign-in settings."
        maxWidth={900}>
        {currentUser.user.admin && <UpdateStatusCard />}
        {currentUser.user.admin && <AdministratorSecurityPolicy />}

        <MFASettings />
        <NotificationPreferences />

        <SurfaceCard
            title="Appearance"
            subtitle="Choose how Gotify MU looks on this device."
            action={<DarkMode color="action" />}>
            <FormControl fullWidth>
                <InputLabel id="theme-select-label">Theme</InputLabel>
                <Select
                    labelId="theme-select-label"
                    className="theme-select"
                    label="Theme"
                    value={themeMode}
                    onChange={(e) => setTheme(e.target.value as ThemeKey)}>
                    <MenuItem value="light">Light</MenuItem>
                    <MenuItem value="dark">Dark</MenuItem>
                    <MenuItem value="system">System</MenuItem>
                </Select>
            </FormControl>
        </SurfaceCard>

        <SurfaceCard
            title="Account Security"
            subtitle="Security controls for your local Gotify MU account."
            action={<Security color="action" />}>
            <Stack spacing={2}>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{justifyContent: 'space-between'}}> 
                    <Typography>Password sign-in</Typography>
                    <Chip
                        size="small"
                        label={config.get('localAuth') ? 'Enabled' : 'Disabled'}
                    />
                </Stack>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{justifyContent: 'space-between'}}> 
                    <Typography>Single sign-on</Typography>
                    <Chip size="small" label={config.get('oidc') ? 'Enabled' : 'Disabled'} />
                </Stack>
            </Stack>
        </SurfaceCard>

        <SurfaceCard
            title="Change Password"
            subtitle="Choose a new password for your account."
            action={<Key color="action" />}>
            <ChangePasswordForm />
        </SurfaceCard>
    </DefaultPage>
    );
};


const MFASettings = () => {
    const {elevateStore, snackManager} = useStores();
    const [status, setStatus] = React.useState<IMFAStatus>();
    const [secret, setSecret] = React.useState('');
    const [otpauthUri, setOtpauthUri] = React.useState('');
    const [code, setCode] = React.useState('');
    const [recoveryCodes, setRecoveryCodes] = React.useState<string[]>([]);

    const refresh = React.useCallback(async () => {
        const response = await axios.get<IMFAStatus>(config.get('url') + 'security/mfa');
        setStatus(response.data);
    }, []);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    if (!status) {
        return (
            <SurfaceCard title="Multi-factor Authentication" subtitle="Protect local sign-in with an authenticator app.">
                <Typography color="text.secondary">Loading multi-factor authentication…</Typography>
            </SurfaceCard>
        );
    }

    if (!elevateStore.elevated) {
        return (
            <SurfaceCard title="Multi-factor Authentication" subtitle="Protect local sign-in with an authenticator app.">
                <ElevationForm />
            </SurfaceCard>
        );
    }

    const start = async () => {
        const response = await axios.post<{secret: string; otpauthUri: string}>(
            config.get('url') + 'security/mfa/start'
        );
        setSecret(response.data.secret);
        setOtpauthUri(response.data.otpauthUri);
        setRecoveryCodes([]);
    };

    const enable = async () => {
        const response = await axios.post<{enabled: boolean; recoveryCodes: string[]}>(
            config.get('url') + 'security/mfa/enable',
            {code}
        );
        setRecoveryCodes(response.data.recoveryCodes);
        setCode('');
        setSecret('');
        setOtpauthUri('');
        await refresh();
        snackManager.snack('Multi-factor authentication enabled');
    };

    const disable = async () => {
        await axios.post(config.get('url') + 'security/mfa/disable', {code});
        setCode('');
        setRecoveryCodes([]);
        await refresh();
        snackManager.snack('Multi-factor authentication disabled');
    };

    return (
        <SurfaceCard
            title="Multi-factor Authentication"
            subtitle="Protect local sign-in with an authenticator app and recovery codes."
            action={<Security color="action" />}>
            <Stack spacing={2}>
                <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                    <Chip
                        size="small"
                        color={status.enabled ? 'success' : 'default'}
                        label={status.enabled ? 'Enabled' : 'Not enabled'}
                    />
                    {status.enabled && (
                        <Typography variant="body2" color="text.secondary">
                            {status.recoveryCodesRemaining} recovery codes remaining
                        </Typography>
                    )}
                </Stack>

                {!status.enabled && !secret && (
                    <Button variant="contained" onClick={() => void start()} sx={{alignSelf: 'flex-start'}}>
                        Set Up Authenticator
                    </Button>
                )}

                {!status.enabled && secret && (
                    <>
                        <Typography>
                            Add this secret to your authenticator app, then enter the current six-digit code.
                        </Typography>
                        <TextField
                            label="Authenticator secret"
                            value={secret}
                            slotProps={{input: {readOnly: true}}}
                            fullWidth
                        />
                        <TextField
                            label="Authenticator setup URI"
                            value={otpauthUri}
                            slotProps={{input: {readOnly: true}}}
                            fullWidth
                        />
                        <TextField
                            label="Verification code"
                            value={code}
                            onChange={(event) => setCode(event.target.value)}
                            autoComplete="one-time-code"
                            fullWidth
                        />
                        <Button
                            variant="contained"
                            disabled={!code}
                            onClick={() => void enable()}
                            sx={{alignSelf: 'flex-start'}}>
                            Enable MFA
                        </Button>
                    </>
                )}

                {status.enabled && (
                    <>
                        <TextField
                            label="Authenticator or recovery code"
                            value={code}
                            onChange={(event) => setCode(event.target.value)}
                            autoComplete="one-time-code"
                            helperText="Required to disable multi-factor authentication."
                            fullWidth
                        />
                        <Button
                            color="error"
                            variant="outlined"
                            disabled={!code}
                            onClick={() => void disable()}
                            sx={{alignSelf: 'flex-start'}}>
                            Disable MFA
                        </Button>
                    </>
                )}

                {recoveryCodes.length > 0 && (
                    <Box sx={{p: 1.5, border: 1, borderColor: 'warning.main', borderRadius: 2}}>
                        <Typography sx={{fontWeight: 700}}>Save these recovery codes now</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{mb: 1}}>
                            Each code works once. They will not be shown again.
                        </Typography>
                        <Typography component="pre" sx={{m: 0, whiteSpace: 'pre-wrap'}}>
                            {recoveryCodes.join('\n')}
                        </Typography>
                    </Box>
                )}
            </Stack>
        </SurfaceCard>
    );
};

const AdministratorSecurityPolicy = () => {
    const {snackManager} = useStores();
    const [policy, setPolicy] = React.useState<ISecurityPolicy>();
    const [saving, setSaving] = React.useState(false);

    React.useEffect(() => {
        void axios
            .get<ISecurityPolicy>(config.get('url') + 'security/policy')
            .then((response) => setPolicy(response.data));
    }, []);

    if (!policy) {
        return (
            <SurfaceCard
                title="Server Security"
                subtitle="Authentication, retention, and native extension policy."
                action={<Security color="action" />}>
                <Typography color="text.secondary">Loading server security policy…</Typography>
            </SurfaceCard>
        );
    }

    const save = async () => {
        setSaving(true);
        try {
            const response = await axios.put<ISecurityPolicy>(
                config.get('url') + 'security/policy',
                policy
            );
            setPolicy(response.data);
            snackManager.snack('Server security policy saved');
        } finally {
            setSaving(false);
        }
    };

    return (
        <SurfaceCard
            title="Server Security"
            subtitle="Authentication, retention, and native extension policy."
            action={<Security color="action" />}>
            <Stack spacing={2}>
                <Stack direction={{xs: 'column', sm: 'row'}} spacing={2}>
                    <TextField
                        label="Minimum password length"
                        type="number"
                        value={policy.minPasswordLength}
                        onChange={(event) =>
                            setPolicy({...policy, minPasswordLength: Number(event.target.value)})
                        }
                        slotProps={{htmlInput: {min: 8, max: 72}}}
                        fullWidth
                    />
                    <TextField
                        label="Web session lifetime"
                        type="number"
                        value={policy.sessionLifetimeHours}
                        onChange={(event) =>
                            setPolicy({...policy, sessionLifetimeHours: Number(event.target.value)})
                        }
                        helperText="Hours"
                        slotProps={{htmlInput: {min: 1, max: 8760}}}
                        fullWidth
                    />
                    <TextField
                        label="Elevated session"
                        type="number"
                        value={policy.elevationMinutes}
                        onChange={(event) =>
                            setPolicy({...policy, elevationMinutes: Number(event.target.value)})
                        }
                        helperText="Minutes"
                        slotProps={{htmlInput: {min: 1, max: 1440}}}
                        fullWidth
                    />
                </Stack>
                <Stack direction={{xs: 'column', sm: 'row'}} spacing={2}>
                    <TextField
                        label="Audit retention"
                        type="number"
                        value={policy.auditRetentionDays}
                        onChange={(event) =>
                            setPolicy({...policy, auditRetentionDays: Number(event.target.value)})
                        }
                        helperText="Days"
                        slotProps={{htmlInput: {min: 1, max: 3650}}}
                        fullWidth
                    />
                    <TextField
                        label="Automation history retention"
                        type="number"
                        value={policy.automationRetentionDays}
                        onChange={(event) =>
                            setPolicy({
                                ...policy,
                                automationRetentionDays: Number(event.target.value),
                            })
                        }
                        helperText="Days"
                        slotProps={{htmlInput: {min: 1, max: 3650}}}
                        fullWidth
                    />
                </Stack>
                <FormControlLabel
                    control={
                        <Switch
                            checked={policy.requireMfaAdmins}
                            onChange={(event) =>
                                setPolicy({...policy, requireMfaAdmins: event.target.checked})
                            }
                        />
                    }
                    label="Require multi-factor authentication for administrators"
                />
                <FormControlLabel
                    control={
                        <Switch
                            checked={policy.requireMfaAll}
                            onChange={(event) =>
                                setPolicy({...policy, requireMfaAll: event.target.checked})
                            }
                        />
                    }
                    label="Require multi-factor authentication for all local users"
                />
                <FormControlLabel
                    control={
                        <Switch
                            checked={policy.allowNativePluginUploads}
                            onChange={(event) =>
                                setPolicy({
                                    ...policy,
                                    allowNativePluginUploads: event.target.checked,
                                })
                            }
                        />
                    }
                    label="Allow administrators to upload native plugin binaries"
                />
                <FormControlLabel
                    control={
                        <Switch
                            checked={policy.requirePluginChecksum}
                            onChange={(event) =>
                                setPolicy({
                                    ...policy,
                                    requirePluginChecksum: event.target.checked,
                                })
                            }
                        />
                    }
                    label="Require a matching SHA-256 checksum for native plugin uploads"
                />
                <FormControlLabel
                    control={
                        <Switch
                            checked={policy.requirePluginSignature}
                            onChange={(event) =>
                                setPolicy({
                                    ...policy,
                                    requirePluginSignature: event.target.checked,
                                    requirePluginChecksum: event.target.checked
                                        ? true
                                        : policy.requirePluginChecksum,
                                })
                            }
                        />
                    }
                    label="Require a signature from a trusted plugin signing key"
                />
                <Typography variant="caption" color="text.secondary">
                    Native plugin binaries execute inside the Gotify MU server process. Uploads are
                    disabled by default. Signature verification uses trusted Ed25519 public keys
                    configured by the server administrator.
                </Typography>
                <Button
                    variant="contained"
                    disabled={saving}
                    onClick={() => void save()}
                    sx={{alignSelf: 'flex-start'}}>
                    Save Server Security
                </Button>
            </Stack>
        </SurfaceCard>
    );
};

const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';

const minuteToTime = (minute: number): string => {
    const normalized = Math.max(0, Math.min(1439, minute));
    return String(Math.floor(normalized / 60)).padStart(2, '0') + ':' +
        String(normalized % 60).padStart(2, '0');
};

const timeToMinute = (value: string): number => {
    const [hour, minute] = value.split(':').map(Number);
    return Math.max(0, Math.min(1439, hour * 60 + minute));
};

const NotificationPreferences = () => {
    const {snackManager} = useStores();
    const [quiet, setQuiet] = React.useState<IQuietHoursPolicy>();
    const [digest, setDigest] = React.useState<IDigestPolicy>();
    const [savingQuiet, setSavingQuiet] = React.useState(false);
    const [savingDigest, setSavingDigest] = React.useState(false);

    React.useEffect(() => {
        void Promise.all([
            axios
                .get<IQuietHoursPolicy>(config.get('url') + 'automation/quiet-hours')
                .then((response) =>
                    setQuiet({
                        ...response.data,
                        timezone: response.data.id
                            ? response.data.timezone
                            : browserTimezone,
                    })
                ),
            axios
                .get<IDigestPolicy>(config.get('url') + 'automation/digest')
                .then((response) => setDigest(response.data)),
        ]);
    }, []);

    const saveQuiet = async () => {
        if (!quiet) return;
        setSavingQuiet(true);
        try {
            const response = await axios.put<IQuietHoursPolicy>(
                config.get('url') + 'automation/quiet-hours',
                quiet
            );
            setQuiet(response.data);
            snackManager.snack('Quiet hours saved');
        } finally {
            setSavingQuiet(false);
        }
    };

    const saveDigest = async () => {
        if (!digest) return;
        setSavingDigest(true);
        try {
            const response = await axios.put<IDigestPolicy>(
                config.get('url') + 'automation/digest',
                digest
            );
            setDigest(response.data);
            snackManager.snack('Digest settings saved');
        } finally {
            setSavingDigest(false);
        }
    };

    if (!quiet || !digest) {
        return (
            <SurfaceCard
                title="Notification Preferences"
                subtitle="Choose when and how Gotify MU notifies you."
                action={<NotificationsNone color="action" />}>
                <Typography color="text.secondary">Loading notification preferences…</Typography>
            </SurfaceCard>
        );
    }

    return (
        <>
            <SurfaceCard
                title="Quiet Hours"
                subtitle="Pause lower-priority realtime notifications during a daily time window."
                action={<NotificationsNone color="action" />}>
                <Stack spacing={2}>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={2}
                        sx={{alignItems: {sm: 'center'}, justifyContent: 'space-between'}}>
                        <Typography>Quiet hours</Typography>
                        <Switch
                            checked={quiet.enabled}
                            onChange={(event) =>
                                setQuiet({...quiet, enabled: event.target.checked})
                            }
                        />
                    </Stack>
                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={2}>
                        <TextField
                            label="Start"
                            type="time"
                            value={minuteToTime(quiet.startMinute)}
                            onChange={(event) =>
                                setQuiet({...quiet, startMinute: timeToMinute(event.target.value)})
                            }
                            slotProps={{inputLabel: {shrink: true}}}
                            fullWidth
                        />
                        <TextField
                            label="End"
                            type="time"
                            value={minuteToTime(quiet.endMinute)}
                            onChange={(event) =>
                                setQuiet({...quiet, endMinute: timeToMinute(event.target.value)})
                            }
                            slotProps={{inputLabel: {shrink: true}}}
                            fullWidth
                        />
                    </Stack>
                    <TextField
                        label="Timezone"
                        value={quiet.timezone || browserTimezone}
                        onChange={(event) => setQuiet({...quiet, timezone: event.target.value})}
                        helperText="Your browser timezone is shown by default."
                    />
                    <TextField
                        label="Allow priority"
                        type="number"
                        value={quiet.allowPriority}
                        onChange={(event) =>
                            setQuiet({...quiet, allowPriority: Number(event.target.value)})
                        }
                        helperText="Messages at this priority or higher are delivered immediately during quiet hours."
                    />
                    <Button
                        variant="contained"
                        disabled={savingQuiet}
                        onClick={() => void saveQuiet()}
                        sx={{alignSelf: 'flex-start'}}>
                        Save Quiet Hours
                    </Button>
                </Stack>
            </SurfaceCard>

            <SurfaceCard
                title="Digest"
                subtitle="Group lower-priority notifications into a periodic summary."
                action={<Schedule color="action" />}>
                <Stack spacing={2}>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={2}
                        sx={{alignItems: {sm: 'center'}, justifyContent: 'space-between'}}>
                        <Typography>Notification digest</Typography>
                        <Switch
                            checked={digest.enabled}
                            onChange={(event) =>
                                setDigest({...digest, enabled: event.target.checked})
                            }
                        />
                    </Stack>
                    <TextField
                        select
                        label="Send digest every"
                        value={digest.intervalMinutes}
                        onChange={(event) =>
                            setDigest({...digest, intervalMinutes: Number(event.target.value)})
                        }>
                        <MenuItem value={15}>15 minutes</MenuItem>
                        <MenuItem value={30}>30 minutes</MenuItem>
                        <MenuItem value={60}>1 hour</MenuItem>
                        <MenuItem value={120}>2 hours</MenuItem>
                        <MenuItem value={240}>4 hours</MenuItem>
                        <MenuItem value={480}>8 hours</MenuItem>
                        <MenuItem value={1440}>24 hours</MenuItem>
                    </TextField>
                    <TextField
                        label="Send immediately at priority"
                        type="number"
                        value={digest.immediatePriority}
                        onChange={(event) =>
                            setDigest({...digest, immediatePriority: Number(event.target.value)})
                        }
                        helperText="Messages at this priority or higher skip the digest and notify you immediately."
                    />
                    <Button
                        variant="contained"
                        disabled={savingDigest}
                        onClick={() => void saveDigest()}
                        sx={{alignSelf: 'flex-start'}}>
                        Save Digest Settings
                    </Button>
                </Stack>
            </SurfaceCard>
        </>
    );
};

const ChangePasswordForm = () => {
    const [pass, setPass] = useState('');
    const {currentUser, elevateStore} = useStores();
    const localAuthEnabled = config.get('localAuth');

    const submit = () => {
        currentUser.changePassword(pass);
        setPass('');
    };

    if (!localAuthEnabled) {
        return <Typography color="text.secondary">Password sign-in is disabled on this server.</Typography>;
    }

    if (!elevateStore.elevated) {
        return <ElevationForm />;
    }

    return (
        <form
            id="changepw-form"
            onSubmit={(e) => {
                e.preventDefault();
                submit();
            }}>
            <Stack spacing={2}>
                <TextField
                    className="newpass"
                    type="password"
                    label="New Password"
                    value={pass}
                    disabled={!localAuthEnabled}
                    onChange={(e) => setPass(e.target.value)}
                    fullWidth
                />
                <Tooltip title={pass.length !== 0 ? '' : 'Password is required'}>
                    <span>
                        <Button
                            className="change"
                            type="submit"
                            disabled={!localAuthEnabled || pass.length === 0}
                            variant="contained">
                            Change Password
                        </Button>
                    </span>
                </Tooltip>
            </Stack>
        </form>
    );
};

export default Settings;
