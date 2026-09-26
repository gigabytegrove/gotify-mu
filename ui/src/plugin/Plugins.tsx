import React from 'react';
import {Link} from 'react-router';
import {
    Alert,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
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
import UploadFile from '@mui/icons-material/UploadFile';
import Add from '@mui/icons-material/Add';
import Delete from '@mui/icons-material/Delete';
import Download from '@mui/icons-material/Download';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import CopyableSecret from '../common/CopyableSecret';
import ElevationForm from '../common/ElevationForm';
import ConfirmDialog from '../common/ConfirmDialog';
import {formatDate} from '../common/TimeAgoFormatter';
import {observer} from 'mobx-react-lite';
import {IPlugin, IPluginCatalogEntry} from '../types';
import {useStores} from '../stores';

const Plugins = observer(() => {
    const {pluginStore, currentUser} = useStores();
    const [query, setQuery] = React.useState('');
    const [filter, setFilter] = React.useState<'all' | 'enabled' | 'disabled'>('all');
    const [installOpen, setInstallOpen] = React.useState(false);
    const [catalog, setCatalog] = React.useState<IPluginCatalogEntry[]>([]);
    const [confirm, setConfirm] = React.useState<{title: string; text: string; action: VoidFunction}>();

    React.useEffect(() => void pluginStore.refresh(), []);
    React.useEffect(() => {
        if (currentUser.user.admin) {
            void pluginStore.getCatalog().then(setCatalog).catch(() => setCatalog([]));
        }
    }, [currentUser.user.admin, pluginStore]);

    const plugins = pluginStore.getItems();
    const normalizedQuery = query.trim().toLowerCase();
    const filteredPlugins = plugins.filter((plugin) => {
        if (filter === 'enabled' && !plugin.enabled) return false;
        if (filter === 'disabled' && plugin.enabled) return false;
        if (!normalizedQuery) return true;
        return (
            plugin.name.toLowerCase().includes(normalizedQuery) ||
            plugin.modulePath.toLowerCase().includes(normalizedQuery) ||
            plugin.id.toString().includes(normalizedQuery)
        );
    });
    const enabledCount = plugins.filter((plugin) => plugin.enabled).length;

    return (
        <DefaultPage
            title="Plugins"
            description="Install, enable, and configure server-side Gotify plugins."
            rightControl={
                currentUser.user.admin ? (
                    <Button
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => setInstallOpen(true)}>
                        Install Plugin
                    </Button>
                ) : undefined
            }>
            {currentUser.user.admin && catalog.length > 0 && (
                <SurfaceCard
                    title="Plugin Catalog"
                    subtitle="Signed plugins available from the server's configured catalog.">
                    <Stack spacing={1}>
                        {catalog.map((entry) => (
                            <Stack
                                key={entry.modulePath + '@' + entry.version}
                                direction={{xs: 'column', sm: 'row'}}
                                spacing={1}
                                sx={{
                                    border: 1,
                                    borderColor: 'divider',
                                    borderRadius: 2,
                                    p: 1.5,
                                    justifyContent: 'space-between',
                                }}>
                                <Stack spacing={0.25}>
                                    <Typography sx={{fontWeight: 700}}>
                                        {entry.name} · {entry.version}
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        {entry.description}
                                    </Typography>
                                    <Typography variant="caption" color="text.secondary">
                                        {entry.modulePath}
                                    </Typography>
                                </Stack>
                                <Button
                                    startIcon={<Download />}
                                    variant="outlined"
                                    onClick={() => void pluginStore.installCatalogPlugin(entry)}>
                                    {entry.installed ? 'Stage Update' : 'Install'}
                                </Button>
                            </Stack>
                        ))}
                    </Stack>
                </SurfaceCard>
            )}

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
                        sx={{width: {xs: '100%', md: 360}}}
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
                                admin={currentUser.user.admin}
                                fToggleStatus={() =>
                                    pluginStore.changeEnabledState(plugin.id, !plugin.enabled)
                                }
                                fUninstall={() =>
                                    setConfirm({
                                        title: 'Uninstall Plugin?',
                                        text: 'The plugin binary will be removed and its instances disabled. Stored configuration and historical messages are retained.',
                                        action: () => void pluginStore.uninstallPlugin(plugin.id),
                                    })
                                }
                            />
                        ))}
                    </TableBody>
                </Table>
            </SurfaceCard>

            {installOpen && <PluginInstallDialog fClose={() => setInstallOpen(false)} />}
            {confirm && (
                <ConfirmDialog
                    title={confirm.title}
                    text={confirm.text}
                    requireElevated
                    fClose={() => setConfirm(undefined)}
                    fOnSubmit={confirm.action}
                />
            )}
        </DefaultPage>
    );
});

