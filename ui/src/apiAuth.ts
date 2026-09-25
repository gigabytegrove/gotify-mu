import axios from 'axios';
import {CurrentUser} from './CurrentUser';
import {SnackReporter} from './snack/SnackManager';

interface ErrorPayload {
    error?: string;
    errorDescription?: string;
}

export const initAxios = (currentUser: CurrentUser, snack: SnackReporter) => {
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
            // The server is authoritative. Refresh the current session so MobX's elevation
            // reaction immediately switches protected pages/dialogs back to the re-auth form.
            void currentUser.tryAuthenticate().catch(() => {});
            snack('Re-authentication required. Confirm your identity and try again.');
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
