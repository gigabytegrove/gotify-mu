import {SnackReporter} from '../snack/SnackManager';
import {CurrentUser} from '../CurrentUser';
import * as config from '../config';
import axios, {AxiosError} from 'axios';
import {IMessage, IMURealtimeEvent} from '../types';

export class WebSocketStore {
    private wsActive = false;
    private ws: WebSocket | null = null;

    private muWsActive = false;
    private muWs: WebSocket | null = null;
    private muReconnectTimer: number | null = null;
    private readonly muListeners = new Set<(event: IMURealtimeEvent) => void>();

    public constructor(
        private readonly snack: SnackReporter,
        private readonly currentUser: CurrentUser
    ) {}

    public listen = (callback: (msg: IMessage) => void) => {
        if (!this.currentUser.loggedIn || this.wsActive) {
            return;
        }
        this.wsActive = true;

        const wsUrl = config.get('url').replace('http', 'ws').replace('https', 'wss');
        const ws = new WebSocket(wsUrl + 'stream');

        ws.onerror = (e) => {
            this.wsActive = false;
            console.log('WebSocket connection errored', e);
        };

        ws.onmessage = (data) => callback(JSON.parse(data.data));

        ws.onclose = () => {
            this.wsActive = false;
            if (!this.currentUser.loggedIn) {
                return;
            }
            this.currentUser
                .tryAuthenticate()
                .then(() => {
                    this.snack('WebSocket connection closed, trying again in 30 seconds.');
                    setTimeout(() => this.listen(callback), 30000);
                })
                .catch((error: AxiosError) => {
                    if (error?.response?.status === 401) {
                        this.snack('Could not authenticate with client token, logging out.');
                    }
                });
        };

        this.ws = ws;
    };

    public listenMU = (callback: (event: IMURealtimeEvent) => void): (() => void) => {
        this.muListeners.add(callback);
        this.ensureMUConnection();

        return () => {
            this.muListeners.delete(callback);
            if (this.muListeners.size === 0) {
                if (this.muReconnectTimer != null) {
                    window.clearTimeout(this.muReconnectTimer);
                    this.muReconnectTimer = null;
                }
                this.muWs?.close(1000, 'No MU realtime listeners');
                this.muWs = null;
                this.muWsActive = false;
            }
        };
    };

    public setTyping = async (appId: number, typing: boolean): Promise<void> => {
        await axios.post(`${config.get('url')}application/${appId}/typing`, {typing});
    };

    private ensureMUConnection = () => {
        if (!this.currentUser.loggedIn || this.muWsActive || this.muListeners.size === 0) {
            return;
        }

        this.muWsActive = true;
        const wsUrl = config.get('url').replace('http', 'ws').replace('https', 'wss');
        const ws = new WebSocket(wsUrl + 'api/mu/v1/events');

        ws.onopen = () => {
            this.muWsActive = true;
        };

        ws.onerror = (error) => {
            console.log('MU realtime WebSocket connection errored', error);
        };

        ws.onmessage = (data) => {
            try {
                const event = JSON.parse(data.data) as IMURealtimeEvent;
                this.muListeners.forEach((listener) => listener(event));
            } catch (error) {
                console.warn('Invalid MU realtime event', error);
            }
        };

        ws.onclose = () => {
            this.muWs = null;
            this.muWsActive = false;
            if (!this.currentUser.loggedIn || this.muListeners.size === 0) {
                return;
            }
            if (this.muReconnectTimer != null) {
                window.clearTimeout(this.muReconnectTimer);
            }
            this.muReconnectTimer = window.setTimeout(() => {
                this.muReconnectTimer = null;
                this.ensureMUConnection();
            }, 5000);
        };

        this.muWs = ws;
    };

    public close = () => {
        this.ws?.close(1000, 'WebSocketStore#close');
        this.muWs?.close(1000, 'WebSocketStore#close');
        if (this.muReconnectTimer != null) {
            window.clearTimeout(this.muReconnectTimer);
            this.muReconnectTimer = null;
        }
        this.muWs = null;
        this.muWsActive = false;
    };
}
