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
import Email from '@mui/icons-material/Email';
import MarkEmailRead from '@mui/icons-material/MarkEmailRead';
import RssFeed from '@mui/icons-material/RssFeed';
import Dns from '@mui/icons-material/Dns';
import Event from '@mui/icons-material/Event';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import * as config from '../config';
import {useStores} from '../stores';
import {
    ICalendarMonitor,
    IEmailGateway,
    IRSSMonitor,
    ISMTPRoute,
    ISyslogRoute,
} from '../types';

const api = (path: string) => config.get('url') + path;

type Channel = {id: number; name: string};
type Kind = 'email' | 'smtp' | 'rss' | 'syslog' | 'calendar';
type AnyItem = IEmailGateway | ISMTPRoute | IRSSMonitor | ISyslogRoute | ICalendarMonitor;

const FirstPartyConnectors = ({channels}: {channels: Channel[]}) => {
    const {snackManager} = useStores();
    const [email, setEmail] = React.useState<IEmailGateway[]>([]);
    const [smtp, setSMTP] = React.useState<ISMTPRoute[]>([]);
    const [rss, setRSS] = React.useState<IRSSMonitor[]>([]);
    const [syslog, setSyslog] = React.useState<ISyslogRoute[]>([]);
    const [calendar, setCalendar] = React.useState<ICalendarMonitor[]>([]);
    const [editing, setEditing] = React.useState<{kind: Kind; item?: AnyItem}>();
    const [confirm, setConfirm] = React.useState<{
        title: string;
        text: string;
        action: () => Promise<void>;
    }>();

    const refresh = React.useCallback(async () => {
        const [a, b, c, d, e] = await Promise.all([
            axios.get<IEmailGateway[]>(api('connector/email')),
            axios.get<ISMTPRoute[]>(api('connector/smtp')),
            axios.get<IRSSMonitor[]>(api('connector/rss')),
            axios.get<ISyslogRoute[]>(api('connector/syslog')),
            axios.get<ICalendarMonitor[]>(api('connector/calendar')),
        ]);
        setEmail(a.data);
        setSMTP(b.data);
        setRSS(c.data);
        setSyslog(d.data);
        setCalendar(e.data);
    }, []);

    React.useEffect(() => {
        void refresh();
    }, [refresh]);

    const name = (id: number) => channels.find((item) => item.id === id)?.name || 'Unknown Channel';
    const remove = (kind: Kind, id: number, label: string) =>
        setConfirm({
            title: 'Delete ' + label + '?',
            text: 'This removes the connector configuration. Existing messages are not deleted.',
            action: async () => {
                await axios.delete(api('connector/' + kind + '/' + id));
                await refresh();
                snackManager.snack(label + ' deleted');
            },
        });

    return (
        <>
            <SurfaceCard
                title="Email Delivery"
                subtitle="Send qualifying Channel notifications through an SMTP server."
                action={<Button variant="contained" startIcon={<Add />} onClick={() => setEditing({kind: 'email'})}>Add Email Delivery</Button>}>
                <ConnectorRows
                    empty="No email delivery connections configured."
                    items={email.map((item) => ({
                        id: item.id,
                        icon: <Email />,
                        title: item.name,
                        subtitle:
                            name(item.sourceApplicationId) +
                            ' → ' +
                            item.toAddresses +
                            ' · ' +
                            item.smtpHost,
                        status: item.enabled ? item.status || 'Ready' : 'Disabled',
                        error: item.lastError,
                        actions: (
                            <>
                                <Button size="small" onClick={async () => {
                                    await axios.post(api('connector/email/' + item.id + '/test'));
                                    snackManager.snack('Test email sent');
                                    await refresh();
                                }}>Send Test</Button>
                                <Button size="small" onClick={() => setEditing({kind:'email', item})}>Edit</Button>
                                <Button size="small" color="error" startIcon={<Delete />} onClick={() => remove('email', item.id, 'Email Delivery')}>Delete</Button>
                            </>
                        ),
                    }))}
                />
            </SurfaceCard>

            <SurfaceCard
                title="SMTP Receiver"
                subtitle="Accept inbound email and route recipients into Channels. Default listener: TCP 2525."
                action={<Button variant="contained" startIcon={<Add />} onClick={() => setEditing({kind:'smtp'})}>Add SMTP Route</Button>}>
                <ConnectorRows
                    empty="No SMTP receiver routes configured."
                    items={smtp.map((item) => ({
                        id:item.id, icon:<MarkEmailRead />, title:item.name,
                        subtitle:item.recipient+' → '+name(item.applicationId),
                        status:item.enabled?'Enabled':'Disabled',
                        actions:<>
                            <Button size="small" onClick={() => setEditing({kind:'smtp',item})}>Edit</Button>
                            <Button size="small" color="error" startIcon={<Delete />} onClick={() => remove('smtp',item.id,'SMTP Route')}>Delete</Button>
                        </>,
                    }))}
                />
            </SurfaceCard>

            <SurfaceCard
                title="RSS / Atom"
                subtitle="Monitor feeds and publish new entries into Channels."
                action={<Button variant="contained" startIcon={<Add />} onClick={() => setEditing({kind:'rss'})}>Add Feed</Button>}>
                <ConnectorRows
                    empty="No RSS or Atom feeds configured."
                    items={rss.map((item) => ({
                        id:item.id, icon:<RssFeed />, title:item.name,
                        subtitle:item.url+' → '+name(item.applicationId),
                        status:item.enabled?item.status||'Waiting':'Disabled', error:item.lastError,
                        actions:<>
                            <Button size="small" startIcon={<Refresh />} onClick={() => void refresh()}>Refresh Status</Button>
                            <Button size="small" onClick={() => setEditing({kind:'rss',item})}>Edit</Button>
                            <Button size="small" color="error" startIcon={<Delete />} onClick={() => remove('rss',item.id,'Feed Monitor')}>Delete</Button>
                        </>,
                    }))}
                />
            </SurfaceCard>

            <SurfaceCard
                title="Syslog Receiver"
                subtitle="Receive syslog over UDP and route matching severity/facility events into Channels. Default listener: UDP 5514."
                action={<Button variant="contained" startIcon={<Add />} onClick={() => setEditing({kind:'syslog'})}>Add Syslog Route</Button>}>
                <ConnectorRows
                    empty="No syslog routes configured."
                    items={syslog.map((item) => ({
                        id:item.id, icon:<Dns />, title:item.name,
                        subtitle:
                            'Facility ' + (item.facility < 0 ? 'Any' : item.facility) +
                            ' · Severity 0-' + item.maxSeverity + ' → ' + name(item.applicationId),
                        status:item.enabled?'Enabled':'Disabled',
                        actions:<>
                            <Button size="small" onClick={() => setEditing({kind:'syslog',item})}>Edit</Button>
                            <Button size="small" color="error" startIcon={<Delete />} onClick={() => remove('syslog',item.id,'Syslog Route')}>Delete</Button>
                        </>,
                    }))}
                />
            </SurfaceCard>

            <SurfaceCard
                title="Calendar / iCal"
                subtitle="Monitor iCalendar feeds and notify a Channel as events approach."
                action={<Button variant="contained" startIcon={<Add />} onClick={() => setEditing({kind:'calendar'})}>Add Calendar</Button>}>
                <ConnectorRows
                    empty="No calendar monitors configured."
                    items={calendar.map((item) => ({
                        id:item.id, icon:<Event />, title:item.name,
                        subtitle:item.url+' → '+name(item.applicationId),
                        status:item.enabled?item.status||'Waiting':'Disabled', error:item.lastError,
                        actions:<>
                            <Button size="small" startIcon={<Refresh />} onClick={() => void refresh()}>Refresh Status</Button>
                            <Button size="small" onClick={() => setEditing({kind:'calendar',item})}>Edit</Button>
                            <Button size="small" color="error" startIcon={<Delete />} onClick={() => remove('calendar',item.id,'Calendar Monitor')}>Delete</Button>
                        </>,
                    }))}
                />
            </SurfaceCard>

            {editing && (
                <ConnectorDialog
                    kind={editing.kind}
                    item={editing.item}
                    channels={channels}
                    onClose={() => setEditing(undefined)}
                    onSaved={async () => {
                        setEditing(undefined);
                        await refresh();
                    }}
                />
            )}
            {confirm && (
                <ConfirmDialog
                    title={confirm.title}
                    text={confirm.text}
                    requireElevated
                    fClose={() => setConfirm(undefined)}
                    fOnSubmit={() => void confirm.action()}
                />
            )}
        </>
    );
};

