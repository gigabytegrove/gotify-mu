import React from 'react';
import axios from 'axios';
import {
    Button,
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
import AlternateEmail from '@mui/icons-material/AlternateEmail';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import * as config from '../config';
import {useStores} from '../stores';
import {
    ICalendarIntegration,
    IEmailGateway,
    IRSSIntegration,
    ISMTPReceiver,
    ISMTPRoute,
    ISyslogReceiver,
} from '../types';

const api = (path: string) => config.get('url') + path;

interface Props {
    channels: Array<{id: number; name: string}>;
}

interface DeleteTarget {
    title: string;
    text: string;
    run: () => Promise<void>;
}

const channelName = (channels: Props['channels'], id: number) =>
    channels.find((channel) => channel.id === id)?.name || 'Unknown Channel';

const ChannelSelect = ({
    label = 'Channel',
    value,
    channels,
    onChange,
}: {
    label?: string;
    value: number;
    channels: Props['channels'];
    onChange: (value: number) => void;
}) => (
    <TextField
        select
        required
        fullWidth
        label={label}
        value={value || ''}
        onChange={(event) => onChange(Number(event.target.value))}>
        {channels.map((channel) => (
            <MenuItem key={channel.id} value={channel.id}>
                {channel.name}
            </MenuItem>
        ))}
    </TextField>
);

const Row = ({
    title,
    subtitle,
    enabled,
    onEdit,
    onDelete,
}: {
    title: string;
    subtitle: string;
    enabled: boolean;
    onEdit: VoidFunction;
    onDelete: VoidFunction;
}) => (
    <Stack
        direction={{xs: 'column', sm: 'row'}}
        spacing={1}
        sx={{
            p: 1.5,
            border: 1,
            borderColor: 'divider',
            borderRadius: 2,
            alignItems: {sm: 'center'},
            justifyContent: 'space-between',
        }}>
        <div>
            <Typography sx={{fontWeight: 700}}>{title}</Typography>
            <Typography variant="body2" color="text.secondary">
                {subtitle} · {enabled ? 'Enabled' : 'Disabled'}
            </Typography>
        </div>
        <Stack direction="row" spacing={0.5}>
            <Button size="small" onClick={onEdit}>
                Edit
            </Button>
            <Button size="small" color="error" startIcon={<Delete />} onClick={onDelete}>
                Delete
            </Button>
        </Stack>
    </Stack>
);

const FirstPartyIntegrations = ({channels}: Props) => {
    const {snackManager} = useStores();
    const [rss, setRss] = React.useState<IRSSIntegration[]>([]);
    const [calendars, setCalendars] = React.useState<ICalendarIntegration[]>([]);
    const [emailGateways, setEmailGateways] = React.useState<IEmailGateway[]>([]);
    const [smtpReceiver, setSmtpReceiver] = React.useState<ISMTPReceiver>();
    const [smtpRoutes, setSmtpRoutes] = React.useState<ISMTPRoute[]>([]);
    const [syslog, setSyslog] = React.useState<ISyslogReceiver[]>([]);
    const [rssEdit, setRssEdit] = React.useState<IRSSIntegration | null | undefined>();
    const [calendarEdit, setCalendarEdit] =
        React.useState<ICalendarIntegration | null | undefined>();
    const [emailEdit, setEmailEdit] = React.useState<IEmailGateway | null | undefined>();
    const [smtpRouteEdit, setSmtpRouteEdit] =
        React.useState<ISMTPRoute | null | undefined>();
    const [syslogEdit, setSyslogEdit] =
        React.useState<ISyslogReceiver | null | undefined>();
    const [smtpOpen, setSmtpOpen] = React.useState(false);
    const [deleteTarget, setDeleteTarget] = React.useState<DeleteTarget>();

    const refresh = React.useCallback(async () => {
        const [rssResult, calendarResult, emailResult, receiverResult, routesResult, syslogResult] =
            await Promise.all([
                axios.get<IRSSIntegration[]>(api('integration/rss')),
                axios.get<ICalendarIntegration[]>(api('integration/calendar')),
                axios.get<IEmailGateway[]>(api('integration/email-gateway')),
                axios.get<ISMTPReceiver>(api('integration/smtp-receiver')),
                axios.get<ISMTPRoute[]>(api('integration/smtp-route')),
                axios.get<ISyslogReceiver[]>(api('integration/syslog')),
            ]);
        setRss(rssResult.data);
        setCalendars(calendarResult.data);
        setEmailGateways(emailResult.data);
        setSmtpReceiver(receiverResult.data);
        setSmtpRoutes(routesResult.data);
        setSyslog(syslogResult.data);
    }, []);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    const requestDelete = (title: string, text: string, run: () => Promise<void>) =>
        setDeleteTarget({
            title,
            text,
            run: async () => {
                await run();
                await refresh();
                snackManager.snack('Deleted');
            },
        });

    return (
        <>
            <SurfaceCard
                title="RSS / Atom"
                subtitle="Watch feeds and publish new entries to a Channel."
                action={
                    <Button variant="contained" startIcon={<Add />} onClick={() => setRssEdit(null)}>
                        Add Feed
                    </Button>
                }>
                <Stack spacing={1}>
                    {rss.length === 0 && (
                        <Typography color="text.secondary">No feed monitors configured.</Typography>
                    )}
                    {rss.map((item) => (
                        <Row
                            key={item.id}
                            title={item.name}
                            subtitle={item.url + ' · ' + channelName(channels, item.applicationId)}
                            enabled={item.enabled}
                            onEdit={() => setRssEdit(item)}
                            onDelete={() =>
                                requestDelete(
                                    'Delete RSS / Atom monitor?',
                                    'This stops polling the feed. Existing messages are not removed.',
                                    () => axios.delete(api('integration/rss/' + item.id)).then(() => undefined)
                                )
                            }
                        />
                    ))}
                </Stack>
            </SurfaceCard>

            <SurfaceCard
                title="Calendar / iCal"
                subtitle="Monitor iCalendar feeds and publish upcoming-event reminders."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setCalendarEdit(null)}>
                        Add Calendar
                    </Button>
                }>
                <Stack spacing={1}>
                    {calendars.length === 0 && (
                        <Typography color="text.secondary">No calendars configured.</Typography>
                    )}
                    {calendars.map((item) => (
                        <Row
                            key={item.id}
                            title={item.name}
                            subtitle={
                                item.url +
                                ' · ' +
                                item.advanceMinutes +
                                ' minute reminder · ' +
                                channelName(channels, item.applicationId)
                            }
                            enabled={item.enabled}
                            onEdit={() => setCalendarEdit(item)}
                            onDelete={() =>
                                requestDelete(
                                    'Delete calendar monitor?',
                                    'This stops calendar polling. Existing messages are not removed.',
                                    () =>
                                        axios
                                            .delete(api('integration/calendar/' + item.id))
                                            .then(() => undefined)
                                )
                            }
                        />
                    ))}
                </Stack>
            </SurfaceCard>

            <SurfaceCard
                title="Email Gateway"
                subtitle="Forward qualifying Channel messages through an SMTP server."
                action={
                    <Button variant="contained" startIcon={<Add />} onClick={() => setEmailEdit(null)}>
                        Add Email Gateway
                    </Button>
                }>
                <Stack spacing={1}>
                    {emailGateways.length === 0 && (
                        <Typography color="text.secondary">No email gateways configured.</Typography>
                    )}
                    {emailGateways.map((item) => (
                        <Row
                            key={item.id}
                            title={item.name}
                            subtitle={
                                item.host +
                                ':' +
                                item.port +
                                ' · ' +
                                item.toAddresses +
                                ' · ' +
                                channelName(channels, item.applicationId)
                            }
                            enabled={item.enabled}
                            onEdit={() => setEmailEdit(item)}
                            onDelete={() =>
                                requestDelete(
                                    'Delete email gateway?',
                                    'Messages will no longer be forwarded through this gateway.',
                                    () =>
                                        axios
                                            .delete(api('integration/email-gateway/' + item.id))
                                            .then(() => undefined)
                                )
                            }
                        />
                    ))}
                </Stack>
            </SurfaceCard>

            <SurfaceCard
                title="SMTP Receiver"
                subtitle="Accept inbound email and route recipient addresses to Channels."
                action={
                    <Button variant="contained" startIcon={<AlternateEmail />} onClick={() => setSmtpOpen(true)}>
                        Configure Receiver
                    </Button>
                }>
                <Stack spacing={1.5}>
                    <Typography variant="body2" color="text.secondary">
                        {smtpReceiver
                            ? smtpReceiver.listenAddress +
                              ' · ' +
                              (smtpReceiver.enabled ? 'Enabled' : 'Disabled')
                            : 'Receiver configuration unavailable'}
                    </Typography>
                    <Stack direction="row" sx={{justifyContent: 'space-between', alignItems: 'center'}}>
                        <Typography sx={{fontWeight: 700}}>Recipient Routes</Typography>
                        <Button size="small" startIcon={<Add />} onClick={() => setSmtpRouteEdit(null)}>
                            Add Route
                        </Button>
                    </Stack>
                    {smtpRoutes.length === 0 && (
                        <Typography color="text.secondary">No SMTP recipient routes configured.</Typography>
                    )}
                    {smtpRoutes.map((item) => (
                        <Row
                            key={item.id}
                            title={item.recipient}
                            subtitle={channelName(channels, item.applicationId)}
                            enabled={item.enabled}
                            onEdit={() => setSmtpRouteEdit(item)}
                            onDelete={() =>
                                requestDelete(
                                    'Delete SMTP route?',
                                    'Mail sent to this recipient will no longer be routed to its Channel.',
                                    () =>
                                        axios
                                            .delete(api('integration/smtp-route/' + item.id))
                                            .then(() => undefined)
                                )
                            }
                        />
                    ))}
                </Stack>
            </SurfaceCard>

            <SurfaceCard
                title="Syslog Receiver"
                subtitle="Listen for syslog messages and publish qualifying events to a Channel."
                action={
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setSyslogEdit(null)}>
                        Add Syslog Receiver
                    </Button>
                }>
                <Stack spacing={1}>
                    {syslog.length === 0 && (
                        <Typography color="text.secondary">No syslog receivers configured.</Typography>
                    )}
                    {syslog.map((item) => (
                        <Row
                            key={item.id}
                            title={item.name}
                            subtitle={
                                item.protocol.toUpperCase() +
                                ' ' +
                                item.listenAddress +
                                ' · ' +
                                channelName(channels, item.applicationId)
                            }
                            enabled={item.enabled}
                            onEdit={() => setSyslogEdit(item)}
                            onDelete={() =>
                                requestDelete(
                                    'Delete syslog receiver?',
                                    'This listener will stop receiving syslog messages.',
                                    () =>
                                        axios
                                            .delete(api('integration/syslog/' + item.id))
                                            .then(() => undefined)
                                )
                            }
                        />
                    ))}
                </Stack>
            </SurfaceCard>

            {rssEdit !== undefined && (
                <RSSDialog
                    item={rssEdit}
                    channels={channels}
                    onClose={() => setRssEdit(undefined)}
                    onSaved={async () => {
                        setRssEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {calendarEdit !== undefined && (
                <CalendarDialog
                    item={calendarEdit}
                    channels={channels}
                    onClose={() => setCalendarEdit(undefined)}
                    onSaved={async () => {
                        setCalendarEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {emailEdit !== undefined && (
                <EmailGatewayDialog
                    item={emailEdit}
                    channels={channels}
                    onClose={() => setEmailEdit(undefined)}
                    onSaved={async () => {
                        setEmailEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {smtpOpen && smtpReceiver && (
                <SMTPReceiverDialog
                    item={smtpReceiver}
                    onClose={() => setSmtpOpen(false)}
                    onSaved={async () => {
                        setSmtpOpen(false);
                        await refresh();
                    }}
                />
            )}
            {smtpRouteEdit !== undefined && (
                <SMTPRouteDialog
                    item={smtpRouteEdit}
                    channels={channels}
                    onClose={() => setSmtpRouteEdit(undefined)}
                    onSaved={async () => {
                        setSmtpRouteEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {syslogEdit !== undefined && (
                <SyslogDialog
                    item={syslogEdit}
                    channels={channels}
                    onClose={() => setSyslogEdit(undefined)}
                    onSaved={async () => {
                        setSyslogEdit(undefined);
                        await refresh();
                    }}
                />
            )}
            {deleteTarget && (
                <ConfirmDialog
                    title={deleteTarget.title}
                    text={deleteTarget.text}
                    requireElevated
                    fClose={() => setDeleteTarget(undefined)}
                    fOnSubmit={() => void deleteTarget.run()}
                />
            )}
        </>
    );
};

const RSSDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: IRSSIntegration | null;
    channels: Props['channels'];
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [url, setUrl] = React.useState(item?.url || '');
    const [pollMinutes, setPollMinutes] = React.useState(item?.pollMinutes || 5);
    const [titlePrefix, setTitlePrefix] = React.useState(item?.titlePrefix || '');
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const save = async () => {
        const payload = {name, applicationId, url, pollMinutes, titlePrefix, enabled};
        if (item) await axios.put(api('integration/rss/' + item.id), payload);
        else await axios.post(api('integration/rss'), payload);
        await onSaved();
    };
    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit RSS / Atom Monitor' : 'Add RSS / Atom Monitor'}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                    <ChannelSelect value={applicationId} channels={channels} onChange={setApplicationId} />
                    <TextField label="Feed URL" value={url} onChange={(e) => setUrl(e.target.value)} required />
                    <TextField label="Poll every (minutes)" type="number" value={pollMinutes} onChange={(e) => setPollMinutes(Number(e.target.value))} />
                    <TextField label="Title prefix" value={titlePrefix} onChange={(e) => setTitlePrefix(e.target.value)} />
                    <FormControlLabel control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />} label="Enabled" />
                </Stack>
            </DialogContent>
            <DialogActions><Button onClick={onClose}>Cancel</Button><Button variant="contained" disabled={!name || !applicationId || !url} onClick={() => void save()}>Save</Button></DialogActions>
        </Dialog>
    );
};

const CalendarDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: ICalendarIntegration | null;
    channels: Props['channels'];
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [url, setUrl] = React.useState(item?.url || '');
    const [pollMinutes, setPollMinutes] = React.useState(item?.pollMinutes || 5);
    const [advanceMinutes, setAdvanceMinutes] = React.useState(item?.advanceMinutes || 15);
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const save = async () => {
        const payload = {name, applicationId, url, pollMinutes, advanceMinutes, enabled};
        if (item) await axios.put(api('integration/calendar/' + item.id), payload);
        else await axios.post(api('integration/calendar'), payload);
        await onSaved();
    };
    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit Calendar Monitor' : 'Add Calendar Monitor'}</DialogTitle>
            <DialogContent><Stack spacing={2} sx={{pt: 1}}>
                <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                <ChannelSelect value={applicationId} channels={channels} onChange={setApplicationId} />
                <TextField label="iCal URL" value={url} onChange={(e) => setUrl(e.target.value)} required />
                <TextField label="Poll every (minutes)" type="number" value={pollMinutes} onChange={(e) => setPollMinutes(Number(e.target.value))} />
                <TextField label="Remind before event (minutes)" type="number" value={advanceMinutes} onChange={(e) => setAdvanceMinutes(Number(e.target.value))} />
                <FormControlLabel control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />} label="Enabled" />
            </Stack></DialogContent>
            <DialogActions><Button onClick={onClose}>Cancel</Button><Button variant="contained" disabled={!name || !applicationId || !url} onClick={() => void save()}>Save</Button></DialogActions>
        </Dialog>
    );
};

const EmailGatewayDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: IEmailGateway | null;
    channels: Props['channels'];
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [host, setHost] = React.useState(item?.host || '');
    const [port, setPort] = React.useState(item?.port || 587);
    const [useTls, setUseTls] = React.useState(item?.useTls || false);
    const [startTls, setStartTls] = React.useState(item?.startTls ?? true);
    const [username, setUsername] = React.useState(item?.username || '');
    const [password, setPassword] = React.useState('');
    const [fromAddress, setFromAddress] = React.useState(item?.fromAddress || '');
    const [toAddresses, setToAddresses] = React.useState(item?.toAddresses || '');
    const [minPriority, setMinPriority] = React.useState(item?.minPriority || 0);
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const save = async () => {
        const payload = {name, applicationId, host, port, useTls, startTls, username, password, fromAddress, toAddresses, minPriority, enabled};
        if (item) await axios.put(api('integration/email-gateway/' + item.id), payload);
        else await axios.post(api('integration/email-gateway'), payload);
        await onSaved();
    };
    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit Email Gateway' : 'Add Email Gateway'}</DialogTitle>
            <DialogContent><Stack spacing={2} sx={{pt: 1}}>
                <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                <ChannelSelect value={applicationId} channels={channels} onChange={setApplicationId} />
                <TextField label="SMTP server" value={host} onChange={(e) => setHost(e.target.value)} required />
                <TextField label="Port" type="number" value={port} onChange={(e) => setPort(Number(e.target.value))} />
                <TextField label="Username" value={username} onChange={(e) => setUsername(e.target.value)} />
                <TextField label={item?.passwordConfigured ? 'New password' : 'Password'} type="password" value={password} onChange={(e) => setPassword(e.target.value)} helperText={item?.passwordConfigured ? 'Leave blank to keep the saved password.' : ''} />
                <TextField label="From address" value={fromAddress} onChange={(e) => setFromAddress(e.target.value)} required />
                <TextField label="Recipients" value={toAddresses} onChange={(e) => setToAddresses(e.target.value)} helperText="Separate multiple addresses with commas." required />
                <TextField label="Minimum priority" type="number" value={minPriority} onChange={(e) => setMinPriority(Number(e.target.value))} />
                <FormControlLabel control={<Switch checked={useTls} onChange={(e) => {setUseTls(e.target.checked); if(e.target.checked) setStartTls(false);}} />} label="Implicit TLS" />
                <FormControlLabel control={<Switch checked={startTls} onChange={(e) => {setStartTls(e.target.checked); if(e.target.checked) setUseTls(false);}} />} label="STARTTLS" />
                <FormControlLabel control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />} label="Enabled" />
            </Stack></DialogContent>
            <DialogActions><Button onClick={onClose}>Cancel</Button><Button variant="contained" disabled={!name || !applicationId || !host || !fromAddress || !toAddresses} onClick={() => void save()}>Save</Button></DialogActions>
        </Dialog>
    );
};

const SMTPReceiverDialog = ({
    item,
    onClose,
    onSaved,
}: {
    item: ISMTPReceiver;
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [listenAddress, setListenAddress] = React.useState(item.listenAddress || ':2525');
    const [username, setUsername] = React.useState(item.username || '');
    const [password, setPassword] = React.useState('');
    const [allowedCidrs, setAllowedCidrs] = React.useState(item.allowedCidrs || '');
    const [maxMessageBytes, setMaxMessageBytes] = React.useState(item.maxMessageBytes || 10 * 1024 * 1024);
    const [enabled, setEnabled] = React.useState(item.enabled);
    const save = async () => {
        await axios.put(api('integration/smtp-receiver'), {listenAddress, username, password, allowedCidrs, maxMessageBytes, enabled});
        await onSaved();
    };
    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>Configure SMTP Receiver</DialogTitle>
            <DialogContent><Stack spacing={2} sx={{pt: 1}}>
                <TextField label="Listen address" value={listenAddress} onChange={(e) => setListenAddress(e.target.value)} helperText="Example: :2525" required />
                <TextField label="Username" value={username} onChange={(e) => setUsername(e.target.value)} />
                <TextField label={item.passwordConfigured ? 'New password' : 'Password'} type="password" value={password} onChange={(e) => setPassword(e.target.value)} helperText={item.passwordConfigured ? 'Leave blank to keep the saved password.' : ''} />
                <TextField label="Allowed networks" value={allowedCidrs} onChange={(e) => setAllowedCidrs(e.target.value)} helperText="Comma-separated CIDR ranges. Leave blank to allow any source." />
                <TextField label="Maximum message size (bytes)" type="number" value={maxMessageBytes} onChange={(e) => setMaxMessageBytes(Number(e.target.value))} />
                <FormControlLabel control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />} label="Enabled" />
            </Stack></DialogContent>
            <DialogActions><Button onClick={onClose}>Cancel</Button><Button variant="contained" disabled={!listenAddress} onClick={() => void save()}>Save</Button></DialogActions>
        </Dialog>
    );
};

const SMTPRouteDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: ISMTPRoute | null;
    channels: Props['channels'];
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [recipient, setRecipient] = React.useState(item?.recipient || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const save = async () => {
        const payload = {recipient, applicationId, enabled};
        if (item) await axios.put(api('integration/smtp-route/' + item.id), payload);
        else await axios.post(api('integration/smtp-route'), payload);
        await onSaved();
    };
    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit SMTP Route' : 'Add SMTP Route'}</DialogTitle>
            <DialogContent><Stack spacing={2} sx={{pt: 1}}>
                <TextField label="Recipient address" value={recipient} onChange={(e) => setRecipient(e.target.value)} required />
                <ChannelSelect value={applicationId} channels={channels} onChange={setApplicationId} />
                <FormControlLabel control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />} label="Enabled" />
            </Stack></DialogContent>
            <DialogActions><Button onClick={onClose}>Cancel</Button><Button variant="contained" disabled={!recipient || !applicationId} onClick={() => void save()}>Save</Button></DialogActions>
        </Dialog>
    );
};

const SyslogDialog = ({
    item,
    channels,
    onClose,
    onSaved,
}: {
    item: ISyslogReceiver | null;
    channels: Props['channels'];
    onClose: VoidFunction;
    onSaved: () => Promise<void>;
}) => {
    const [name, setName] = React.useState(item?.name || '');
    const [applicationId, setApplicationId] = React.useState(item?.applicationId || 0);
    const [listenAddress, setListenAddress] = React.useState(item?.listenAddress || ':5514');
    const [protocol, setProtocol] = React.useState<'udp' | 'tcp'>(item?.protocol || 'udp');
    const [allowedCidrs, setAllowedCidrs] = React.useState(item?.allowedCidrs || '');
    const [minSeverity, setMinSeverity] = React.useState(item?.minSeverity ?? 7);
    const [enabled, setEnabled] = React.useState(item?.enabled ?? true);
    const save = async () => {
        const payload = {name, applicationId, listenAddress, protocol, allowedCidrs, minSeverity, enabled};
        if (item) await axios.put(api('integration/syslog/' + item.id), payload);
        else await axios.post(api('integration/syslog'), payload);
        await onSaved();
    };
    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item ? 'Edit Syslog Receiver' : 'Add Syslog Receiver'}</DialogTitle>
            <DialogContent><Stack spacing={2} sx={{pt: 1}}>
                <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} required />
                <ChannelSelect value={applicationId} channels={channels} onChange={setApplicationId} />
                <TextField label="Listen address" value={listenAddress} onChange={(e) => setListenAddress(e.target.value)} required />
                <TextField select label="Protocol" value={protocol} onChange={(e) => setProtocol(e.target.value as 'udp' | 'tcp')}>
                    <MenuItem value="udp">UDP</MenuItem><MenuItem value="tcp">TCP</MenuItem>
                </TextField>
                <TextField label="Allowed networks" value={allowedCidrs} onChange={(e) => setAllowedCidrs(e.target.value)} helperText="Comma-separated CIDR ranges. Leave blank to allow any source." />
                <TextField label="Minimum severity" type="number" value={minSeverity} onChange={(e) => setMinSeverity(Number(e.target.value))} helperText="0 = Emergency, 7 = Debug" />
                <FormControlLabel control={<Switch checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />} label="Enabled" />
            </Stack></DialogContent>
            <DialogActions><Button onClick={onClose}>Cancel</Button><Button variant="contained" disabled={!name || !applicationId || !listenAddress} onClick={() => void save()}>Save</Button></DialogActions>
        </Dialog>
    );
};

export default FirstPartyIntegrations;
