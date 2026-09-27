import React from 'react';
import axios from 'axios';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import LinearProgress from '@mui/material/LinearProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import NewReleases from '@mui/icons-material/NewReleases';
import SystemUpdateAlt from '@mui/icons-material/SystemUpdateAlt';
import Refresh from '@mui/icons-material/Refresh';
import {Link} from 'react-router';
import SurfaceCard from '../common/SurfaceCard';
import {useStores} from '../stores';
import * as config from '../config';
import {
    canInstallPublishedRelease,
    classifyUpdate,
    latestPublishedRelease,
    PublishedRelease,
    RELEASES_API,
    UpdateClassification,
} from './release';
import {activeUpdaterStates, updaterStatusForDisplay, type UpdaterStatus} from './status';

type ReleaseState =
    | {status: 'loading'}
    | {status: 'none'}
    | {status: 'error'; message: string}
    | {status: 'ready'; release: PublishedRelease; classification: UpdateClassification};

const releaseLabel = (release: PublishedRelease) => release.name || release.tag_name;
const normalizeTag = (tag: string) => tag.replace(/^v/i, '');

export const useReleaseUpdate = (refreshKey = 0): ReleaseState => {
    const [state, setState] = React.useState<ReleaseState>({status: 'loading'});
    const currentVersion = config.get('version').version;

    React.useEffect(() => {
        const controller = new AbortController();

        const check = async () => {
            try {
                const response = await fetch(RELEASES_API, {
                    headers: {Accept: 'application/vnd.github+json'},
                    signal: controller.signal,
                });
                if (!response.ok) {
                    throw new Error(`GitHub returned HTTP ${response.status}`);
                }

                const releases = (await response.json()) as PublishedRelease[];
                const release = latestPublishedRelease(releases);
                if (!release) {
                    setState({status: 'none'});
                    return;
                }

                setState({
                    status: 'ready',
                    release,
                    classification: classifyUpdate(currentVersion, normalizeTag(release.tag_name)),
                });
            } catch (error) {
                if (controller.signal.aborted) return;
                setState({
                    status: 'error',
                    message: error instanceof Error ? error.message : 'Release check failed',
                });
            }
        };

        void check();
        return () => controller.abort();
    }, [currentVersion, refreshKey]);

    return state;
};

export const UpdateAvailableBanner = () => {
    const state = useReleaseUpdate();
    const currentVersion = config.get('version').version;

    if (state.status !== 'ready') return null;
    if (state.classification !== 'available' && state.classification !== 'development') return null;

    const development = state.classification === 'development';

    return (
        <Alert
            severity={development ? 'info' : 'success'}
            icon={<NewReleases />}
            action={
                <Button
                    color="inherit"
                    size="small"
                    component={Link}
                    to="/settings"
                    endIcon={<SystemUpdateAlt fontSize="small" />}>
                    Review Update
                </Button>
            }>
            {development
                ? `Published release ${state.release.tag_name} is available. This server is running preview version ${currentVersion}.`
                : `Monita ${state.release.tag_name} is available. This server is running ${currentVersion}.`}
        </Alert>
    );
};

