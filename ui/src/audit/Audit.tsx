import React from 'react';
import {
    Chip,
    InputAdornment,
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
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import {observer} from 'mobx-react-lite';
import {useStores} from '../stores';
import {formatDate} from '../common/TimeAgoFormatter';

const Audit = observer(() => {
    const {auditStore} = useStores();
    const [query, setQuery] = React.useState('');

    React.useEffect(() => void auditStore.refresh(), [auditStore]);

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
