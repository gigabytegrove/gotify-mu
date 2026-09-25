import axios from 'axios';
import {action, observable} from 'mobx';
import * as config from '../config';
import {IAuditEvent} from '../types';

export class AuditStore {
    @observable private accessor items: IAuditEvent[] = [];

    public getItems = (): IAuditEvent[] => this.items;

    @action
    public refresh = async (limit = 200, actionFilter = '', targetFilter = ''): Promise<void> => {
        const response = await axios.get<IAuditEvent[]>(`${config.get('url')}audit`, {
            params: {
                limit,
                action: actionFilter || undefined,
                target: targetFilter || undefined,
            },
        });
        this.items = response.data || [];
    };

    @action
    public clear = (): void => {
        this.items = [];
    };
}
