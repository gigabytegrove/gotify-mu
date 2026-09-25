import React, {useState} from 'react';
import axios from 'axios';
import {
    Alert,
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
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
import {IDigestPolicy, IMFASetupResult, IMFAStatus, IPasskey, IQuietHoursPolicy} from '../types';
import {createPasskey} from '../passkey';
import ConfirmDialog from '../common/ConfirmDialog';
import {PriorityField, TimezoneField} from '../common/NotificationFields';

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

        <MFASettings />
        <PasskeySettings />

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
                        mode: response.data.mode || 'suppress',
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
                    <TimezoneField
                        value={quiet.timezone || browserTimezone}
                        onChange={(timezone) => setQuiet({...quiet, timezone})}
                    />
                    <PriorityField
                        label="Allow immediately at priority"
                        value={quiet.allowPriority}
                        onChange={(allowPriority) => setQuiet({...quiet, allowPriority})}
                        helperText="Messages at this priority or higher are delivered immediately during quiet hours."
                    />
                    <TextField
                        select
                        label="Lower-priority notifications"
                        value={quiet.mode || 'suppress'}
                        onChange={(event) =>
                            setQuiet({...quiet, mode: event.target.value as 'suppress' | 'defer'})
                        }>
                        <MenuItem value="suppress">Keep in history without a realtime alert</MenuItem>
                        <MenuItem value="defer">Send the realtime alert after quiet hours end</MenuItem>
                    </TextField>
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
                    <PriorityField
                        label="Send immediately at priority"
                        value={digest.immediatePriority}
                        onChange={(immediatePriority) =>
                            setDigest({...digest, immediatePriority})
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


const MFASettings = () => {
    const {currentUser, elevateStore, snackManager} = useStores();
    const [status, setStatus] = React.useState<IMFAStatus>();
    const [setup, setSetup] = React.useState<IMFASetupResult>();
    const [code, setCode] = React.useState('');
    const [busy, setBusy] = React.useState(false);
    const [recoveryCodes, setRecoveryCodes] = React.useState<string[]>();

    const refresh = React.useCallback(async () => {
        const response = await axios.get<IMFAStatus>(config.get('url') + 'current/user/mfa/status');
        setStatus(response.data);
    }, []);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    if (!config.get('localAuth')) return null;

    const requireElevation = (): boolean => {
        if (!elevateStore.elevated) {
            elevateStore.requestReauthentication();
            return false;
        }
        return true;
    };

    const beginSetup = async () => {
        if (!requireElevation()) return;
        setBusy(true);
        try {
            const response = await axios.post<IMFASetupResult>(
                config.get('url') + 'current/user/mfa/setup'
            );
            setSetup(response.data);
            setRecoveryCodes(response.data.recoveryCodes);
            setCode('');
        } finally {
            setBusy(false);
        }
    };

    const enable = async () => {
        if (!setup || !code) return;
        setBusy(true);
        try {
            await axios.post(config.get('url') + 'current/user/mfa/enable', {code});
            await refresh();
            await currentUser.tryAuthenticate();
            setSetup(undefined);
            setCode('');
            snackManager.snack('Multi-factor authentication enabled');
        } finally {
            setBusy(false);
        }
    };

    const disable = async () => {
        if (!requireElevation() || !code) return;
        setBusy(true);
        try {
            await axios.post(config.get('url') + 'current/user/mfa/disable', {code});
            await refresh();
            await currentUser.tryAuthenticate();
            setCode('');
            snackManager.snack('Multi-factor authentication disabled');
        } finally {
            setBusy(false);
        }
    };

    const regenerate = async () => {
        if (!requireElevation() || !code) return;
        setBusy(true);
        try {
            const response = await axios.post<{recoveryCodes: string[]}>(
                config.get('url') + 'current/user/mfa/recovery-codes',
                {code}
            );
            setRecoveryCodes(response.data.recoveryCodes);
            setCode('');
            await refresh();
        } finally {
            setBusy(false);
        }
    };

    return (
        <>
            <SurfaceCard
                title="Multi-factor Authentication"
                subtitle="Protect password sign-in with an authenticator app and recovery codes."
                action={<Security color="action" />}>
                <Stack spacing={2}>
                    <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                        <Typography sx={{flex: 1}}>
                            Authenticator verification
                        </Typography>
                        <Chip
                            size="small"
                            color={status?.enabled ? 'success' : 'default'}
                            label={status?.enabled ? 'Enabled' : 'Disabled'}
                        />
                    </Stack>
                    {currentUser.user.mfaRequired && !status?.enabled && (
                        <Alert severity="warning">
                            Multi-factor authentication is required by server policy.
                        </Alert>
                    )}
                    {status?.enabled ? (
                        <>
                            <TextField
                                label="Authenticator or recovery code"
                                value={code}
                                onChange={(event) => setCode(event.target.value)}
                                autoComplete="one-time-code"
                                helperText={
                                    (status.recoveryCodes || 0) +
                                    ' recovery code' +
                                    (status.recoveryCodes === 1 ? '' : 's') +
                                    ' remaining.'
                                }
                            />
                            <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                                <Button
                                    variant="outlined"
                                    disabled={busy || !code}
                                    onClick={() => void regenerate()}>
                                    Replace Recovery Codes
                                </Button>
                                <Button
                                    color="error"
                                    variant="outlined"
                                    disabled={busy || !code}
                                    onClick={() => void disable()}>
                                    Disable MFA
                                </Button>
                            </Stack>
                        </>
                    ) : (
                        <Button
                            variant="contained"
                            disabled={busy}
                            onClick={() => void beginSetup()}
                            sx={{alignSelf: 'flex-start'}}>
                            Set Up Authenticator
                        </Button>
                    )}
                </Stack>
            </SurfaceCard>

            <Dialog open={Boolean(setup)} onClose={() => !busy && setSetup(undefined)} fullWidth maxWidth="sm">
                <DialogTitle>Set Up Authenticator</DialogTitle>
                <DialogContent>
                    {setup && (
                        <Stack spacing={2} sx={{pt: 1}}>
                            <Typography>
                                Add this account to your authenticator app, then enter the current
                                six-digit code to verify setup.
                            </Typography>
                            <TextField
                                label="Setup key"
                                value={setup.secret}
                                slotProps={{htmlInput: {readOnly: true}}}
                                fullWidth
                            />
                            <TextField
                                label="Authenticator setup link"
                                value={setup.provisioningUri}
                                slotProps={{htmlInput: {readOnly: true}}}
                                fullWidth
                            />
                            <Alert severity="warning">
                                Save the recovery codes shown below before enabling MFA. They are
                                displayed only when generated.
                            </Alert>
                            <Typography
                                component="pre"
                                sx={{whiteSpace: 'pre-wrap', fontFamily: 'monospace'}}>
                                {(recoveryCodes || []).join('\n')}
                            </Typography>
                            <TextField
                                autoFocus
                                label="Verification code"
                                value={code}
                                onChange={(event) => setCode(event.target.value)}
                                autoComplete="one-time-code"
                            />
                        </Stack>
                    )}
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setSetup(undefined)} disabled={busy}>
                        Cancel
                    </Button>
                    <Button
                        variant="contained"
                        disabled={busy || !code}
                        onClick={() => void enable()}>
                        Verify and Enable
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog
                open={Boolean(recoveryCodes && !setup)}
                onClose={() => setRecoveryCodes(undefined)}
                fullWidth
                maxWidth="sm">
                <DialogTitle>New Recovery Codes</DialogTitle>
                <DialogContent>
                    <Alert severity="warning" sx={{mb: 2}}>
                        Store these recovery codes somewhere safe. They will not be shown again.
                    </Alert>
                    <Typography component="pre" sx={{whiteSpace: 'pre-wrap', fontFamily: 'monospace'}}>
                        {(recoveryCodes || []).join('\n')}
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setRecoveryCodes(undefined)}>Done</Button>
                </DialogActions>
            </Dialog>
        </>
    );
};


