import axios from 'axios';
import {action} from 'mobx';
import {BaseStore} from '../common/BaseStore';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';
import {IUserGroup, IUserGroupMember} from '../types';

export class GroupStore extends BaseStore<IUserGroup> {
    public constructor(private readonly snack: SnackReporter) {
        super();
    }

    protected requestItems = (): Promise<IUserGroup[]> =>
        axios.get<IUserGroup[]>(`${config.get('url')}group`).then((response) => response.data);

    protected requestDelete = (id: number): Promise<void> =>
        axios.delete(`${config.get('url')}group/${id}`).then(() => {
            this.snack('Group deleted');
        });

    @action
    public create = async (name: string, description: string): Promise<void> => {
        await axios.post(`${config.get('url')}group`, {name, description});
        await this.refresh();
        this.snack('Group created');
    };

    @action
    public update = async (id: number, name: string, description: string): Promise<void> => {
        await axios.put(`${config.get('url')}group/${id}`, {name, description});
        await this.refresh();
        this.snack('Group updated');
    };

    public getMembers = (id: number): Promise<IUserGroupMember[]> =>
        axios
            .get<IUserGroupMember[]>(`${config.get('url')}group/${id}/members`)
            .then((response) => response.data);

    @action
    public addMember = async (groupId: number, userId: number): Promise<void> => {
        await axios.post(`${config.get('url')}group/${groupId}/members`, {userId});
        await this.refresh();
        this.snack('Group member added');
    };

    @action
    public removeMember = async (groupId: number, userId: number): Promise<void> => {
        await axios.delete(`${config.get('url')}group/${groupId}/members/${userId}`);
        await this.refresh();
        this.snack('Group member removed');
    };
}