const PluginInstallDialog = observer(({fClose}: {fClose: VoidFunction}) => {
    const {pluginStore, elevateStore} = useStores();
    const [file, setFile] = React.useState<File>();
    const [sha256, setSha256] = React.useState('');
    const [signature, setSignature] = React.useState('');
    const [publicKey, setPublicKey] = React.useState('');
    const [installing, setInstalling] = React.useState(false);

    const close = () => {
        elevateStore.cleanupOidcElevate();
        fClose();
    };

    const install = async () => {
        if (!file || installing) return;

        setInstalling(true);
        try {
            await pluginStore.installPlugin(file, {sha256, signature, publicKey});
            close();
        } finally {
            setInstalling(false);
        }
    };

    return (
        <Dialog open onClose={close} fullWidth maxWidth="sm">
            <DialogTitle>Install Plugin</DialogTitle>
            <DialogContent>
                {!elevateStore.elevated ? (
                    <Stack spacing={2} sx={{pt: 0.5}}>
                        <Typography color="text.secondary">
                            Plugin installation changes the server and requires administrator
                            re-authentication.
                        </Typography>
                        <ElevationForm />
                    </Stack>
                ) : (
                    <Stack spacing={2} sx={{pt: 0.5}}>
                        <Alert severity="warning">
                            Plugins execute native code inside Gotify MU with the same access as the
                            server. Install binaries only from sources you trust.
                        </Alert>

                        <Button
                            component="label"
                            variant="outlined"
                            startIcon={<UploadFile />}
                            sx={{alignSelf: 'flex-start'}}>
                            Choose .so Plugin
                            <input
                                hidden
                                type="file"
                                accept=".so,application/octet-stream"
                                onChange={(event) => {
                                    const selected = event.target.files?.[0];
                                    setFile(selected);
                                    event.target.value = '';
                                }}
                            />
                        </Button>

                        {file ? (
                            <Stack spacing={0.25}>
                                <Typography sx={{fontWeight: 700}}>{file.name}</Typography>
                                <Typography variant="body2" color="text.secondary">
                                    {(file.size / 1024 / 1024).toFixed(1)} MiB
                                </Typography>
                            </Stack>
                        ) : (
                            <Typography variant="body2" color="text.secondary">
                                Select a Linux Go plugin binary ending in .so.
                            </Typography>
                        )}

                        <TextField
                            label="SHA-256 checksum"
                            value={sha256}
                            onChange={(event) => setSha256(event.target.value)}
                            helperText="Required when your server does not permit unsigned plugin installs."
                        />
                        <TextField
                            label="Ed25519 signature"
                            value={signature}
                            onChange={(event) => setSignature(event.target.value)}
                            multiline
                            minRows={2}
                        />
                        <TextField
                            label="Signing public key"
                            value={publicKey}
                            onChange={(event) => setPublicKey(event.target.value)}
                            multiline
                            minRows={2}
                            helperText="The key must already be trusted by the server."
                        />
                        <Alert severity="info">
                            Plugins are native server code. Gotify MU verifies configured checksums
                            and trusted Ed25519 signatures before loading new uploads. Unsigned
                            installation is disabled unless the server administrator explicitly
                            opts in.
                        </Alert>
                    </Stack>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={close}>Cancel</Button>
                {elevateStore.elevated && (
                    <Button
                        variant="contained"
                        disabled={!file || installing}
                        loading={installing}
                        onClick={() => void install()}>
                        Install Plugin
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
});

const PluginRow = observer(
    ({
        plugin,
        fToggleStatus,
        fUninstall,
        admin,
    }: {
        plugin: IPlugin;
        fToggleStatus: VoidFunction;
        fUninstall: VoidFunction;
        admin: boolean;
    }) => (
        <TableRow hover>
            <TableCell>{plugin.id}</TableCell>
            <TableCell>
                <Stack direction="row" spacing={0.75} sx={{alignItems: 'center'}}>
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
                </Stack>
            </TableCell>
            <TableCell>
                <Stack spacing={0.2}>
                    <Typography sx={{fontWeight: 700}}>{plugin.name}</Typography>
                    <Typography variant="caption" color="text.secondary">
                        {plugin.modulePath}
                    </Typography>
                </Stack>
            </TableCell>
            <TableCell>
                <CopyableSecret
                    value={plugin.token}
                    style={{display: 'flex', alignItems: 'center'}}
                />
            </TableCell>
            <TableCell title={plugin.createdAt}>{formatDate(plugin.createdAt)}</TableCell>
            <TableCell align="right">
                <Stack direction="row" spacing={0.5} sx={{justifyContent: 'flex-end'}}>
                    <Button
                        size="small"
                        component={Link}
                        to={`/plugins/${plugin.id}`}
                        startIcon={<Settings />}>
                        Configure
                    </Button>
                    {admin && (
                        <Button
                            size="small"
                            color="error"
                            startIcon={<Delete />}
                            onClick={fUninstall}>
                            Uninstall
                        </Button>
                    )}
                </Stack>
            </TableCell>
        </TableRow>
    )
);

export default Plugins;
