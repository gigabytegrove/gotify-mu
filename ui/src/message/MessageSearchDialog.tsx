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
    MenuItem,
    Stack,
    TextField,
    Typography,
} from '@mui/material';
import Delete from '@mui/icons-material/Delete';
import Search from '@mui/icons-material/Search';
import Save from '@mui/icons-material/Save';
import * as config from '../config';
import {IMessage, ISavedMessageSearch} from '../types';
import {useStores} from '../stores';

const api = (path: string) => config.get('url') + path;

interface Props {
    open: boolean;
    onClose: VoidFunction;
    initialApplicationId?: number;
}

const MessageSearchDialog = ({open, onClose, initialApplicationId}: Props) => {
    const {appStore, snackManager} = useStores();
    const [query, setQuery] = React.useState('');
    const [applicationId, setApplicationId] = React.useState(initialApplicationId || 0);
    const [minPriority, setMinPriority] = React.useState('');
    const [maxPriority, setMaxPriority] = React.useState('');
    const [sender, setSender] = React.useState('');
    const [status, setStatus] = React.useState('');
    const [acknowledged, setAcknowledged] = React.useState('');
    const [results, setResults] = React.useState<IMessage[]>([]);
    const [saved, setSaved] = React.useState<ISavedMessageSearch[]>([]);
    const [searching, setSearching] = React.useState(false);
    const [saveName, setSaveName] = React.useState('');

    const loadSaved = React.useCallback(async () => {
        const response = await axios.get<ISavedMessageSearch[]>(api('saved-search'));
        setSaved(response.data);
    }, []);

    React.useEffect(() => {
        if (open) void loadSaved();
    }, [open, loadSaved]);

    const search = async () => {
        setSearching(true);
        try {
            const response = await axios.get<IMessage[]>(api('message/search'), {
                params: {
                    q: query || undefined,
                    applicationId: applicationId || undefined,
                    minPriority: minPriority || undefined,
                    maxPriority: maxPriority || undefined,
                    sender: sender || undefined,
                    status: status || undefined,
                    acknowledged: acknowledged || undefined,
                    limit: 500,
                },
            });
            setResults(response.data);
        } finally {
            setSearching(false);
        }
    };

    const saveSearch = async () => {
        const name = saveName.trim();
        if (!name) return;
        await axios.post(api('saved-search'), {
            name,
            query,
            applicationId,
            minPriority: Number(minPriority || 0),
            maxPriority: Number(maxPriority || 0),
            sender,
            status,
            acknowledged,
        });
        setSaveName('');
        await loadSaved();
        snackManager.snack('Search saved');
    };

    const apply = (item: ISavedMessageSearch) => {
        setQuery(item.query || '');
        setApplicationId(item.applicationId || 0);
        setMinPriority(item.minPriority ? String(item.minPriority) : '');
        setMaxPriority(item.maxPriority ? String(item.maxPriority) : '');
        setSender(item.sender || '');
        setStatus(item.status || '');
        setAcknowledged(item.acknowledged || '');
    };

    return (
        <Dialog open={open} onClose={onClose} fullWidth maxWidth="lg">
            <DialogTitle>Advanced Message Search</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{pt: 1}}>
                    {saved.length > 0 && (
                        <Stack spacing={1}>
                            <Typography variant="subtitle2">Saved searches</Typography>
                            <Stack direction="row" spacing={0.75} useFlexGap sx={{flexWrap: 'wrap'}}>
                                {saved.map((item) => (
                                    <Chip
                                        key={item.id}
                                        label={item.name}
                                        clickable
                                        onClick={() => apply(item)}
                                        onDelete={async () => {
                                            await axios.delete(api('saved-search/' + item.id));
                                            await loadSaved();
                                        }}
                                        deleteIcon={<Delete />}
                                    />
                                ))}
                            </Stack>
                        </Stack>
                    )}

                    <Box
                        sx={{
                            display: 'grid',
                            gridTemplateColumns: {xs: '1fr', md: 'repeat(3, 1fr)'},
                            gap: 1.5,
                        }}>
                        <TextField
                            label="Words or phrase"
                            value={query}
                            onChange={(event) => setQuery(event.target.value)}
                        />
                        <TextField
                            select
                            label="Channel"
                            value={applicationId}
                            onChange={(event) => setApplicationId(Number(event.target.value))}>
                            <MenuItem value={0}>All Channels</MenuItem>
                            {appStore.getItems().map((app) => (
                                <MenuItem key={app.id} value={app.id}>
                                    {app.name}
                                </MenuItem>
                            ))}
                        </TextField>
                        <TextField
                            label="Sender"
                            value={sender}
                            onChange={(event) => setSender(event.target.value)}
                        />
                        <TextField
                            type="number"
                            label="Minimum priority"
                            value={minPriority}
                            onChange={(event) => setMinPriority(event.target.value)}
                        />
                        <TextField
                            type="number"
                            label="Maximum priority"
                            value={maxPriority}
                            onChange={(event) => setMaxPriority(event.target.value)}
                        />
                        <TextField
                            select
                            label="Status"
                            value={status}
                            onChange={(event) => setStatus(event.target.value)}>
                            <MenuItem value="">Any status</MenuItem>
                            <MenuItem value="open">Open</MenuItem>
                            <MenuItem value="resolved">Resolved</MenuItem>
                        </TextField>
                        <TextField
                            select
                            label="Acknowledgement"
                            value={acknowledged}
                            onChange={(event) => setAcknowledged(event.target.value)}>
                            <MenuItem value="">Any</MenuItem>
                            <MenuItem value="yes">Acknowledged</MenuItem>
                            <MenuItem value="no">Not acknowledged</MenuItem>
                        </TextField>
                    </Box>

                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                        <Button
                            variant="contained"
                            startIcon={<Search />}
                            disabled={searching}
                            onClick={() => void search()}>
                            Search
                        </Button>
                        <TextField
                            size="small"
                            label="Save this search as"
                            value={saveName}
                            onChange={(event) => setSaveName(event.target.value)}
                        />
                        <Button
                            variant="outlined"
                            startIcon={<Save />}
                            disabled={!saveName.trim()}
                            onClick={() => void saveSearch()}>
                            Save Search
                        </Button>
                    </Stack>

                    <Stack spacing={1}>
                        <Typography variant="subtitle2">
                            {results.length} result{results.length === 1 ? '' : 's'}
                        </Typography>
                        {results.map((message) => (
                            <Box
                                key={message.id}
                                sx={{border: 1, borderColor: 'divider', borderRadius: 2, p: 1.25}}>
                                <Stack
                                    direction={{xs: 'column', sm: 'row'}}
                                    spacing={1}
                                    sx={{justifyContent: 'space-between'}}>
                                    <Box>
                                        <Typography sx={{fontWeight: 700}}>
                                            {message.title || appStore.getName(message.appid)}
                                        </Typography>
                                        <Typography variant="caption" color="text.secondary">
                                            {appStore.getName(message.appid)}
                                            {message.senderName ? ' · ' + message.senderName : ''}
                                            {' · '}
                                            {new Date(message.date).toLocaleString()}
                                        </Typography>
                                    </Box>
                                    <Stack direction="row" spacing={0.5}>
                                        {message.acknowledgedByAnyone && (
                                            <Chip size="small" color="success" label="Acknowledged" />
                                        )}
                                        {message.collaboration?.status === 'resolved' && (
                                            <Chip size="small" color="success" variant="outlined" label="Resolved" />
                                        )}
                                    </Stack>
                                </Stack>
                                <Typography
                                    variant="body2"
                                    sx={{whiteSpace: 'pre-wrap', mt: 0.75}}>
                                    {message.message}
                                </Typography>
                            </Box>
                        ))}
                    </Stack>
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Close</Button>
            </DialogActions>
        </Dialog>
    );
};

export default MessageSearchDialog;
