import React from 'react';
import {Link} from 'react-router';
import {
    Button,
    Chip,
    Switch,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
} from '@mui/material';
import Settings from '@mui/icons-material/Settings';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import CopyableSecret from '../common/CopyableSecret';
import {formatDate} from '../common/TimeAgoFormatter';
import {observer} from 'mobx-react-lite';
import {IPlugin} from '../types';
import {useStores} from '../stores';

const Plugins = observer(() => {
    const {pluginStore} = useStores();

    React.useEffect(() => void pluginStore.refresh(), []);

    const plugins = pluginStore.getItems();

    return (
        <DefaultPage
            title="Plugins"
            description="Manage server-side Gotify plugins and their configuration.">
            <SurfaceCard
                title="Installed Plugins"
                subtitle={`${plugins.length} plugin${plugins.length === 1 ? '' : 's'} installed`}>
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
                        {plugins.map((plugin: IPlugin) => (
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
