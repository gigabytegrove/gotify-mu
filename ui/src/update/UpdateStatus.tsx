import React from 'react';
import axios from 'axios';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import LinearProgress from '@mui/material/LinearProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Download from '@mui/icons-material/Download';
import NewReleases from '@mui/icons-material/NewReleases';
import OpenInNew from '@mui/icons-material/OpenInNew';
import SystemUpdateAlt from '@mui/icons-material/SystemUpdateAlt';
import {Link} from 'react-router';
import SurfaceCard from '../common/SurfaceCard';
import * as config from '../config';
import {
    classifyUpdate,
    latestPublishedRelease,
    PublishedRelease,
    RELEASES_API,
    UpdateClassification,
} from './release';

type ReleaseState =
    | {status: 'loading'}
    | {status: 'none'}
    | {status: 'error'; message: string}
    | {status: 'ready'; release: PublishedRelease; classification: UpdateClassification};

interface UpdateActivity {
    timestamp: string;
    message: string;
}

interface UpdaterStatus {
    ready: boolean;
    state: string;
    version?: string;
    message?: string;
    step?: string;
    progress?: number;
    activity?: UpdateActivity[];
    startedAt?: string;
    finishedAt?: string;
}

const releaseLabel = (release: PublishedRelease) => release.name || release.tag_name;
const normalizeTag = (tag: string) => tag.replace(/^v/i, '');
const activeUpdaterStates = new Set(['preparing', 'downloading', 'building', 'replacing', 'verifying']);

export const useReleaseUpdate = (): ReleaseState => {
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
    }, [currentVersion]);

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
                : `Gotify MU ${state.release.tag_name} is available. This server is running ${currentVersion}.`}
        </Alert>
    );
};

export const UpdateStatusCard = () => {
    const state = useReleaseUpdate();
    const current = config.get('version');
    const currentVersion = current.version;
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
                throw new Error(`HTTP ${response.status}`);
            }

            const next = (await response.json()) as UpdaterStatus;
            updaterRef.current = next;
            setUpdater(next);

            if (activeUpdaterStates.has(next.state)) {
                sawActiveUpdate.current = true;
            }

            const completedThisSession =
                updateStartedHere.current &&
                sawActiveUpdate.current &&
                next.state === 'completed';

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
            subtitle="Install available updates here. If an update cannot be completed safely, the previous version is restored automatically."
            action={<NewReleases color="action" />}>
            {state.status === 'loading' && (
                <Stack direction="row" spacing={1.25} sx={{alignItems: 'center'}}>
                    <CircularProgress size={20} />
                    <Typography color="text.secondary">Checking for updates…</Typography>
                </Stack>
            )}

            {state.status === 'none' && (
                <Alert severity="info">No published Gotify MU release is available yet.</Alert>
            )}

            {state.status === 'error' && (
                <Alert severity="warning">
                    Could not check GitHub releases: {state.message}
                </Alert>
            )}

            {state.status === 'ready' && (
                <ReleaseUpdateDetails
                    state={state}
                    updater={updater}
                    installing={installing}
                    updaterBusy={updaterBusy}
                    currentVersion={currentVersion}
                    currentCommit={current.commit}
                    installRelease={installRelease}
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
    currentCommit,
    installRelease,
}: {
    state: Extract<ReleaseState, {status: 'ready'}>;
    updater?: UpdaterStatus;
    installing: boolean;
    updaterBusy: boolean;
    currentVersion: string;
    currentCommit: string;
    installRelease: (release: PublishedRelease) => Promise<void>;
}) => {
    const sameCommit =
        Boolean(currentCommit) &&
        Boolean(state.release.target_commitish) &&
        currentCommit === state.release.target_commitish;
    const safeAutomaticInstall =
        state.classification === 'available' ||
        (state.classification === 'development' && sameCommit);
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
            {state.classification === 'development' && sameCommit && (
                <Alert severity="success">
                    This preview version matches {state.release.tag_name}. You can install the
                    published version without changing your application data.
                </Alert>
            )}
            {state.classification === 'development' && !sameCommit && (
                <Alert severity="info">
                    This server is running a preview version newer than {state.release.tag_name}.
                    Installing the older published version is disabled to prevent an accidental
                    downgrade.
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
                    {updaterBusy ? 'Updating…' : `Install ${state.release.tag_name}`}
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
                <Typography variant="body2" color="text.secondary">
                    Automatic installation is not enabled on this server. Updates can still be
                    downloaded below and installed by the server administrator.
                </Typography>
            )}

            {state.release.assets.length > 0 && (
                <Stack spacing={1}>
                    <Typography variant="subtitle2">Download files</Typography>
                    <Stack direction="row" spacing={1} useFlexGap sx={{flexWrap: 'wrap'}}>
                        {state.release.assets.map((asset) => (
                            <Button
                                key={asset.name}
                                component="a"
                                href={asset.browser_download_url}
                                variant="outlined"
                                size="small"
                                startIcon={<Download />}>
                                {asset.name}
                            </Button>
                        ))}
                    </Stack>
                </Stack>
            )}

            <Button
                component="a"
                href={state.release.html_url}
                target="_blank"
                rel="noreferrer"
                variant="text"
                sx={{alignSelf: 'flex-start'}}
                endIcon={<OpenInNew />}>
                Release Notes
            </Button>
        </Stack>
    );
};