export const UpdateStatusCard = () => {
    const [releaseRefreshKey, setReleaseRefreshKey] = React.useState(0);
    const state = useReleaseUpdate(releaseRefreshKey);
    const {elevateStore} = useStores();
    const currentVersion = config.get('version').version;
    const [updater, setUpdater] = React.useState<UpdaterStatus>();
    const [installing, setInstalling] = React.useState(false);
    const updaterRef = React.useRef<UpdaterStatus | undefined>(undefined);
    const updateStartedHere = React.useRef(false);
    const sawActiveUpdate = React.useRef(false);
    const reloadScheduled = React.useRef(false);

    const loadUpdaterStatus = React.useCallback(async () => {
        try {
            const response = await fetch(`${config.get('url')}update/status`, {
                credentials: 'same-origin',
                headers: {Accept: 'application/json'},
            });
            if (!response.ok) {
                let message = `HTTP ${response.status}`;
                try {
                    const payload = (await response.json()) as {
                        message?: string;
                        errorDescription?: string;
                    };
                    message = payload.message || payload.errorDescription || message;
                } catch {
                    // Keep the HTTP status when the response is not JSON.
                }
                const unavailable: UpdaterStatus = {
                    ready: false,
                    state: 'unavailable',
                    message,
                };
                updaterRef.current = unavailable;
                setUpdater(unavailable);
                return;
            }

            const next = (await response.json()) as UpdaterStatus;

            if (activeUpdaterStates.has(next.state)) {
                sawActiveUpdate.current = true;
            }

            const visibleNext = updaterStatusForDisplay(
                next,
                updateStartedHere.current,
                sawActiveUpdate.current
            );
            updaterRef.current = visibleNext;
            setUpdater(visibleNext);

            const completedThisSession =
                updateStartedHere.current && sawActiveUpdate.current && next.state === 'completed';

            if (completedThisSession && !reloadScheduled.current) {
                reloadScheduled.current = true;
                window.setTimeout(() => window.location.reload(), 1500);
            }
        } catch {
            const previous = updaterRef.current;
            if (previous && activeUpdaterStates.has(previous.state)) {
                return;
            }
            const unavailable: UpdaterStatus = {
                ready: false,
                state: 'unavailable',
                message: 'Automatic updates are temporarily unavailable.',
            };
            updaterRef.current = unavailable;
            setUpdater(unavailable);
        }
    }, []);

    React.useEffect(() => {
        void loadUpdaterStatus();
        const interval = window.setInterval(() => void loadUpdaterStatus(), 2500);
        return () => window.clearInterval(interval);
    }, [loadUpdaterStatus]);

    const installRelease = async (release: PublishedRelease) => {
        if (!elevateStore.elevated) {
            elevateStore.requestReauthentication();
            return;
        }

        if (state.status === 'ready' && state.classification === 'development') {
            const confirmed = window.confirm(
                `Replace preview build ${currentVersion} with published ${release.tag_name}? Your application data will be preserved.`
            );
            if (!confirmed) return;
        }

        setInstalling(true);
        updateStartedHere.current = true;
        sawActiveUpdate.current = false;
        reloadScheduled.current = false;

        try {
            await axios.post(`${config.get('url')}update/install`, {
                version: normalizeTag(release.tag_name),
            });
            await loadUpdaterStatus();
        } finally {
            setInstalling(false);
        }
    };

    const updaterBusy = Boolean(updater && activeUpdaterStates.has(updater.state));

    return (
        <SurfaceCard
            title="Software Update"
            subtitle="Check for a new Monita release and install it automatically."
            action={
                <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                    <Button
                        size="small"
                        variant="outlined"
                        startIcon={<Refresh fontSize="small" />}
                        disabled={state.status === 'loading'}
                        onClick={() => setReleaseRefreshKey((value) => value + 1)}>
                        Check for Updates
                    </Button>
                    <NewReleases color="action" />
                </Stack>
            }>
            {state.status === 'loading' && (
                <Stack direction="row" spacing={1.25} sx={{alignItems: 'center'}}>
                    <CircularProgress size={20} />
                    <Typography color="text.secondary">Checking for updates…</Typography>
                </Stack>
            )}

            {state.status === 'none' && (
                <Alert severity="info">No published Monita release is available yet.</Alert>
            )}

            {state.status === 'error' && (
                <Alert severity="warning">Could not check GitHub releases: {state.message}</Alert>
            )}

            {state.status === 'ready' && (
                <ReleaseUpdateDetails
                    state={state}
                    updater={updater}
                    installing={installing}
                    updaterBusy={updaterBusy}
                    currentVersion={currentVersion}
                    installRelease={installRelease}
                    elevated={elevateStore.elevated}
                />
            )}
        </SurfaceCard>
    );
};

