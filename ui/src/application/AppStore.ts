import axios from 'axios';
import {generateKeyBetween} from 'fractional-indexing';
import {action, runInAction} from 'mobx';
import {BaseStore} from '../common/BaseStore';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';
import {IApplication, IApplicationMember, IUser} from '../types';
import {arrayMove} from '@dnd-kit/sortable';

export class AppStore extends BaseStore<IApplication> {
    public onDelete: () => void = () => {};

    public constructor(private readonly snack: SnackReporter) {
        super();
    }

    protected requestItems = (): Promise<IApplication[]> =>
        axios
            .get<IApplication[]>(`${config.get('url')}application`)
            .then((response) => response.data);

    protected requestDelete = (id: number): Promise<void> =>
        axios.delete(`${config.get('url')}application/${id}`).then(() => {
            this.onDelete();
            return this.snack('Channel deleted');
        });

    @action
    public uploadImage = async (id: number, file: Blob): Promise<void> => {
        const formData = new FormData();
        formData.append('file', file);
        await axios.post(`${config.get('url')}application/${id}/image`, formData, {
            headers: {'content-type': 'multipart/form-data'},
        });
        await this.refresh();
        this.snack('Channel image updated');
    };

    public async regenerateToken(id: number): Promise<string> {
        const response = await axios.put(`${config.get('url')}application/${id}/security`, {
            regenerateToken: true,
        });
        if (!response.data?.regenerateToken?.token) {
            throw new Error('unexpected response from server');
        }
        return response.data.regenerateToken.token;
    }

    public async deleteImage(id: number): Promise<void> {
        try {
            await axios.delete(`${config.get('url')}application/${id}/image`);
            await this.refresh();
            this.snack('Channel image deleted');
        } catch (error) {
            console.error('Error deleting application image:', error);
            throw error;
        }
    }

    @action
    public reorder = async (fromId: number, toId: number): Promise<void> => {
        const fromIndex = this.items.findIndex((app) => app.id === fromId);
        const toIndex = this.items.findIndex((app) => app.id === toId);
        if (fromIndex === -1 || toIndex === -1) {
            throw Error('unknown apps');
        }

        const toUpdate = this.items[fromIndex];

        const normalizedIndex =
            toUpdate.sortKey > this.items[toIndex].sortKey ? toIndex - 1 : toIndex;

        const newSortKey = generateKeyBetween(
            this.items[normalizedIndex]?.sortKey,
            this.items[normalizedIndex + 1]?.sortKey
        );

        runInAction(() => (this.items = arrayMove(this.items, fromIndex, toIndex)));

        await this.update({...toUpdate, sortKey: newSortKey});
    };

    @action
    public update = async ({
        id,
        ...app
    }: Pick<
        IApplication,
        'id' | 'name' | 'description' | 'defaultPriority' | 'sortKey' | 'retentionDays'
    >): Promise<void> => {
        await axios.put(`${config.get('url')}application/${id}`, app);
        await this.refresh();
        this.snack('Channel updated');
    };

    @action
    public create = async (
        name: string,
        description: string,
        defaultPriority: number,
        autoAssign = false,
        allowMemberPost = false
    ): Promise<string> => {
        const response = await axios.post(`${config.get('url')}application`, {
            name,
            description,
            defaultPriority,
            autoAssign,
            allowMemberPost,
        });
        await this.refresh();
        this.snack('Channel created');
        return response.data.token;
    };

    public getMembers = async (id: number): Promise<IApplicationMember[]> =>
        axios
            .get<IApplicationMember[]>(`${config.get('url')}application/${id}/members`)
            .then((response) => response.data);

    public getAssignableUsers = async (id: number): Promise<IUser[]> =>
        axios
            .get<IUser[]>(`${config.get('url')}application/${id}/assignable-users`)
            .then((response) => response.data);

    public setMember = async (
        id: number,
        userId: number,
        receiveNotifications = true
    ): Promise<IApplicationMember> =>
        axios
            .post<IApplicationMember>(`${config.get('url')}application/${id}/members`, {
                userId,
                receiveNotifications,
            })
            .then((response) => response.data);

    public removeMember = async (id: number, userId: number): Promise<void> => {
        await axios.delete(`${config.get('url')}application/${id}/members/${userId}`);
    };

    public setAutoAssign = async (id: number, enabled: boolean): Promise<void> => {
        await axios.put(`${config.get('url')}application/${id}/auto-assign`, {enabled});
        await this.refresh();
        this.snack(
            enabled ? 'Channel auto-assignment enabled' : 'Channel auto-assignment disabled'
        );
    };

    public setMemberPosting = async (id: number, enabled: boolean): Promise<void> => {
        await axios.put(`${config.get('url')}application/${id}/member-posting`, {enabled});
        await this.refresh();
        this.snack(enabled ? 'Chat posting enabled' : 'Chat posting disabled');
    };

    public setNotifications = async (id: number, enabled: boolean): Promise<void> => {
        await axios.put(`${config.get('url')}application/${id}/notifications`, {enabled});
        await this.refresh();
        this.snack(enabled ? 'Channel notifications enabled' : 'Channel notifications muted');
    };

    public transferOwnership = async (id: number, userId: number): Promise<void> => {
        await axios.put(`${config.get('url')}application/${id}/owner`, {userId});
        await this.refresh();
        this.snack('Channel ownership transferred');
    };

    public clearHistoryForEveryone = async (id: number): Promise<void> => {
        await axios.delete(`${config.get('url')}application/${id}/message/all`);
        this.snack('Channel history cleared for everyone');
    };

    public getName = (id: number): string => {
        const app = this.getByIDOrUndefined(id);
        return id === -1 ? 'All Messages' : app !== undefined ? app.name : 'unknown';
    };
}
