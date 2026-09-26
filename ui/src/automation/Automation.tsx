import React from 'react';
import axios from 'axios';
import {
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControlLabel,
    MenuItem,
    Stack,
    Switch,
    TextField,
    Typography,
} from '@mui/material';
import Add from '@mui/icons-material/Add';
import Delete from '@mui/icons-material/Delete';
import Refresh from '@mui/icons-material/Refresh';
import Schedule from '@mui/icons-material/Schedule';
import TrendingUp from '@mui/icons-material/TrendingUp';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import * as config from '../config';
import {useStores} from '../stores';
import {IEscalationRule, IScheduledNotification} from '../types';

const api = (path: string) => config.get('url') + path;
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';

const channelName = (
    channels: Array<{id: number; name: string}>,
    id: number
): string => channels.find((channel) => channel.id === id)?.name || 'Unknown Channel';

const Automation = () => {
    const {appStore, snackManager} = useStores();
    const [schedules, setSchedules] = React.useState<IScheduledNotification[]>([]);
    const [escalations, setEscalations] = React.useState<IEscalationRule[]>([]);
    const [scheduleEdit, setScheduleEdit] =
        React.useState<IScheduledNotification | null | undefined>();
    const [escalationEdit, setEscalationEdit] =
        React.useState<IEscalationRule | null | undefined>();
    const [loading, setLoading] = React.useState(true);
    const [confirm, setConfirm] = React.useState<{
        title: string;
        text: string;
        action: () => void;
    }>();

    const refresh = React.useCallback(async () => {
        setLoading(true);
        try {
            await appStore.refresh();
            const [scheduleResponse, escalationResponse] = await Promise.all([
                axios.get<IScheduledNotification[]>(api('automation/schedule')),
                axios.get<IEscalationRule[]>(api('automation/escalation')),
            ]);
            setSchedules(scheduleResponse.data);
            setEscalations(escalationResponse.data);
        } finally {
            setLoading(false);
        }
    }, [appStore]);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    const channels = appStore.getItems();

    return (
        <DefaultPage
            title="Automation"
            description="Schedule notifications and escalate messages that need attention."
            rightControl={
                <Button startIcon={<Refresh />} onClick={() => void refresh()} disabled={loading}>
                    Refresh
                </Button>
            }>
            <SurfaceCard
                title="Scheduled Notifications"
                subtitle="Send one-time or recurring notifications to a Channel."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setScheduleEdit(null)}>
                        Add Schedule
                    </Button>
                }>
                {schedules.length === 0 ? (
                    <Typography color="text.secondary">
                        No scheduled notifications have been created.
                    </Typography>
                ) : (
                    <Stack spacing={1}>
                        {schedules.map((item) => (
                            <AutomationRow
                                key={item.id}
                                icon={<Schedule />}
                                title={item.name}
                                enabled={item.enabled}
                                subtitle={scheduleSummary(item, channels)}
                                detail={
                                    item.nextRunAt
                                        ? 'Next: ' + new Date(item.nextRunAt).toLocaleString()
                                        : item.enabled
                                          ? 'Waiting for a future run time'
                                          : 'Disabled'
                                }
                                onEdit={() => setScheduleEdit(item)}
                                onDelete={() =>
                                    setConfirm({
                                        title: 'Delete Schedule?',
                                        text: 'This scheduled notification will no longer run.',
                                        action: () => {
                                            void (async () => {
                                                await axios.delete(api('automation/schedule/' + item.id));
                                                await refresh();
                                                snackManager.snack('Schedule deleted');
                                            })();
                                        },
                                    })
                                }
                            />
                        ))}
                    </Stack>
                )}
            </SurfaceCard>

            <SurfaceCard
                title="Escalations"
                subtitle="Forward important messages when nobody acknowledges them in time."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setEscalationEdit(null)}>
                        Add Escalation
                    </Button>
                }>
                {escalations.length === 0 ? (
                    <Typography color="text.secondary">No escalation rules have been created.</Typography>
                ) : (
                    <Stack spacing={1}>
                        {escalations.map((item) => (
                            <AutomationRow
                                key={item.id}
                                icon={<TrendingUp />}
                                title={item.name}
                                enabled={item.enabled}
                                subtitle={
                                    channelName(channels, item.sourceApplicationId) +
                                    ' → ' +
                                    channelName(channels, item.targetApplicationId)
                                }
                                detail={
                                    'After ' +
                                    item.delayMinutes +
                                    ' minute' +
                                    (item.delayMinutes === 1 ? '' : 's') +
                                    ' if priority is ' +
                                    item.minPriority +
                                    ' or higher and the message is still unacknowledged.'
                                }
                                onEdit={() => setEscalationEdit(item)}
                                onDelete={() =>
                                    setConfirm({
                                        title: 'Delete Escalation?',
                                        text: 'Pending and future escalation behavior for this rule will stop.',
                                        action: () => {
                                            void (async () => {
                                                await axios.delete(api('automation/escalation/' + item.id));
                                                await refresh();
                                                snackManager.snack('Escalation deleted');
                                            })();
                                        },
                                    })
                                }
                            />
                        ))}
                    </Stack>
                )}
            </SurfaceCard>

            {confirm && (
                <ConfirmDialog
                    title={confirm.title}
                    text={confirm.text}
                    fClose={() => setConfirm(undefined)}
                    fOnSubmit={confirm.action}
                />
            )}

            {scheduleEdit !== undefined && (
                <ScheduleDialog
                    item={scheduleEdit}
                    channels={channels}
                    onClose={() => setScheduleEdit(undefined)}
                    onSaved={async () => {
                        setScheduleEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {escalationEdit !== undefined && (
                <EscalationDialog
                    item={escalationEdit}
                    channels={channels}
                    onClose={() => setEscalationEdit(undefined)}
                    onSaved={async () => {
                        setEscalationEdit(undefined);
                        await refresh();
                    }}
                />
            )}
        </DefaultPage>
    );
};

const AutomationRow = ({
    icon,
    title,
    subtitle,
    detail,
    enabled,
    onEdit,
    onDelete,
}: {
    icon: React.ReactNode;
    title: string;
    subtitle: string;
    detail: string;
    enabled: boolean;
    onEdit: VoidFunction;
    onDelete: VoidFunction;
}) => (
    <Box sx={{p: 1.5, border: 1, borderColor: 'divider', borderRadius: 2}}>
        <Stack
            direction={{xs: 'column', sm: 'row'}}
            spacing={1.5}
            sx={{alignItems: {sm: 'flex-start'}, justifyContent: 'space-between'}}>
            <Stack direction="row" spacing={1.25} sx={{minWidth: 0}}>
                <Box sx={{pt: 0.25, color: 'text.secondary'}}>{icon}</Box>
                <Box sx={{minWidth: 0}}>
                    <Stack
                        direction="row"
                        spacing={0.75}
                        sx={{alignItems: 'center', flexWrap: 'wrap'}}>
                        <Typography sx={{fontWeight: 700}}>{title}</Typography>
                        <Chip
                            size="small"
                            color={enabled ? 'success' : 'default'}
                            variant={enabled ? 'filled' : 'outlined'}
                            label={enabled ? 'Enabled' : 'Disabled'}
                        />
                    </Stack>
                    <Typography variant="body2" color="text.secondary">
                        {subtitle}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                        {detail}
                    </Typography>
                </Box>
            </Stack>
            <Stack direction="row" spacing={0.5}>
                <Button size="small" onClick={onEdit}>
                    Edit
                </Button>
                <Button
                    size="small"
                    color="error"
                    startIcon={<Delete />}
                    onClick={onDelete}>
                    Delete
                </Button>
            </Stack>
        </Stack>
    </Box>
);

const ChannelSelect = ({
    label = 'Channel',
    value,
    onChange,
    channels,
}: {
    label?: string;
    value: number;
    onChange: (value: number) => void;
    channels: Array<{id: number; name: string}>;
}) => (
    <TextField
        select
        label={label}
        value={value || ''}
        onChange={(event) => onChange(Number(event.target.value))}
        required
        fullWidth>
        {channels.map((channel) => (
            <MenuItem key={channel.id} value={channel.id}>
                {channel.name}
            </MenuItem>
        ))}
    </TextField>
);

const scheduleSummary = (
    item: IScheduledNotification,
    channels: Array<{id: number; name: string}>
): string => {
    const channel = channelName(channels, item.applicationId);
    switch (item.scheduleType) {
        case 'once':
            return channel + ' · One time';
        case 'hourly':
            return channel + ' · Hourly at minute ' + item.minute;
        case 'daily':
            return (
                channel +
                ' · Daily at ' +
                two(item.hour) +
                ':' +
                two(item.minute) +
                ' ' +
                item.timezone
            );
        case 'weekly':
            return (
                channel +
                ' · ' +
                (weekdayNames[item.weekday] || 'Weekly') +
                ' at ' +
                two(item.hour) +
                ':' +
                two(item.minute) +
                ' ' +
                item.timezone
            );
        default:
            return channel;
    }
};

const weekdayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
const two = (value: number) => String(value).padStart(2, '0');

const ScheduleDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: IScheduledNotification | null;
    channels: Array<{id: number; name: string}>;
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [title, setTitle] = React.useState(item?.title || '');
    const [message, setMessage] = React.useState(item?.message || '');
    const [priority, setPriority] = React.useState(item?.priority || 0);
    const [scheduleType, setScheduleType] = React.useState<
        IScheduledNotification['scheduleType']
    >(item?.scheduleType || 'once');
    const [runAt, setRunAt] = React.useState(
        item?.runAt
            ? localInputValue(item.runAt)
            : localInputValue(new Date(Date.now() + 3600000).toISOString())
    );
    const [hour, setHour] = React.useState(item?.hour ?? 9);
    const [minute, setMinute] = React.useState(item?.minute ?? 0);
    const [weekday, setWeekday] = React.useState(item?.weekday ?? 1);
    const [timezoneValue, setTimezoneValue] = React.useState(item?.timezone || timezone);
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const [saving, setSaving] = React.useState(false);

    const save = async () => {
        setSaving(true);
        try {
            const payload = {
                name,
                applicationId,
                title,
                message,
                priority,
                scheduleType,
                runAt: scheduleType === 'once' ? new Date(runAt).toISOString() : null,
                hour,
                minute,
                weekday,
                timezone: timezoneValue,
                enabled,
            };
            if (item) {
                await axios.put(api('automation/schedule/' + item.id), payload);
            } else {
                await axios.post(api('automation/schedule'), payload);
            }
            await onSaved();
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit Schedule' : 'Add Schedule'}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                    <ChannelSelect value={applicationId} onChange={setApplicationId} channels={channels} />
                    <TextField label="Title" value={title} onChange={(e) => setTitle(e.target.value)} />
                    <TextField
                        label="Message"
                        value={message}
                        onChange={(e) => setMessage(e.target.value)}
                        multiline
                        minRows={3}
                        required
                    />
                    <TextField
                        label="Priority"
                        type="number"
                        value={priority}
                        onChange={(e) => setPriority(Number(e.target.value))}
                    />
                    <TextField
                        select
                        label="Schedule"
                        value={scheduleType}
                        onChange={(e) =>
                            setScheduleType(e.target.value as IScheduledNotification['scheduleType'])
                        }>
                        <MenuItem value="once">One time</MenuItem>
                        <MenuItem value="hourly">Hourly</MenuItem>
                        <MenuItem value="daily">Daily</MenuItem>
                        <MenuItem value="weekly">Weekly</MenuItem>
                    </TextField>

                    {scheduleType === 'once' && (
                        <TextField
                            type="datetime-local"
                            label="Send at"
                            value={runAt}
                            onChange={(e) => setRunAt(e.target.value)}
                            slotProps={{inputLabel: {shrink: true}}}
                        />
                    )}
                    {scheduleType === 'hourly' && (
                        <TextField
                            type="number"
                            label="Minute of the hour"
                            value={minute}
                            onChange={(e) => setMinute(Number(e.target.value))}
                            slotProps={{htmlInput: {min: 0, max: 59}}}
                        />
                    )}
                    {(scheduleType === 'daily' || scheduleType === 'weekly') && (
                        <Stack direction={{xs: 'column', sm: 'row'}} spacing={2}>
                            {scheduleType === 'weekly' && (
                                <TextField
                                    select
                                    label="Day"
                                    value={weekday}
                                    onChange={(e) => setWeekday(Number(e.target.value))}
                                    fullWidth>
                                    {weekdayNames.map((day, index) => (
                                        <MenuItem key={day} value={index}>
                                            {day}
                                        </MenuItem>
                                    ))}
                                </TextField>
                            )}
                            <TextField
                                type="number"
                                label="Hour"
                                value={hour}
                                onChange={(e) => setHour(Number(e.target.value))}
                                slotProps={{htmlInput: {min: 0, max: 23}}}
                                fullWidth
                            />
                            <TextField
                                type="number"
                                label="Minute"
                                value={minute}
                                onChange={(e) => setMinute(Number(e.target.value))}
                                slotProps={{htmlInput: {min: 0, max: 59}}}
                                fullWidth
                            />
                        </Stack>
                    )}
                    {scheduleType !== 'once' && (
                        <TextField
                            label="Timezone"
                            value={timezoneValue}
                            onChange={(e) => setTimezoneValue(e.target.value)}
                            helperText="Use an IANA timezone such as America/New_York."
                        />
                    )}
                    <FormControlLabel
                        control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />}
                        label="Enabled"
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
                <Button
                    variant="contained"
                    disabled={saving || !name || !applicationId || !message}
                    onClick={() => void save()}>
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
};

const EscalationDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: IEscalationRule | null;
    channels: Array<{id: number; name: string}>;
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [sourceApplicationId, setSourceApplicationId] = React.useState(
        item?.sourceApplicationId || 0
    );
    const [targetApplicationId, setTargetApplicationId] = React.useState(
        item?.targetApplicationId || 0
    );
    const [minPriority, setMinPriority] = React.useState(item?.minPriority ?? 4);
    const [delayMinutes, setDelayMinutes] = React.useState(item?.delayMinutes ?? 15);
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const [saving, setSaving] = React.useState(false);

    const save = async () => {
        setSaving(true);
        try {
            const payload = {
                name,
                sourceApplicationId,
                targetApplicationId,
                minPriority,
                delayMinutes,
                enabled,
            };
            if (item) {
                await axios.put(api('automation/escalation/' + item.id), payload);
            } else {
                await axios.post(api('automation/escalation'), payload);
            }
            await onSaved();
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit Escalation' : 'Add Escalation'}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                    <ChannelSelect
                        label="Watch Channel"
                        value={sourceApplicationId}
                        onChange={setSourceApplicationId}
                        channels={channels}
                    />
                    <ChannelSelect
                        label="Escalate to"
                        value={targetApplicationId}
                        onChange={setTargetApplicationId}
                        channels={channels.filter((channel) => channel.id !== sourceApplicationId)}
                    />
                    <TextField
                        type="number"
                        label="Minimum priority"
                        value={minPriority}
                        onChange={(e) => setMinPriority(Number(e.target.value))}
                    />
                    <TextField
                        type="number"
                        label="Wait before escalating"
                        value={delayMinutes}
                        onChange={(e) => setDelayMinutes(Number(e.target.value))}
                        helperText="Minutes without acknowledgement before the message is escalated."
                        slotProps={{htmlInput: {min: 1, max: 10080}}}
                    />
                    <FormControlLabel
                        control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />}
                        label="Enabled"
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
                <Button
                    variant="contained"
                    disabled={
                        saving ||
                        !name ||
                        !sourceApplicationId ||
                        !targetApplicationId ||
                        sourceApplicationId === targetApplicationId
                    }
                    onClick={() => void save()}>
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
};

const localInputValue = (iso: string): string => {
    const date = new Date(iso);
    const offset = date.getTimezoneOffset() * 60000;
    return new Date(date.getTime() - offset).toISOString().slice(0, 16);
};

export default Automation;