interface Row {
    id: number;
    icon: React.ReactNode;
    title: string;
    subtitle: string;
    status: string;
    error?: string;
    actions: React.ReactNode;
}
const ConnectorRows = ({items, empty}: {items: Row[]; empty: string}) => {
    if (!items.length) return <Typography color="text.secondary">{empty}</Typography>;
    return <Stack spacing={1}>{items.map((item) => (
        <Box key={item.id} sx={{border:1,borderColor:'divider',borderRadius:2,p:1.5}}>
            <Stack direction={{xs:'column',md:'row'}} spacing={1.5} sx={{justifyContent:'space-between'}}>
                <Stack direction="row" spacing={1.25} sx={{minWidth:0}}>
                    <Box sx={{pt:0.25,color:'text.secondary'}}>{item.icon}</Box>
                    <Box sx={{minWidth:0}}>
                        <Stack direction="row" spacing={0.75} useFlexGap sx={{alignItems:'center',flexWrap:'wrap'}}>
                            <Typography sx={{fontWeight:700}}>{item.title}</Typography>
                            <Chip size="small" variant="outlined" label={item.status} color={item.error?'error':'default'} />
                        </Stack>
                        <Typography variant="body2" color="text.secondary" sx={{wordBreak:'break-word'}}>{item.subtitle}</Typography>
                        {item.error && <Typography variant="caption" color="error">{item.error}</Typography>}
                    </Box>
                </Stack>
                <Stack direction="row" spacing={0.5} useFlexGap sx={{flexWrap:'wrap'}}>{item.actions}</Stack>
            </Stack>
        </Box>
    ))}</Stack>;
};

