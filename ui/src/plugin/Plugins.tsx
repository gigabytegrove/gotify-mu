import React from 'react';
import {Link} from 'react-router';
import {
    Button,
    Chip,
    InputAdornment,
    Stack,
    Switch,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    TextField,
    ToggleButton,
    ToggleButtonGroup,
    Typography,
} from '@mui/material';
import Settings from '@mui/icons-material/Settings';
import Search from '@mui/icons-material/Search';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import CopyableSecret from '../common/CopyableSecret';
import {formatDate} from '../common/TimeAgoFormatter';
import {observer} from 'mobx-react-lite';
import {IPlugin} from '../types';
import {useStores} from '../stores';

const Plugins = observer(() => {
    const {pluginStore} = useStores();
    const [query, setQuery] = React.useState('');
    const [filter, setFilter] = React.useState<'all' | 'enabled' | 'disabled'>('all');

    React.useEffect(() => void pluginStore.refresh(), []);

    const plugins = pluginStore.getItems();
    const normalizedQuery = query.trim().toLowerCase();
    const filteredPlugins = plugins.filter((plugin) => {
        if (filter === 'enabled' && !plugin.enabled) return false;
        if (filter === 'disabled' && plugin.enabled) return false;
        if (!normalizedQuery) return true;
        return (
            plugin.name.toLowerCase().includes(normalizedQuery) ||
            plugin.id.toString().includes(normalizedQuery)
        );
    });
    const enabledCount = plugins.filter((plugin) => plugin.enabled).length;

    return (
        <DefaultPage
            title="Plugins"
            description="Manage server-side Gotify plugins and their configuration.">
            <SurfaceCard
                title="Installed Plugins"
                subtitle={`${plugins.length} plugin${plugins.length === 1 ? '' : 's'} installed · ${enabledCount} enabled`}>
                <Stack
                    direction={{xs: 'column', md: 'row'}}
                    spacing={1}
                    sx={{mb: 1.5, alignItems: {md: 'center'}, justifyContent: 'space-between'}}>
                    <TextField
                        value={query}
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder="Search plugins"
                        aria-label="Search plugins"
                        sx={{width: {xs: '100%', md: 320}}}
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
                    <ToggleButtonGroup
                        size="small"
                        exclusive
                        value={filter}
                        onChange={(_event, value) => value && setFilter(value)}>
                        <ToggleButton value="all">All</ToggleButton>
                        <ToggleButton value="enabled">Enabled</ToggleButton>
                        <ToggleButton value="disabled">Disabled</ToggleButton>
                    </ToggleButtonGroup>
                </Stack>
                {filteredPlugins.length === 0 && (
                    <Typography color="text.secondary" sx={{py: 3, textAlign: 'center'}}>
                        No plugins match this search or filter.
                    </Typography>
                )}
                <Table id="plugin-table">
                    <TableHead>
                        <TableRow>
                            <TableCell>ID</TableCell>
                            <TableCell>Status</TableCell>
                            <TableCell>Name</TableCell>
                            <TableCell>Token</TableCell>
                            <TableCell>Created</TableCell>
                            <TableCell align="right">Configuration</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {filteredPlugins.map((plugin: IPlugin) => (
                            <PluginRow
                                key={plugin.token}
                                plugin={plugin}
                                fToggleStatus={() =>
                                    pluginStore.changeEnabledState(plugin.id, !plugin.enabled)
                                }
                            />
                        ))}
                    </TableBody>
                </Table>
            </SurfaceCard>
        </DefaultPage>
    );
});

const PluginRow = observer(
    ({plugin, fToggleStatus}: {plugin: IPlugin; fToggleStatus: VoidFunction}) => (
        <TableRow hover>
            <TableCell>{plugin.id}</TableCell>
            <TableCell>
                <Switch
                    size="small"
                    checked={plugin.enabled}
                    onClick={fToggleStatus}
                    className="switch"
                    data-enabled={plugin.enabled}
                />
                <Chip
                    size="small"
                    label={plugin.enabled ? 'Enabled' : 'Disabled'}
                    variant={plugin.enabled ? 'filled' : 'outlined'}
                />
            </TableCell>
            <TableCell>
                <strong>{plugin.name}</strong>
            </TableCell>
            <TableCell>
                <CopyableSecret
                    value={plugin.token}
                    style={{display: 'flex', alignItems: 'center'}}
                />
            </TableCell>
            <TableCell title={plugin.createdAt}>{formatDate(plugin.createdAt)}</TableCell>
            <TableCell align="right">
                <Button
                    size="small"
                    component={Link}
                    to={`/plugins/${plugin.id}`}
                    startIcon={<Settings />}>
                    Configure
                </Button>
            </TableCell>
        </TableRow>
    )
);

export default Plugins;