const PasskeySettings = () => {
    const {currentUser, elevateStore, snackManager} = useStores();
    const [items, setItems] = React.useState<IPasskey[]>([]);
    const [name, setName] = React.useState('');
    const [busy, setBusy] = React.useState(false);
    const [deleteItem, setDeleteItem] = React.useState<IPasskey>();

    const refresh = React.useCallback(async () => {
        const response = await axios.get<IPasskey[]>(config.get('url') + 'current/user/passkeys');
        setItems(response.data);
    }, []);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    const add = async () => {
        if (!elevateStore.elevated) {
            elevateStore.requestReauthentication();
            return;
        }
        setBusy(true);
        try {
            await createPasskey(name.trim() || 'Passkey');
            setName('');
            await refresh();
            await currentUser.tryAuthenticate();
            snackManager.snack('Passkey added');
        } finally {
            setBusy(false);
        }
    };

    const remove = async (item: IPasskey) => {
        await axios.delete(config.get('url') + 'current/user/passkeys/' + item.id);
        await refresh();
        await currentUser.tryAuthenticate();
        snackManager.snack('Passkey removed');
    };

    return (
        <>
            <SurfaceCard
                title="Passkeys"
                subtitle="Use a device passkey or security key for passwordless sign-in and identity confirmation."
                action={<Key color="action" />}>
                <Stack spacing={2}>
                    {items.length === 0 ? (
                        <Typography color="text.secondary">No passkeys are registered.</Typography>
                    ) : (
                        <Stack spacing={1}>
                            {items.map((item) => (
                                <Stack
                                    key={item.id}
                                    direction={{xs: 'column', sm: 'row'}}
                                    spacing={1}
                                    sx={{
                                        border: 1,
                                        borderColor: 'divider',
                                        borderRadius: 2,
                                        p: 1.5,
                                        justifyContent: 'space-between',
                                        alignItems: {sm: 'center'},
                                    }}>
                                    <Box>
                                        <Typography sx={{fontWeight: 700}}>{item.name}</Typography>
                                        <Typography variant="caption" color="text.secondary">
                                            Added {new Date(item.createdAt).toLocaleString()}
                                            {item.lastUsedAt
                                                ? ' · Last used ' +
                                                  new Date(item.lastUsedAt).toLocaleString()
                                                : ''}
                                        </Typography>
                                    </Box>
                                    <Button
                                        size="small"
                                        color="error"
                                        onClick={() => setDeleteItem(item)}>
                                        Remove
                                    </Button>
                                </Stack>
                            ))}
                        </Stack>
                    )}
                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                        <TextField
                            label="Passkey name"
                            value={name}
                            onChange={(event) => setName(event.target.value)}
                            placeholder="Laptop, phone, security key"
                            fullWidth
                        />
                        <Button
                            variant="contained"
                            disabled={busy}
                            onClick={() => void add()}>
                            Add Passkey
                        </Button>
                    </Stack>
                </Stack>
            </SurfaceCard>
            {deleteItem && (
                <ConfirmDialog
                    title="Remove Passkey?"
                    text={'Remove "' + deleteItem.name + '" from this account?'}
                    requireElevated
                    fClose={() => setDeleteItem(undefined)}
                    fOnSubmit={() => void remove(deleteItem)}
                />
            )}
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
