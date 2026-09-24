import {BaseStore} from '../common/BaseStore';
import {action, IObservableArray, observable, reaction, runInAction} from 'mobx';
import axios, {AxiosResponse} from 'axios';
import * as config from '../config';
import {createTransformer} from 'mobx-utils';
import {SnackReporter} from '../snack/SnackManager';
import {IApplication, IMessage, IPagedMessages} from '../types';
import {closeSnackbar, SnackbarKey} from 'notistack';

const AllMessages = -1;

interface MessagesState {
    messages: IObservableArray<IMessage>;
    hasMore: boolean;
    nextSince: number;
    loaded: boolean;
}

interface PendingDelete {
    key: SnackbarKey;
    message: IMessage;
}

export class MessagesStore {
    @observable private accessor state: Record<string, MessagesState> = {};
    @observable private accessor archivedState: Record<string, MessagesState> = {};
    @observable private accessor pendingDeletes: Map<number, PendingDelete> = observable.map();

    private loading = false;

    public constructor(
        private readonly appStore: BaseStore<IApplication>,
        private readonly snack: SnackReporter
    ) {
        reaction(() => appStore.getItems(), this.createEmptyStatesForApps);
    }

    private stateOf = (appId: number, archived = false, create = true) => {
        const states = archived ? this.archivedState : this.state;
        if (!states[appId] && create) {
            states[appId] = this.emptyState();
        }
        return states[appId] || this.emptyState();
    };

    public loaded = (appId: number, archived = false) =>
        this.stateOf(appId, archived, /*create*/ false).loaded;

    public canLoadMore = (appId: number, archived = false) =>
        this.stateOf(appId, archived, /*create*/ false).hasMore;

    @action
    public loadMore = async (appId: number, archived = false) => {
        const state = this.stateOf(appId, archived);
        if (!state.hasMore || this.loading) {
            return Promise.resolve();
        }
        this.loading = true;

        try {
            const pagedResult = await this.fetchMessages(appId, state.nextSince, archived).then(
                (resp) => resp.data
            );
            runInAction(() => {
                state.messages.replace([...state.messages, ...pagedResult.messages]);
                state.nextSince = pagedResult.paging.since ?? 0;
                state.hasMore = 'next' in pagedResult.paging;
                state.loaded = true;
            });
        } finally {
            this.loading = false;
        }

        return Promise.resolve();
    };

    @action
    public publishSingleMessage = (message: IMessage) => {
        if (this.exists(AllMessages)) {
            this.stateOf(AllMessages, false).messages.unshift(message);
        }
        if (this.exists(message.appid)) {
            this.stateOf(message.appid, false).messages.unshift(message);
        }
    };

    @action
    public archiveByApp = async (appId: number) => {
        if (appId === AllMessages) {
            await axios.post(config.get('url') + 'message/archive');
            this.snack('Archived all messages');
        } else {
            await axios.post(config.get('url') + 'application/' + appId + '/message/archive');
            this.snack(`Archived all messages from ${this.appStore.getByID(appId).name}`);
        }
        this.clearAll();
        await this.loadMore(appId, false);
    };

    @action
    public restoreByApp = async (appId: number) => {
        if (appId === AllMessages) {
            await axios.delete(config.get('url') + 'message/archive');
            this.snack('Restored all archived messages');
        } else {
            await axios.delete(config.get('url') + 'application/' + appId + '/message/archive');
            this.snack(`Restored archived messages from ${this.appStore.getByID(appId).name}`);
        }
        this.clearAll();
        await this.loadMore(appId, true);
    };

    @action
    public removeByApp = async (appId: number) => {
        if (appId === AllMessages) {
            await axios.delete(config.get('url') + 'message');
            this.snack('Deleted all messages');
            this.clearAll();
        } else {
            await axios.delete(config.get('url') + 'application/' + appId + '/message');
            this.snack(`Deleted all messages from ${this.appStore.getByID(appId).name}`);
            this.clear(AllMessages);
            this.clear(appId);
        }
        await this.loadMore(appId, false);
    };

    @action
    public archiveSingle = async (message: IMessage) => {
        await axios.post(config.get('url') + 'message/' + message.id + '/archive');
        if (this.exists(AllMessages, false)) {
            this.removeFromList(this.state[AllMessages].messages, message);
        }
        if (this.exists(message.appid, false)) {
            this.removeFromList(this.state[message.appid].messages, message);
        }
        this.clear(AllMessages, true);
        this.clear(message.appid, true);
        this.snack('Message archived');
    };

    @action
    public restoreSingle = async (message: IMessage) => {
        await axios.delete(config.get('url') + 'message/' + message.id + '/archive');
        if (this.exists(AllMessages, true)) {
            this.removeFromList(this.archivedState[AllMessages].messages, message);
        }
        if (this.exists(message.appid, true)) {
            this.removeFromList(this.archivedState[message.appid].messages, message);
        }
        this.clear(AllMessages, false);
        this.clear(message.appid, false);
        this.snack('Message restored');
    };