const ChannelSelect = ({value,onChange,channels,label='Channel'}:{value:number;onChange:(id:number)=>void;channels:Channel[];label?:string}) => (
    <TextField select label={label} value={value||''} onChange={(e)=>onChange(Number(e.target.value))} required fullWidth>
        {channels.map((channel)=><MenuItem key={channel.id} value={channel.id}>{channel.name}</MenuItem>)}
    </TextField>
);

const ConnectorDialog = ({
    kind,item,channels,onClose,onSaved,
}:{
    kind:Kind;item?:AnyItem;channels:Channel[];onClose:VoidFunction;onSaved:()=>Promise<void>;
}) => {
    const [value,setValue]=React.useState<Record<string, string|number|boolean|undefined>>(()=>{
        if(kind==='email'){
            const x=item as IEmailGateway|undefined;return {
                name:x?.name||'',sourceApplicationId:x?.sourceApplicationId||0,smtpHost:x?.smtpHost||'',
                smtpPort:x?.smtpPort||587,tlsMode:x?.tlsMode||'starttls',username:x?.username||'',password:'',
                fromAddress:x?.fromAddress||'',toAddresses:x?.toAddresses||'',minPriority:x?.minPriority||0,enabled:x?.enabled??true,
            };
        }
        if(kind==='smtp'){
            const x=item as ISMTPRoute|undefined;return {
                name:x?.name||'',applicationId:x?.applicationId||0,recipient:x?.recipient||'',
                allowedCidrs:x?.allowedCidrs||'',username:x?.username||'',password:'',enabled:x?.enabled??true,
            };
        }
        if(kind==='rss'){
            const x=item as IRSSMonitor|undefined;return {name:x?.name||'',applicationId:x?.applicationId||0,url:x?.url||'',intervalMinutes:x?.intervalMinutes||15,enabled:x?.enabled??true};
        }
        if(kind==='syslog'){
            const x=item as ISyslogRoute|undefined;return {name:x?.name||'',applicationId:x?.applicationId||0,facility:x?.facility??-1,maxSeverity:x?.maxSeverity??7,allowedCidrs:x?.allowedCidrs||'',enabled:x?.enabled??true};
        }
        const x=item as ICalendarMonitor|undefined;return {name:x?.name||'',applicationId:x?.applicationId||0,url:x?.url||'',intervalMinutes:x?.intervalMinutes||15,notifyBeforeMinutes:x?.notifyBeforeMinutes||60,enabled:x?.enabled??true};
    });
    const [saving,setSaving]=React.useState(false);
    const set=(key:string,next:string|number|boolean)=>setValue((current)=>({...current,[key]:next}));
    const endpoint='connector/'+kind;
    const save=async()=>{
        setSaving(true);
        try{
            if(item) await axios.put(api(endpoint+'/'+item.id),value);
            else await axios.post(api(endpoint),value);
            await onSaved();
        }finally{setSaving(false);}
    };
    const appIdKey=kind==='email'?'sourceApplicationId':'applicationId';
    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{item?'Edit ':'Add '}{kind==='email'?'Email Delivery':kind==='smtp'?'SMTP Route':kind==='rss'?'Feed Monitor':kind==='syslog'?'Syslog Route':'Calendar Monitor'}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt:1}}>
                    <TextField label="Name" value={value.name} onChange={(e)=>set('name',e.target.value)} required />
                    <ChannelSelect value={Number(value[appIdKey])} onChange={(id)=>set(appIdKey,id)} channels={channels} />
                    {kind==='email' && <>
                        <TextField label="SMTP host" value={value.smtpHost} onChange={(e)=>set('smtpHost',e.target.value)} required />
                        <TextField type="number" label="SMTP port" value={value.smtpPort} onChange={(e)=>set('smtpPort',Number(e.target.value))} />
                        <TextField select label="Connection security" value={value.tlsMode} onChange={(e)=>set('tlsMode',e.target.value)}>
                            <MenuItem value="starttls">STARTTLS</MenuItem><MenuItem value="tls">TLS</MenuItem><MenuItem value="none">None</MenuItem>
                        </TextField>
                        <TextField label="Username" value={value.username} onChange={(e)=>set('username',e.target.value)} />
                        <TextField type="password" label={(item as IEmailGateway|undefined)?.passwordConfigured?'New password':'Password'} value={value.password} onChange={(e)=>set('password',e.target.value)} helperText={(item as IEmailGateway|undefined)?.passwordConfigured?'Leave blank to keep the current password.':''} />
                        <TextField label="From address" value={value.fromAddress} onChange={(e)=>set('fromAddress',e.target.value)} required />
                        <TextField label="Recipients" value={value.toAddresses} onChange={(e)=>set('toAddresses',e.target.value)} helperText="Separate multiple addresses with commas." required />
                        <TextField type="number" label="Minimum priority" value={value.minPriority} onChange={(e)=>set('minPriority',Number(e.target.value))} />
                    </>}
                    {kind==='smtp' && <>
                        <TextField label="Recipient" value={value.recipient} onChange={(e)=>set('recipient',e.target.value)} helperText="Exact address or wildcard such as *@alerts.example.com" required />
                        <TextField label="Allowed source networks" value={value.allowedCidrs} onChange={(e)=>set('allowedCidrs',e.target.value)} helperText="Optional CIDR list. Empty allows any source that can reach the listener." />
                        <TextField label="Username" value={value.username} onChange={(e)=>set('username',e.target.value)} helperText="Optional SMTP AUTH username." />
                        <TextField type="password" label={(item as ISMTPRoute|undefined)?.passwordConfigured?'New password':'Password'} value={value.password} onChange={(e)=>set('password',e.target.value)} helperText={(item as ISMTPRoute|undefined)?.passwordConfigured?'Leave blank to keep the current password.':'Used when SMTP AUTH is enabled for this route.'} />
                    </>}
                    {kind==='rss' && <>
                        <TextField label="Feed URL" value={value.url} onChange={(e)=>set('url',e.target.value)} required />
                        <TextField type="number" label="Check every" value={value.intervalMinutes} onChange={(e)=>set('intervalMinutes',Number(e.target.value))} helperText="Minutes between checks." />
                    </>}
                    {kind==='syslog' && <>
                        <TextField type="number" label="Facility" value={value.facility} onChange={(e)=>set('facility',Number(e.target.value))} helperText="-1 receives any facility; otherwise use 0-23." />
                        <TextField type="number" label="Maximum severity" value={value.maxSeverity} onChange={(e)=>set('maxSeverity',Number(e.target.value))} helperText="0 is Emergency, 7 is Debug. Events at or above this importance are routed." />
                        <TextField label="Allowed source networks" value={value.allowedCidrs} onChange={(e)=>set('allowedCidrs',e.target.value)} helperText="Optional CIDR list." />
                    </>}
                    {kind==='calendar' && <>
                        <TextField label="iCal URL" value={value.url} onChange={(e)=>set('url',e.target.value)} required />
                        <TextField type="number" label="Check every" value={value.intervalMinutes} onChange={(e)=>set('intervalMinutes',Number(e.target.value))} helperText="Minutes between checks." />
                        <TextField type="number" label="Notify before" value={value.notifyBeforeMinutes} onChange={(e)=>set('notifyBeforeMinutes',Number(e.target.value))} helperText="Minutes before the event start." />
                    </>}
                    <FormControlLabel control={<Switch checked={Boolean(value.enabled)} onChange={(e)=>set('enabled',e.target.checked)} />} label="Enabled" />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
                <Button variant="contained" disabled={saving||!value.name||!value[appIdKey]} onClick={()=>void save()}>Save</Button>
            </DialogActions>
        </Dialog>
    );
};

export default FirstPartyConnectors;
