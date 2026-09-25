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
import FirstPartyConnectors from './FirstPartyConnectors';
import {
    IHomeAssistantIntegration,
    IMQTTIntegration,
    IWebhookDelivery,
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
    const [webhookHistory, setWebhookHistory] = React.useState<{name: string; items: IWebhookDelivery[]} | undefined>();
    const [confirm, setConfirm] = React.useState<
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
                                        onClick={async () => {
                                            await axios.post(api(`integration/webhook/${item.id}/test`));
                                            snackManager.snack('Webhook test notification sent');
                                        }}>
                                        Send Test
                                    </Button>
                                    <Button
                                        size="small"
                                        onClick={async () => {
                                            const response = await axios.get<IWebhookDelivery[]>(
                                                api('integration/webhook/' + item.id + '/history')
                                            );
                                            setWebhookHistory({name: item.name, items: response.data});
                                        }}>
                                        History
                                    </Button>
                                    <Button
                                        size="small"
                                        startIcon={<Refresh />}
                                        onClick={() =>
                                            setConfirm({
                                                title: 'Regenerate Webhook URL?',
                                                text: 'The current webhook URL will stop working immediately. Systems using it must be updated.',
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
                                </Stack>
                                <Stack direction="row" spacing={0.75} useFlexGap sx={{flexWrap: 'wrap'}}>
                                    {item.requireSignature && <Chip size="small" label="Signed requests required" />}
                                    <Chip size="small" variant="outlined" label={`${item.rateLimitPerMinute}/min`} />
                                    {item.allowedCidrs && (
                                        <Chip size="small" variant="outlined" label="Source IP restrictions" />
                                    )}
                                </Stack>
                            </Stack>
                        ),
                        onEdit: () => setWebhookEdit(item),
                        onDelete: async () => {
                            setConfirm({
                                title: 'Delete Webhook?',
                                text: 'Requests to this webhook will stop working immediately.',
                                run: async () => {
                                    await axios.delete(api(`integration/webhook/${item.id}`));
                                    await refresh();
                                    snackManager.snack('Webhook deleted');
                                },
                            });
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
                        details: (
                            <IntegrationHealth
                                status={item.status}
                                lastConnectedAt={item.lastConnectedAt}
                                lastActivityAt={item.lastMessageAt}
                                lastActivityLabel="Last message"
                                lastError={item.lastError}
                                reconnectCount={item.reconnectCount}>
                                <Button
                                    size="small"
                                    onClick={async () => {
                                        await axios.post(api(`integration/mqtt/${item.id}/test`));
                                        snackManager.snack('MQTT connection test succeeded');
                                        await refresh();
                                    }}>
                                    Test Connection
                                </Button>
                            </IntegrationHealth>
                        ),
                        onEdit: () => setMqttEdit(item),
                        onDelete: async () => {
                            setConfirm({
                                title: 'Delete MQTT Connection?',
                                text: 'Gotify MU will stop listening to this MQTT topic.',
                                run: async () => {
                                    await axios.delete(api(`integration/mqtt/${item.id}`));
                                    await refresh();
                                    snackManager.snack('MQTT connection deleted');
                                },
                            });
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
                        details: (
                            <IntegrationHealth
                                status={item.status}
                                lastConnectedAt={item.lastConnectedAt}
                                lastActivityAt={item.lastEventAt}
                                lastActivityLabel="Last event"
                                lastError={item.lastError}
                                reconnectCount={item.reconnectCount}>
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
                            </IntegrationHealth>
                        ),
                        onEdit: () => setHomeAssistantEdit(item),
                        onDelete: async () => {
                            setConfirm({
                                title: 'Delete Home Assistant Connection?',
                                text: 'Gotify MU will stop receiving events from this Home Assistant connection.',
                                run: async () => {
                                    await axios.delete(api(`integration/home-assistant/${item.id}`));
                                    await refresh();
                                    snackManager.snack('Home Assistant connection deleted');
                                },
                            });
                        },
                    }))}
                />
            </SurfaceCard>

            {webhookHistory && (
                <Dialog open onClose={() => setWebhookHistory(undefined)} fullWidth maxWidth="md">
                    <DialogTitle>{webhookHistory.name} · Delivery History</DialogTitle>
                    <DialogContent>
                        <Stack spacing={1} sx={{pt: 1}}>
                            {webhookHistory.items.length === 0 ? (
                                <Typography color="text.secondary">No webhook requests have been recorded.</Typography>
                            ) : webhookHistory.items.map((entry) => (
                                <Box key={entry.id} sx={{p: 1.25, border: 1, borderColor: 'divider', borderRadius: 2}}>
                                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={1} sx={{justifyContent: 'space-between'}}>
                                        <Stack direction="row" spacing={1} sx={{alignItems: 'center', flexWrap: 'wrap'}}>
                                            <Chip
                                                size="small"
                                                label={entry.status}
                                                color={entry.status === 'delivered' ? 'success' : entry.status === 'ignored' ? 'default' : 'warning'}
                                                variant="outlined"
                                            />
                                            <Typography variant="body2">{new Date(entry.createdAt).toLocaleString()}</Typography>
                                            {entry.ipAddress && <Typography variant="caption" color="text.secondary">{entry.ipAddress}</Typography>}
                                        </Stack>
                                        {entry.messageId ? <Typography variant="caption">Message #{entry.messageId}</Typography> : null}
                                    </Stack>
                                    {entry.detail && <Typography variant="caption" color="text.secondary">{entry.detail}</Typography>}
                                </Box>
                            ))}
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={() => setWebhookHistory(undefined)}>Close</Button>
                    </DialogActions>
                </Dialog>
            )}

            {confirm && (
                <ConfirmDialog
                    title={confirm.title}
                    text={confirm.text}
                    requireElevated
                    fClose={() => setConfirm(undefined)}
                    fOnSubmit={() => {
                        const action = confirm.run;
                        setConfirm(undefined);
                        void action();
                    }}
                />
            )}

            <FirstPartyConnectors channels={channels} />

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
        </DefaultPage>
    );
};

