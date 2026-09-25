import React from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Download from '@mui/icons-material/Download';
import NewReleases from '@mui/icons-material/NewReleases';
import OpenInNew from '@mui/icons-material/OpenInNew';
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

const releaseLabel = (release: PublishedRelease) => release.name || release.tag_name;
const normalizeTag = (tag: string) => tag.replace(/^v/i, '');

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
                    component="a"
                    href={state.release.html_url}
                    target="_blank"
                    rel="noreferrer"
                    endIcon={<OpenInNew fontSize="small" />}>
                    View / Download
                </Button>
            }>
            {development
                ? `Published release ${state.release.tag_name} is available. This server is running development build ${currentVersion}.`
                : `Gotify MU ${state.release.tag_name} is available. This server is running ${currentVersion}.`}
        </Alert>
    );
};

export const UpdateStatusCard = () => {
    const state = useReleaseUpdate();
    const current = config.get('version');
    const currentVersion = current.version;

    return (
        <SurfaceCard
            title="Software Update"
            subtitle="Check the official Gotify MU GitHub releases and download published builds."
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
                            <Typography variant="caption" color="text.secondary">
                                Commit {current.commit || 'unknown'}
                            </Typography>
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
                        <Alert severity="info">
                            This server is running a development build. The latest published release
                            is {state.release.tag_name}.
                        </Alert>
                    )}

                    {state.release.assets.length > 0 ? (
                        <Stack spacing={1}>
                            <Typography variant="subtitle2">Release downloads</Typography>
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
                    ) : (
                        <Alert severity="warning">
                            This release does not currently have downloadable build assets.
                        </Alert>
                    )}

                    <Button
                        component="a"
                        href={state.release.html_url}
                        target="_blank"
                        rel="noreferrer"
                        variant="contained"
                        sx={{alignSelf: 'flex-start'}}
                        endIcon={<OpenInNew />}>
                        Open Release Page
                    </Button>
                </Stack>
            )}
        </SurfaceCard>
    );
};