    @action
    public addPendingDelete = (pending: PendingDelete) =>
        this.pendingDeletes.set(pending.message.id, pending);

    @action
    public cancelPendingDelete = (message: IMessage): boolean => {
        const pending = this.pendingDeletes.get(message.id);
        if (pending) {
            this.pendingDeletes.delete(message.id);
            closeSnackbar(pending.key);
        }
        return !!pending;
    };

    @action
    public executePendingDeletes = () =>
        Array.from(this.pendingDeletes.values()).forEach(({message}) => this.removeSingle(message));

    public visible = (message: number): boolean => !this.pendingDeletes.has(message);

    @action
    public removeSingle = async (message: IMessage) => {
        if (!this.pendingDeletes.has(message.id)) {
            return;
        }

        await axios.delete(config.get('url') + 'message/' + message.id, {
            adapter: 'fetch',
            fetchOptions: {keepalive: true},
        });
        if (this.exists(AllMessages)) {
            this.removeFromList(this.state[AllMessages].messages, message);
        }
        if (this.exists(message.appid)) {
            this.removeFromList(this.state[message.appid].messages, message);
        }
        this.cancelPendingDelete(message);
    };

    public sendMessage = async (
        appId: number,
        message: string,
        title: string,
        priority: number
    ): Promise<void> => {
        const app = this.appStore.getByID(appId);
        const payload: Pick<IMessage, 'appid' | 'title' | 'message' | 'priority'> = {
            appid: appId,
            message,
            priority,
            title,
        };

        await axios.post(`${config.get('url')}message`, payload);
        await this.refreshByApp(appId, false);
        this.snack(`Message sent to ${app.name}`);
    };

    @action
    public clearAll = () => {
        this.state = {};
        this.archivedState = {};
        this.createEmptyStatesForApps(this.appStore.getItems());
    };

    @action
    public refreshByApp = async (appId: number, archived = false) => {
        this.clear(appId, archived);
        await this.loadMore(appId, archived);
    };

    public exists = (id: number, archived = false) =>
        this.stateOf(id, archived, /*create*/ false).loaded;

    @action
    private removeFromList(messages: IMessage[], messageToDelete: IMessage): false | number {
        if (messages) {
            const index = messages.findIndex((message) => message.id === messageToDelete.id);
            if (index !== -1) {
                messages.splice(index, 1);
                return index;
            }
        }
        return false;
    }

    @action
    private clear = (appId: number, archived = false) => {
        if (archived) {
            this.archivedState[appId] = this.emptyState();
        } else {
            this.state[appId] = this.emptyState();
        }
    };

    private fetchMessages = (
        appId: number,
        since: number,
        archived = false
    ): Promise<AxiosResponse<IPagedMessages>> => {
        const archivedQuery = archived ? '&archived=true' : '';
        if (appId === AllMessages) {
            return axios.get(config.get('url') + 'message?since=' + since + archivedQuery);
        } else {
            return axios.get(
                config.get('url') +
                    'application/' +
                    appId +
                    '/message?since=' +
                    since +
                    archivedQuery
            );
        }
    };

    private getUnCached = (appId: number): Array<IMessage> => {
        const appToImage: Partial<Record<string, string>> = this.appStore
            .getItems()
            .reduce((all, app) => ({...all, [app.id]: app.image}), {});

        return this.stateOf(appId, false, false)
            .messages.filter((message) => !this.pendingDeletes.has(message.id))
            .map((message: IMessage): IMessage => ({...message, image: appToImage[message.appid]}));
    };

    private getArchivedUnCached = (appId: number): Array<IMessage> => {
        const appToImage: Partial<Record<string, string>> = this.appStore
            .getItems()
            .reduce((all, app) => ({...all, [app.id]: app.image}), {});

        return this.stateOf(appId, true, false).messages.map(
            (message: IMessage): IMessage => ({...message, image: appToImage[message.appid]})
        );
    };

    public get = createTransformer(this.getUnCached);
    public getArchived = createTransformer(this.getArchivedUnCached);

    private clearCache = () => {
        this.get = createTransformer(this.getUnCached);
        this.getArchived = createTransformer(this.getArchivedUnCached);
    };

    private createEmptyStatesForApps = (apps: IApplication[]) => {
        apps.map((app) => app.id).forEach((id) => {
            this.stateOf(id, false, /*create*/ true);
            this.stateOf(id, true, /*create*/ true);
        });
        this.stateOf(AllMessages, false, /*create*/ true);
        this.stateOf(AllMessages, true, /*create*/ true);
        this.clearCache();
    };

    private emptyState = (): MessagesState => ({
        messages: observable.array(),
        hasMore: true,
        nextSince: 0,
        loaded: false,
    });
}
