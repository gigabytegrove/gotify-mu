export interface UpdateActivity {
    timestamp: string;
    message: string;
}

export interface UpdaterStatus {
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

export const activeUpdaterStates = new Set([
    'preparing',
    'downloading',
    'verifying',
    'installing',
    'restarting',
]);

export const updaterStatusForDisplay = (
    status: UpdaterStatus,
    updateStartedHere: boolean,
    sawActiveUpdate: boolean
): UpdaterStatus => {
    if (status.state !== 'completed' || updateStartedHere || sawActiveUpdate) {
        return status;
    }

    return {
        ready: status.ready,
        state: 'idle',
        version: status.version,
        progress: 0,
    };
};
