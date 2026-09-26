import React from 'react';
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
    FormControlLabel,
    MenuItem,
    Stack,
    Switch,
    TextField,
    Typography,
} from '@mui/material';
import Add from '@mui/icons-material/Add';
import ContentCopy from '@mui/icons-material/ContentCopy';
import Delete from '@mui/icons-material/Delete';
import Refresh from '@mui/icons-material/Refresh';
import Webhook from '@mui/icons-material/Webhook';
import Sensors from '@mui/icons-material/Sensors';
import Home from '@mui/icons-material/Home';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import * as config from '../config';
import {useStores} from '../stores';
import {
    IHomeAssistantIntegration,
    IIntegrationEvent,
    IIntegrationStatus,
    IMQTTIntegration,
    IWebhookRoute,
} from '../types';

const api = (path: string) => `${config.get('url')}${path}`;

const channelName = (
    channels: Array<{id: number; name: string}>,
    id: number
): string => channels.find((channel) => channel.id === id)?.name || 'Unknown Channel';

const Integrations = () => {
    const {appStore, snackManager} = useStores();
    const [webhooks, setWebhooks] = React.useState<IWebhookRoute[]>([]);
    const [mqtt, setMqtt] = React.useState<IMQTTIntegration[]>([]);
    const [homeAssistant, setHomeAssistant] = React.useState<IHomeAssistantIntegration[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [webhookEdit, setWebhookEdit] = React.useState<IWebhookRoute | null | undefined>();
    const [mqttEdit, setMqttEdit] = React.useState<IMQTTIntegration | null | undefined>();
    const [homeAssistantEdit, setHomeAssistantEdit] =
        React.useState<IHomeAssistantIntegration | null | undefined>();
    const [history, setHistory] = React.useState<
        {kind: string; id: number; title: string} | undefined
    >();
    const [confirmAction, setConfirmAction] = React.useState<
        {title: string; text: string; run: () => Promise<void>} | undefined
    >();

    const refresh = React.useCallback(async () => {
        setLoading(true);
        try {
            await appStore.refresh();
            const [webhookResponse, mqttResponse, homeAssistantResponse] = await Promise.all([
                axios.get<IWebhookRoute[]>(api('integration/webhook')),
                axios.get<IMQTTIntegration[]>(api('integration/mqtt')),
                axios.get<IHomeAssistantIntegration[]>(api('integration/home-assistant')),
            ]);
            setWebhooks(webhookResponse.data);
            setMqtt(mqttResponse.data);
            setHomeAssistant(homeAssistantResponse.data);
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
            title="Integrations"
            description="Connect external systems directly to Gotify MU."
            rightControl={
                <Button startIcon={<Refresh />} onClick={() => void refresh()} disabled={loading}>
                    Refresh
                </Button>
            }>
            <Alert severity="info">
                Integration credentials are never shown again after they are saved. Leave a password
                or access token blank while editing to keep the current value.
            </Alert>

            <SurfaceCard
                title="Webhooks"
                subtitle="Create inbound URLs that route JSON or plain text into a Channel."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setWebhookEdit(null)}>
                        Add Webhook
                    </Button>
                }>
                <IntegrationList
                    empty="No webhook routes have been created."
                    items={webhooks.map((item) => ({
                        id: item.id,
                        icon: <Webhook />,
                        title: item.name,
                        subtitle: channelName(channels, item.applicationId),
                        enabled: item.enabled,
                        status: item.status,
                        details: (
                            <Stack spacing={0.75}>
                                <Typography variant="body2" sx={{wordBreak: 'break-all'}}>
                                    {api(item.path.replace(/^\//, ''))}
                                </Typography>
                                <Stack direction="row" spacing={1} useFlexGap sx={{flexWrap: 'wrap'}}>
                                    <Button
                                        size="small"
                                        startIcon={<ContentCopy />}
                                        onClick={() => {
                                            void navigator.clipboard.writeText(
                                                api(item.path.replace(/^\//, ''))
                                            );
                                            snackManager.snack('Webhook URL copied');
                                        }}>
                                        Copy URL
                                    </Button>
                                    <Button
                                        size="small"
                                        startIcon={<Refresh />}
                                        onClick={() =>
                                            setConfirmAction({
                                                title: 'Regenerate Webhook URL',
                                                text: 'Regenerate this Webhook URL? The current URL will stop working immediately.',
                                                run: async () => {
                                                    await axios.post(
                                                        api(`integration/webhook/${item.id}/regenerate`)
                                                    );
                                                    await refresh();
                                                    snackManager.snack('Webhook URL regenerated');
                                                },
                                            })
                                        }>
                                        Regenerate URL
                                    </Button>
                                    <Button
                                        size="small"
                                        onClick={() =>
                                            setHistory({kind: 'webhook', id: item.id, title: item.name})
                                        }>
                                        Activity
                                    </Button>
                                </Stack>
                            </Stack>
                        ),
                        onEdit: () => setWebhookEdit(item),
                        onDelete: () => {
                            setConfirmAction({
                                title: 'Delete Webhook',
                                text: `Delete ${item.name}? Its URL will stop working immediately.`,
                                run: async () => {
                                    await axios.delete(api(`integration/webhook/${item.id}`));
                                    await refresh();
                                    snackManager.snack('Webhook deleted');
                                },
                            });
                            return Promise.resolve();
                        },
                    }))}
                />
            </SurfaceCard>

            <SurfaceCard
                title="MQTT"
                subtitle="Subscribe to broker topics and route received messages into Channels."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setMqttEdit(null)}>
                        Add MQTT Connection
                    </Button>
                }>
                <IntegrationList
                    empty="No MQTT connections have been configured."
                    items={mqtt.map((item) => ({
                        id: item.id,
                        icon: <Sensors />,
                        title: item.name,
                        subtitle: `${item.brokerUrl} · ${item.topic} · ${channelName(
                            channels,
                            item.applicationId
                        )}`,
                        enabled: item.enabled,
                        status: item.status,
                        details: (
                            <Stack direction="row" spacing={1} useFlexGap sx={{flexWrap: 'wrap'}}>
                                <Button
                                    size="small"
                                    onClick={async () => {
                                        await axios.post(api(`integration/mqtt/${item.id}/test`));
                                        await refresh();
                                        snackManager.snack('MQTT connection test succeeded');
                                    }}>
                                    Test Connection
                                </Button>
                                <Button
                                    size="small"
                                    onClick={() =>
                                        setHistory({kind: 'mqtt', id: item.id, title: item.name})
                                    }>
                                    Activity
                                </Button>
                            </Stack>
                        ),
                        onEdit: () => setMqttEdit(item),
                        onDelete: () => {
                            setConfirmAction({
                                title: 'Delete MQTT Connection',
                                text: `Delete ${item.name}? Gotify MU will stop subscribing to this broker/topic.`,
                                run: async () => {
                                    await axios.delete(api(`integration/mqtt/${item.id}`));
                                    await refresh();
                                    snackManager.snack('MQTT connection deleted');
                                },
                            });
                            return Promise.resolve();
                        },
                    }))}
                />
            </SurfaceCard>

            <SurfaceCard
                title="Home Assistant"
                subtitle="Subscribe directly to Home Assistant events and send them into a Channel."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setHomeAssistantEdit(null)}>
                        Add Home Assistant
                    </Button>
                }>
                <IntegrationList
                    empty="No Home Assistant connections have been configured."
                    items={homeAssistant.map((item) => ({
                        id: item.id,
                        icon: <Home />,
                        title: item.name,
                        subtitle: `${item.baseUrl} · ${item.eventType || 'All events'} · ${channelName(
                            channels,
                            item.applicationId
                        )}`,
                        enabled: item.enabled,
                        status: item.status,
                        details: (
                            <Stack direction="row" spacing={1} useFlexGap sx={{flexWrap: 'wrap'}}>
                            <Button
                                size="small"
                                onClick={async () => {
                                    await axios.post(
                                        api('integration/home-assistant/' + item.id + '/event'),
                                        {
                                            eventType: 'gotify_mu_test',
                                            data: {message: 'Gotify MU connection test'},
                                        }
                                    );
                                    snackManager.snack('Test event sent to Home Assistant');
                                }}>
                                Send Test Event
                            </Button>
                            <Button
                                size="small"
                                onClick={() =>
                                    setHistory({kind: 'home-assistant', id: item.id, title: item.name})
                                }>
                                Activity
                            </Button>
                            </Stack>
                        ),
                        onEdit: () => setHomeAssistantEdit(item),
                        onDelete: () => {
                            setConfirmAction({
                                title: 'Delete Home Assistant Connection',
                                text: `Delete ${item.name}? Gotify MU will disconnect from this Home Assistant instance.`,
                                run: async () => {
                                    await axios.delete(api(`integration/home-assistant/${item.id}`));
                                    await refresh();
                                    snackManager.snack('Home Assistant connection deleted');
                                },
                            });
                            return Promise.resolve();
                        },
                    }))}
                />
            </SurfaceCard>

            {webhookEdit !== undefined && (
                <WebhookDialog
                    item={webhookEdit}
                    channels={channels}
                    onClose={() => setWebhookEdit(undefined)}
                    onSaved={async () => {
                        setWebhookEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {mqttEdit !== undefined && (
                <MQTTDialog
                    item={mqttEdit}
                    channels={channels}
                    onClose={() => setMqttEdit(undefined)}
                    onSaved={async () => {
                        setMqttEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {homeAssistantEdit !== undefined && (
                <HomeAssistantDialog
                    item={homeAssistantEdit}
                    channels={channels}
                    onClose={() => setHomeAssistantEdit(undefined)}
                    onSaved={async () => {
                        setHomeAssistantEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {history && (
                <IntegrationHistoryDialog
                    kind={history.kind}
                    id={history.id}
                    title={history.title}
                    onClose={() => setHistory(undefined)}
                />
            )}
            {confirmAction && (
                <ConfirmDialog
                    title={confirmAction.title}
                    text={confirmAction.text}
                    fClose={() => setConfirmAction(undefined)}
                    fOnSubmit={async () => {
                        const action = confirmAction;
                        setConfirmAction(undefined);
                        await action.run();
                    }}
                />
            )}
        </DefaultPage>
    );
};

interface ListItem {
    id: number;
    icon: React.ReactNode;
    title: string;
    subtitle: string;
    enabled: boolean;
    status?: IIntegrationStatus;
    details?: React.ReactNode;
    onEdit: VoidFunction;
    onDelete: () => Promise<void>;
}

const IntegrationList = ({items, empty}: {items: ListItem[]; empty: string}) => {
    if (items.length === 0) {
        return <Typography color="text.secondary">{empty}</Typography>;
    }

    return (
        <Stack spacing={1}>
            {items.map((item) => (
                <Box
                    key={item.id}
                    sx={{
                        p: 1.5,
                        border: 1,
                        borderColor: 'divider',
                        borderRadius: 2,
                    }}>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={1.5}
                        sx={{alignItems: {sm: 'flex-start'}, justifyContent: 'space-between'}}>
                        <Stack direction="row" spacing={1.25} sx={{minWidth: 0}}>
                            <Box sx={{pt: 0.25, color: 'text.secondary'}}>{item.icon}</Box>
                            <Box sx={{minWidth: 0}}>
                                <Stack
                                    direction="row"
                                    spacing={0.75}
                                    sx={{alignItems: 'center', flexWrap: 'wrap'}}>
                                    <Typography sx={{fontWeight: 700}}>{item.title}</Typography>
                                    <Chip
                                        size="small"
                                        color={item.enabled ? 'success' : 'default'}
                                        variant={item.enabled ? 'filled' : 'outlined'}
                                        label={item.enabled ? 'Enabled' : 'Disabled'}
                                    />
                                    {item.status && <IntegrationStatusChip status={item.status} />}
                                </Stack>
                                <Typography variant="body2" color="text.secondary">
                                    {item.subtitle}
                                </Typography>
                                {item.status?.message && (
                                    <Typography variant="caption" color="text.secondary">
                                        {item.status.message}
                                        {item.status.lastEventAt
                                            ? ` · Last activity ${new Date(item.status.lastEventAt).toLocaleString()}`
                                            : ''}
                                    </Typography>
                                )}
                                {item.details && <Box sx={{mt: 1}}>{item.details}</Box>}
                            </Box>
                        </Stack>
                        <Stack direction="row" spacing={0.5}>
                            <Button size="small" onClick={item.onEdit}>
                                Edit
                            </Button>
                            <Button
                                size="small"
                                color="error"
                                startIcon={<Delete />}
                                onClick={() => void item.onDelete()}>
                                Delete
                            </Button>
                        </Stack>
                    </Stack>
                </Box>
            ))}
        </Stack>
    );
};

const IntegrationStatusChip = ({status}: {status: IIntegrationStatus}) => {
    const state = status.state.toLowerCase();
    const color =
        state === 'connected' || state === 'ready'
            ? 'success'
            : state === 'error'
              ? 'error'
              : state === 'reconnecting' || state === 'connecting'
                ? 'warning'
                : 'default';
    return <Chip size="small" color={color} variant="outlined" label={status.state} />;
};

const IntegrationHistoryDialog = ({
    kind,
    id,
    title,
    onClose,
}: {
    kind: string;
    id: number;
    title: string;
    onClose: VoidFunction;
}) => {
    const [items, setItems] = React.useState<IIntegrationEvent[]>([]);

    React.useEffect(() => {
        void axios
            .get<IIntegrationEvent[]>(api(`integration/${kind}/${id}/events?limit=100`))
            .then((response) => setItems(response.data));
    }, [kind, id]);

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="md">
            <DialogTitle>{title} Activity</DialogTitle>
            <DialogContent>
                {items.length === 0 ? (
                    <Typography color="text.secondary">No activity has been recorded yet.</Typography>
                ) : (
                    <Stack spacing={1}>
                        {items.map((item) => (
                            <Box
                                key={item.id}
                                sx={{p: 1.25, border: 1, borderColor: 'divider', borderRadius: 2}}>
                                <Stack
                                    direction="row"
                                    spacing={1}
                                    sx={{alignItems: 'center', justifyContent: 'space-between'}}>
                                    <Typography sx={{fontWeight: 700}}>
                                        {item.event.replace(/_/g, ' ')}
                                    </Typography>
                                    <Chip
                                        size="small"
                                        color={item.level === 'error' ? 'error' : 'default'}
                                        label={item.level}
                                    />
                                </Stack>
                                <Typography variant="body2">{item.message}</Typography>
                                <Typography variant="caption" color="text.secondary">
                                    {new Date(item.createdAt).toLocaleString()}
                                </Typography>
                            </Box>
                        ))}
                    </Stack>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Close</Button>
            </DialogActions>
        </Dialog>
    );
};

const ChannelSelect = ({
    value,
    onChange,
    channels,
}: {
    value: number;
    onChange: (value: number) => void;
    channels: Array<{id: number; name: string}>;
}) => (
    <TextField
        select
        label="Channel"
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

const WebhookDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: IWebhookRoute | null;
    channels: Array<{id: number; name: string}>;
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const [titleField, setTitleField] = React.useState(item?.titleField || 'title');
    const [messageField, setMessageField] = React.useState(item?.messageField || 'message');
    const [priorityField, setPriorityField] = React.useState(item?.priorityField || 'priority');
    const [defaultTitle, setDefaultTitle] = React.useState(item?.defaultTitle || '');
    const [defaultPriority, setDefaultPriority] = React.useState(item?.defaultPriority || 0);
    const [allowedCidrs, setAllowedCidrs] = React.useState((item?.allowedCidrs || []).join('\n'));
    const [requireSignature, setRequireSignature] = React.useState(item?.requireSignature ?? false);
    const [signingSecret, setSigningSecret] = React.useState('');
    const [replayWindowSeconds, setReplayWindowSeconds] = React.useState(
        item?.replayWindowSeconds || 300
    );
    const [saving, setSaving] = React.useState(false);

    const save = async () => {
        setSaving(true);
        try {
            const payload = {
                name,
                applicationId,
                enabled,
                titleField,
                messageField,
                priorityField,
                defaultTitle,
                defaultPriority,
                allowedCidrs: allowedCidrs
                    .split(/[\n,]+/)
                    .map((value) => value.trim())
                    .filter(Boolean),
                requireSignature,
                signingSecret,
                replayWindowSeconds,
            };
            if (item) {
                await axios.put(api(`integration/webhook/${item.id}`), payload);
            } else {
                await axios.post(api('integration/webhook'), payload);
            }
            await onSaved();
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit Webhook' : 'Add Webhook'}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                    <ChannelSelect value={applicationId} onChange={setApplicationId} channels={channels} />
                    <TextField
                        label="Title field"
                        value={titleField}
                        onChange={(e) => setTitleField(e.target.value)}
                        helperText="JSON field path, for example alert.title"
                    />
                    <TextField
                        label="Message field"
                        value={messageField}
                        onChange={(e) => setMessageField(e.target.value)}
                        helperText="JSON field path, for example alert.message"
                    />
                    <TextField
                        label="Priority field"
                        value={priorityField}
                        onChange={(e) => setPriorityField(e.target.value)}
                    />
                    <TextField
                        label="Default title"
                        value={defaultTitle}
                        onChange={(e) => setDefaultTitle(e.target.value)}
                    />
                    <TextField
                        label="Default priority"
                        type="number"
                        value={defaultPriority}
                        onChange={(e) => setDefaultPriority(Number(e.target.value))}
                    />
                    <TextField
                        label="Allowed source networks"
                        value={allowedCidrs}
                        onChange={(e) => setAllowedCidrs(e.target.value)}
                        multiline
                        minRows={2}
                        placeholder={'192.0.2.0/24\n2001:db8::/32'}
                        helperText="Optional. One CIDR per line. Leave blank to allow any source."
                    />
                    <FormControlLabel
                        control={
                            <Switch
                                checked={requireSignature}
                                onChange={(e) => setRequireSignature(e.target.checked)}
                            />
                        }
                        label="Require signed requests"
                    />
                    {requireSignature && (
                        <>
                            <TextField
                                label={item?.signatureConfigured ? 'New signing secret' : 'Signing secret'}
                                type="password"
                                value={signingSecret}
                                onChange={(e) => setSigningSecret(e.target.value)}
                                helperText={
                                    item?.signatureConfigured
                                        ? 'Leave blank to keep the current signing secret.'
                                        : 'At least 16 characters.'
                                }
                            />
                            <TextField
                                label="Replay window"
                                type="number"
                                value={replayWindowSeconds}
                                onChange={(e) => setReplayWindowSeconds(Number(e.target.value))}
                                helperText="Maximum age in seconds for signed Webhook requests."
                                slotProps={{htmlInput: {min: 1, max: 3600}}}
                            />
                        </>
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
                    disabled={
                        saving ||
                        !name ||
                        !applicationId ||
                        (requireSignature && !item?.signatureConfigured && signingSecret.length < 16)
                    }
                    onClick={() => void save()}>
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
};

const MQTTDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: IMQTTIntegration | null;
    channels: Array<{id: number; name: string}>;
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [brokerUrl, setBrokerUrl] = React.useState(item?.brokerUrl || 'mqtt://');
    const [clientId, setClientId] = React.useState(item?.clientId || '');
    const [username, setUsername] = React.useState(item?.username || '');
    const [password, setPassword] = React.useState('');
    const [topic, setTopic] = React.useState(item?.topic || '');
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const [saving, setSaving] = React.useState(false);

    const save = async () => {
        setSaving(true);
        try {
            const payload = {name, applicationId, brokerUrl, clientId, username, password, topic, enabled};
            if (item) {
                await axios.put(api(`integration/mqtt/${item.id}`), payload);
            } else {
                await axios.post(api('integration/mqtt'), payload);
            }
            await onSaved();
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit MQTT Connection' : 'Add MQTT Connection'}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                    <ChannelSelect value={applicationId} onChange={setApplicationId} channels={channels} />
                    <TextField
                        label="Broker URL"
                        value={brokerUrl}
                        onChange={(e) => setBrokerUrl(e.target.value)}
                        placeholder="mqtt://192.168.1.10:1883"
                        required
                    />
                    <TextField
                        label="Topic"
                        value={topic}
                        onChange={(e) => setTopic(e.target.value)}
                        placeholder="home/alerts/#"
                        required
                    />
                    <TextField label="Client ID" value={clientId} onChange={(e) => setClientId(e.target.value)} />
                    <TextField label="Username" value={username} onChange={(e) => setUsername(e.target.value)} />
                    <TextField
                        label={item?.passwordConfigured ? 'New password' : 'Password'}
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        helperText={item?.passwordConfigured ? 'Leave blank to keep the current password.' : ''}
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
                    disabled={saving || !name || !applicationId || !brokerUrl || !topic}
                    onClick={() => void save()}>
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
};

const HomeAssistantDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: IHomeAssistantIntegration | null;
    channels: Array<{id: number; name: string}>;
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || 'Home Assistant');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [baseUrl, setBaseUrl] = React.useState(item?.baseUrl || 'http://');
    const [token, setToken] = React.useState('');
    const [eventType, setEventType] = React.useState(item?.eventType || '');
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const [saving, setSaving] = React.useState(false);

    const save = async () => {
        setSaving(true);
        try {
            const payload = {name, applicationId, baseUrl, token, eventType, enabled};
            if (item) {
                await axios.put(api(`integration/home-assistant/${item.id}`), payload);
            } else {
                await axios.post(api('integration/home-assistant'), payload);
            }
            await onSaved();
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit Home Assistant' : 'Add Home Assistant'}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                    <ChannelSelect value={applicationId} onChange={setApplicationId} channels={channels} />
                    <TextField
                        label="Home Assistant URL"
                        value={baseUrl}
                        onChange={(e) => setBaseUrl(e.target.value)}
                        placeholder="http://homeassistant.local:8123"
                        required
                    />
                    <TextField
                        label={item?.tokenConfigured ? 'New access token' : 'Long-lived access token'}
                        type="password"
                        value={token}
                        onChange={(e) => setToken(e.target.value)}
                        helperText={item?.tokenConfigured ? 'Leave blank to keep the current token.' : ''}
                    />
                    <TextField
                        label="Event type"
                        value={eventType}
                        onChange={(e) => setEventType(e.target.value)}
                        placeholder="state_changed"
                        helperText="Leave blank to receive all Home Assistant events."
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
                        !applicationId ||
                        !baseUrl ||
                        (!item?.tokenConfigured && !token)
                    }
                    onClick={() => void save()}>
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default Integrations;