const IntegrationHealth = ({
    status,
    lastConnectedAt,
    lastActivityAt,
    lastActivityLabel,
    lastError,
    reconnectCount,
    children,
}: {
    status?: string;
    lastConnectedAt?: string;
    lastActivityAt?: string;
    lastActivityLabel: string;
    lastError?: string;
    reconnectCount: number;
    children?: React.ReactNode;
}) => (
    <Stack spacing={0.75}>
        <Stack direction="row" spacing={0.75} useFlexGap sx={{flexWrap: 'wrap', alignItems: 'center'}}>
            <Chip
                size="small"
                color={status === 'connected' ? 'success' : status === 'reconnecting' ? 'warning' : 'default'}
                variant="outlined"
                label={status || 'Waiting'}
            />
            {lastConnectedAt && (
                <Typography variant="caption" color="text.secondary">
                    Connected {new Date(lastConnectedAt).toLocaleString()}
                </Typography>
            )}
            {lastActivityAt && (
                <Typography variant="caption" color="text.secondary">
                    {lastActivityLabel} {new Date(lastActivityAt).toLocaleString()}
                </Typography>
            )}
            {reconnectCount > 0 && (
                <Typography variant="caption" color="text.secondary">
                    Reconnects {reconnectCount}
                </Typography>
            )}
        </Stack>
        {lastError && (
            <Alert severity="warning" sx={{py: 0}}>
                {lastError}
            </Alert>
        )}
        {children && <Stack direction="row" spacing={1}>{children}</Stack>}
    </Stack>
);

