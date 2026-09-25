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
import {IDigestPolicy, IQuietHoursPolicy} from '../types';

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
