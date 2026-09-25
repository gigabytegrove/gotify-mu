import axios from 'axios';
import {CurrentUser} from './CurrentUser';
import {ElevateStore} from './ElevateStore';
import {SnackReporter} from './snack/SnackManager';

interface ErrorPayload {
    error?: string;
    errorDescription?: string;
}

export const initAxios = (
    currentUser: CurrentUser,
    elevateStore: ElevateStore,
    snack: SnackReporter
) => {
    axios.interceptors.response.use(undefined, (error) => {
        if (!error.response) {
            snack('Gotify server is not reachable, try refreshing the page.');
            return Promise.reject(error);
        }

        const status = error.response.status;
        const payload = (error.response.data || {}) as ErrorPayload;
        const description = payload.errorDescription || '';
        const elevationRequired =
            status === 403 && description.toLowerCase().includes('session not elevated');

        if (elevationRequired) {
            elevateStore.requestReauthentication();

            // The server is authoritative. Refresh the current session so the local elevation
            // timer/state also reflects the expired server-side session.
            void currentUser.tryAuthenticate().catch(() => {});
            return Promise.reject(error);
        }

        if (status === 401) {
            currentUser.tryAuthenticate().then(() => snack('Could not complete request.'));
        }

        if (status === 400 || status === 403 || status === 500) {
            snack(
                payload.error && description
                    ? payload.error + ': ' + description
                    : payload.error || description || 'The request could not be completed.'
            );
        }

        return Promise.reject(error);
    });
};