interface ListItem {
    id: number;
    icon: React.ReactNode;
    title: string;
    subtitle: string;
    enabled: boolean;
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
                                </Stack>
                                <Typography variant="body2" color="text.secondary">
                                    {item.subtitle}
                                </Typography>
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
    const [matchField, setMatchField] = React.useState(item?.matchField || '');
    const [matchValue, setMatchValue] = React.useState(item?.matchValue || '');
    const [titleTemplate, setTitleTemplate] = React.useState(item?.titleTemplate || '');
    const [messageTemplate, setMessageTemplate] = React.useState(item?.messageTemplate || '');
    const [defaultTitle, setDefaultTitle] = React.useState(item?.defaultTitle || '');
    const [defaultPriority, setDefaultPriority] = React.useState(item?.defaultPriority || 0);
    const [requireSignature, setRequireSignature] = React.useState(item?.requireSignature ?? true);
    const [allowedCidrs, setAllowedCidrs] = React.useState(item?.allowedCidrs || '');
    const [rateLimitPerMinute, setRateLimitPerMinute] = React.useState(item?.rateLimitPerMinute || 120);
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
                matchField,
                matchValue,
                titleTemplate,
                messageTemplate,
                defaultTitle,
                defaultPriority,
                requireSignature,
                allowedCidrs,
                rateLimitPerMinute,
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
                        label="Conditional field"
                        value={matchField}
                        onChange={(e) => setMatchField(e.target.value)}
                        helperText="Optional JSON field path. The request is ignored unless this field exists."
                    />
                    <TextField
                        label="Required value"
                        value={matchValue}
                        onChange={(e) => setMatchValue(e.target.value)}
                        helperText="Optional exact value for the conditional field."
                    />
                    <TextField
                        label="Title template"
                        value={titleTemplate}
                        onChange={(e) => setTitleTemplate(e.target.value)}
                        helperText="Optional. Use placeholders such as {{alert.title}} or {{items.0.name}}."
                    />
                    <TextField
                        label="Message template"
                        value={messageTemplate}
                        onChange={(e) => setMessageTemplate(e.target.value)}
                        multiline
                        minRows={2}
                        helperText="Optional. Use JSON field placeholders or {{raw}} for the original body."
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
                    <FormControlLabel
                        control={<Switch checked={requireSignature} onChange={(e) => setRequireSignature(e.target.checked)} />}
                        label="Require signed requests"
                    />
                    <TextField
                        label="Allowed source IPs / networks"
                        value={allowedCidrs}
                        onChange={(e) => setAllowedCidrs(e.target.value)}
                        placeholder="192.168.1.0/24, 10.0.0.10"
                        helperText="Optional. Separate IP addresses or CIDR networks with commas."
                    />
                    <TextField
                        label="Rate limit per minute"
                        type="number"
                        value={rateLimitPerMinute}
                        onChange={(e) => setRateLimitPerMinute(Number(e.target.value))}
                        slotProps={{htmlInput: {min: 1, max: 10000}}}
                    />
                    <FormControlLabel
                        control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />}
                        label="Enabled"
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
                <Button variant="contained" disabled={saving || !name || !applicationId} onClick={() => void save()}>
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
    const [protocolVersion, setProtocolVersion] = React.useState(item?.protocolVersion || 5);
    const [qos, setQos] = React.useState(item?.qos ?? 0);
    const [caCertificate, setCaCertificate] = React.useState(item?.caCertificate || '');
    const [clientCertificate, setClientCertificate] = React.useState(item?.clientCertificate || '');
    const [clientKey, setClientKey] = React.useState('');
    const [topic, setTopic] = React.useState(item?.topic || '');
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const [saving, setSaving] = React.useState(false);

    const save = async () => {
        setSaving(true);
        try {
            const payload = {
                name,
                applicationId,
                brokerUrl,
                clientId,
                username,
                password,
                protocolVersion,
                qos,
                caCertificate,
                clientCertificate,
                clientKey,
                topic,
                enabled,
            };
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
                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={2}>
                        <TextField
                            select
                            label="Protocol"
                            value={protocolVersion}
                            onChange={(e) => setProtocolVersion(Number(e.target.value))}
                            fullWidth>
                            <MenuItem value={5}>MQTT 5</MenuItem>
                            <MenuItem value={4}>MQTT 3.1.1</MenuItem>
                        </TextField>
                        <TextField
                            select
                            label="QoS"
                            value={qos}
                            onChange={(e) => setQos(Number(e.target.value))}
                            fullWidth>
                            <MenuItem value={0}>0 · At most once</MenuItem>
                            <MenuItem value={1}>1 · At least once</MenuItem>
                            <MenuItem value={2}>2 · Exactly once</MenuItem>
                        </TextField>
                    </Stack>
                    <TextField label="Client ID" value={clientId} onChange={(e) => setClientId(e.target.value)} />
                    <TextField label="Username" value={username} onChange={(e) => setUsername(e.target.value)} />
                    <TextField
                        label={item?.passwordConfigured ? 'New password' : 'Password'}
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        helperText={item?.passwordConfigured ? 'Leave blank to keep the current password.' : ''}
                    />
                    <TextField
                        label="Custom CA certificate"
                        value={caCertificate}
                        onChange={(e) => setCaCertificate(e.target.value)}
                        multiline
                        minRows={3}
                        placeholder="-----BEGIN CERTIFICATE-----"
                        helperText="Optional PEM certificate authority for private brokers."
                    />
                    <TextField
                        label="Client certificate"
                        value={clientCertificate}
                        onChange={(e) => setClientCertificate(e.target.value)}
                        multiline
                        minRows={3}
                        placeholder="-----BEGIN CERTIFICATE-----"
                        helperText="Optional PEM client certificate for mutual TLS."
                    />
                    <TextField
                        label={item?.clientKeyConfigured ? 'New client private key' : 'Client private key'}
                        type="password"
                        value={clientKey}
                        onChange={(e) => setClientKey(e.target.value)}
                        multiline
                        minRows={3}
                        helperText={
                            item?.clientKeyConfigured
                                ? 'Leave blank to keep the current private key.'
                                : 'Required only when a client certificate is configured.'
                        }
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
    const [entityIds, setEntityIds] = React.useState(item?.entityIds || '');
    const [dataField, setDataField] = React.useState(item?.dataField || '');
    const [dataValue, setDataValue] = React.useState(item?.dataValue || '');
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const [saving, setSaving] = React.useState(false);

    const save = async () => {
        setSaving(true);
        try {
            const payload = {name, applicationId, baseUrl, token, eventType, entityIds, dataField, dataValue, enabled};
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
                    <TextField
                        label="Entity IDs"
                        value={entityIds}
                        onChange={(e) => setEntityIds(e.target.value)}
                        placeholder="binary_sensor.front_door, alarm_control_panel.home"
                        helperText="Optional. Only route events whose data.entity_id matches one of these values."
                    />
                    <TextField
                        label="Event data field"
                        value={dataField}
                        onChange={(e) => setDataField(e.target.value)}
                        placeholder="new_state.state"
                        helperText="Optional dotted event-data path that must exist."
                    />
                    <TextField
                        label="Required field value"
                        value={dataValue}
                        onChange={(e) => setDataValue(e.target.value)}
                        helperText="Optional exact value required for the event-data field."
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
