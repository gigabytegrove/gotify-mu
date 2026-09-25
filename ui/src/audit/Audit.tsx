import React from 'react';
import axios from 'axios';
import {
    Button,
    Chip,
    InputAdornment,
    MenuItem,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    TextField,
    Typography,
} from '@mui/material';
import Search from '@mui/icons-material/Search';
import Download from '@mui/icons-material/Download';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import {observer} from 'mobx-react-lite';
import {useStores} from '../stores';
import {formatDate} from '../common/TimeAgoFormatter';
import * as config from '../config';
import {IAuditSettings} from '../types';

const Audit = observer(() => {
    const {auditStore} = useStores();
    const [query, setQuery] = React.useState('');
    const [settings, setSettings] = React.useState<IAuditSettings>();

    React.useEffect(() => {
        void auditStore.refresh();
        void axios
            .get<IAuditSettings>(config.get('url') + 'audit/settings')
            .then((response) => setSettings(response.data));
    }, [auditStore]);

    const saveRetention = async (retentionDays: number) => {
        const response = await axios.put<IAuditSettings>(
            config.get('url') + 'audit/settings',
            {retentionDays}
        );
        setSettings(response.data);
        await auditStore.refresh();
    };

    const events = auditStore.getItems();
    const normalized = query.trim().toLowerCase();
    const filtered = normalized
        ? events.filter((event) =>
              [
                  event.username,
                  event.action,
                  event.target,
                  event.targetId,
                  event.details,
                  event.ipAddress,
              ]
                  .filter(Boolean)
                  .some((value) => value!.toLowerCase().includes(normalized))
          )
        : events;

    return (
        <DefaultPage
            title="Audit Log"
            description="Security-sensitive and administrative changes recorded by Gotify MU.">
            <SurfaceCard
                title="Retention & Export"
                subtitle="Control how long administrative history is kept and export a copy when needed.">
                <Stack direction={{xs: 'column', sm: 'row'}} spacing={2} sx={{alignItems: {sm: 'center'}}}>
                    <TextField
                        select
                        label="Keep audit history"
                        value={settings?.retentionDays || 180}
                        onChange={(event) => void saveRetention(Number(event.target.value))}
                        sx={{minWidth: 220}}>
                        <MenuItem value={30}>30 days</MenuItem>
                        <MenuItem value={90}>90 days</MenuItem>
                        <MenuItem value={180}>180 days</MenuItem>
                        <MenuItem value={365}>1 year</MenuItem>
                        <MenuItem value={730}>2 years</MenuItem>
                        <MenuItem value={1825}>5 years</MenuItem>
                    </TextField>
                    <Button
                        startIcon={<Download />}
                        href={config.get('url') + 'audit/export'}
                        target="_blank"
                        rel="noreferrer">
                        Export CSV
                    </Button>
                </Stack>
            </SurfaceCard>

            <SurfaceCard
                title="Recent Activity"
                subtitle={`Showing up to ${events.length} recent administrative events`}>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{mb: 1.5, alignItems: {sm: 'center'}, justifyContent: 'space-between'}}>
                    <TextField
                        value={query}
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder="Search audit events"
                        aria-label="Search audit events"
                        sx={{width: {xs: '100%', sm: 360}}}
                        slotProps={{
                            input: {
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <Search fontSize="small" />
                                    </InputAdornment>
                                ),
                            },
                        }}
                    />
                    <Chip size="small" variant="outlined" label={`${filtered.length} shown`} />
                </Stack>

                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Time</TableCell>
                            <TableCell>User</TableCell>
                            <TableCell>Action</TableCell>
                            <TableCell>Target</TableCell>
                            <TableCell>IP Address</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {filtered.map((event) => (
                            <TableRow key={event.id} hover>
                                <TableCell title={event.createdAt}>
                                    {formatDate(event.createdAt)}
                                </TableCell>
                                <TableCell>{event.username || 'System / anonymous'}</TableCell>
                                <TableCell>
                                    <Chip
                                        size="small"
                                        variant="outlined"
                                        label={event.action.toUpperCase()}
                                    />
                                </TableCell>
                                <TableCell>
                                    <Typography variant="body2" sx={{fontFamily: 'monospace'}}>
                                        {event.target}
                                    </Typography>
                                </TableCell>
                                <TableCell>{event.ipAddress || '—'}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>

                {filtered.length === 0 && (
                    <Typography color="text.secondary" sx={{py: 3, textAlign: 'center'}}>
                        No audit events match your search.
                    </Typography>
                )}
            </SurfaceCard>
        </DefaultPage>
    );
});

export default Audit;
