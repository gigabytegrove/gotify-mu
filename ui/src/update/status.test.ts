import {describe, expect, it} from 'vitest';
import {updaterStatusForDisplay, type UpdaterStatus} from './status';

const completedStatus = (): UpdaterStatus => ({
    ready: true,
    state: 'completed',
    version: '1.1.3',
    message: 'Update installed successfully',
    step: 'Update complete',
    progress: 100,
    startedAt: '2026-09-27T23:12:53Z',
    finishedAt: '2026-09-27T23:12:56Z',
    activity: [
        {
            timestamp: '2026-09-27T23:12:56Z',
            message: 'Update complete',
        },
    ],
});

describe('software update status presentation', () => {
    it('hides a completed run when the update page is revisited', () => {
        expect(updaterStatusForDisplay(completedStatus(), false, false)).toEqual({
            ready: true,
            state: 'idle',
            version: '1.1.3',
            progress: 0,
        });
    });

    it('keeps completion visible for an update started on the current page', () => {
        const status = completedStatus();
        expect(updaterStatusForDisplay(status, true, true)).toBe(status);
    });

    it('keeps completion visible when this page observed the update running', () => {
        const status = completedStatus();
        expect(updaterStatusForDisplay(status, false, true)).toBe(status);
    });

    it('does not hide failures', () => {
        const status: UpdaterStatus = {
            ready: true,
            state: 'failed',
            message: 'The update could not be installed.',
            progress: 84,
        };
        expect(updaterStatusForDisplay(status, false, false)).toBe(status);
    });
});