const ReleaseUpdateDetails = ({
    state,
    updater,
    installing,
    updaterBusy,
    currentVersion,
    installRelease,
    elevated,
}: {
    state: Extract<ReleaseState, {status: 'ready'}>;
    updater?: UpdaterStatus;
    installing: boolean;
    updaterBusy: boolean;
    currentVersion: string;
    installRelease: (release: PublishedRelease) => Promise<void>;
    elevated: boolean;
}) => {
    const safeAutomaticInstall = canInstallPublishedRelease(state.classification);
    const updaterReady = updater?.ready === true;

    return (
        <Stack spacing={2}>
            <Stack
                direction={{xs: 'column', sm: 'row'}}
                spacing={1}
                sx={{alignItems: {sm: 'center'}, justifyContent: 'space-between'}}>
                <Stack spacing={0.35}>
                    <Typography variant="body2" color="text.secondary">
                        Installed
                    </Typography>
                    <Typography sx={{fontWeight: 700}}>{currentVersion}</Typography>
                </Stack>
                <Stack spacing={0.35} sx={{alignItems: {sm: 'flex-end'}}}>
                    <Typography variant="body2" color="text.secondary">
                        Latest published release
                    </Typography>
                    <Stack direction="row" spacing={0.75} sx={{alignItems: 'center'}}>
                        <Typography sx={{fontWeight: 700}}>
                            {releaseLabel(state.release)}
                        </Typography>
                        {state.release.prerelease && (
                            <Chip size="small" variant="outlined" label="Pre-release" />
                        )}
                    </Stack>
                </Stack>
            </Stack>

            {state.classification === 'available' && (
                <Alert severity="success">
                    An update is available: {currentVersion} → {state.release.tag_name}
                </Alert>
            )}
            {state.classification === 'current' && (
                <Alert severity="success">
                    This server is running the latest published release.
                </Alert>
            )}
            {state.classification === 'newer' && (
                <Alert severity="info">
                    This server is newer than the latest published release.
                </Alert>
            )}
            {state.classification === 'development' && (
                <Alert severity="warning">
                    This server is running preview build {currentVersion}. Installing{' '}
                    {state.release.tag_name} will switch this server to the published release.
                    Application data is preserved during the update.
                </Alert>
            )}

            <Stack
                direction={{xs: 'column', sm: 'row'}}
                spacing={1}
                sx={{alignItems: {sm: 'center'}, justifyContent: 'space-between'}}>
                <Stack direction="row" spacing={0.75} sx={{alignItems: 'center'}}>
                    <Typography variant="body2" color="text.secondary">
                        Automatic updates
                    </Typography>
                    <Chip
                        size="small"
                        color={updaterReady ? 'success' : 'default'}
                        variant={updaterReady ? 'filled' : 'outlined'}
                        label={updaterReady ? 'Ready' : 'Unavailable'}
                    />
                </Stack>

                <Button
                    variant="contained"
                    startIcon={
                        updaterBusy || installing ? (
                            <CircularProgress size={16} color="inherit" />
                        ) : (
                            <SystemUpdateAlt />
                        )
                    }
                    disabled={!safeAutomaticInstall || !updaterReady || updaterBusy || installing}
                    onClick={() => void installRelease(state.release)}>
                    {updaterBusy
                        ? 'Updating…'
                        : !elevated
                          ? `Re-authenticate to install ${state.release.tag_name}`
                          : `Install ${state.release.tag_name}`}
                </Button>
            </Stack>

            {updater && updater.state !== 'idle' && updater.state !== 'unavailable' && (
                <Stack spacing={1.25}>
                    <Stack
                        direction="row"
                        spacing={1}
                        sx={{alignItems: 'center', justifyContent: 'space-between'}}>
                        <Typography sx={{fontWeight: 700}}>
                            {updater.step || 'Preparing update'}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                            {Math.max(0, Math.min(100, updater.progress ?? 0))}%
                        </Typography>
                    </Stack>
                    <LinearProgress
                        variant="determinate"
                        value={Math.max(0, Math.min(100, updater.progress ?? 0))}
                    />
                    {updater.message && (
                        <Typography variant="body2" color="text.secondary">
                            {updater.message}
                        </Typography>
                    )}

                    {updater.activity && updater.activity.length > 0 && (
                        <Stack spacing={0.5}>
                            <Typography variant="subtitle2">Update activity</Typography>
                            <Stack
                                spacing={0.5}
                                sx={{
                                    maxHeight: 220,
                                    overflowY: 'auto',
                                    p: 1.25,
                                    borderRadius: 1.5,
                                    bgcolor: 'action.hover',
                                }}>
                                {updater.activity.slice(-12).map((entry, index) => (
                                    <Stack
                                        key={`${entry.timestamp}-${index}`}
                                        direction="row"
                                        spacing={1}
                                        sx={{alignItems: 'baseline'}}>
                                        <Typography
                                            variant="caption"
                                            color="text.secondary"
                                            sx={{minWidth: 74}}>
                                            {new Date(entry.timestamp).toLocaleTimeString([], {
                                                hour: 'numeric',
                                                minute: '2-digit',
                                                second: '2-digit',
                                            })}
                                        </Typography>
                                        <Typography variant="body2">{entry.message}</Typography>
                                    </Stack>
                                ))}
                            </Stack>
                        </Stack>
                    )}
                </Stack>
            )}

            {(updater?.state === 'failed' || updater?.state === 'rolled_back') &&
                updater.message && <Alert severity="warning">{updater.message}</Alert>}

            {!updaterReady && (
                <Alert severity="warning">
                    {updater?.message || 'Managed updater status is unavailable.'}
                </Alert>
            )}
        </Stack>
    );
};
