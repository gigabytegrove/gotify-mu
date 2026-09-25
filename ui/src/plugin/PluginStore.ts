import axios from 'axios';
import {action} from 'mobx';
import {BaseStore} from '../common/BaseStore';
import * as config from '../config';
import {SnackReporter} from '../snack/SnackManager';
import {IPlugin, IPluginCatalogEntry} from '../types';

export interface IPluginInstallResult {
    name: string;
    modulePath: string;
    warnings: string[];
}

export class PluginStore extends BaseStore<IPlugin> {
    public onDelete: () => void = () => {};

    public constructor(private readonly snack: SnackReporter) {
        super();
    }

    public requestConfig = (id: number): Promise<string> =>
        axios.get(`${config.get('url')}plugin/${id}/config`).then((response) => response.data);

    public requestDisplay = (id: number): Promise<string> =>
        axios.get(`${config.get('url')}plugin/${id}/display`).then((response) => response.data);

    protected requestItems = (): Promise<IPlugin[]> =>
        axios.get<IPlugin[]>(`${config.get('url')}plugin`).then((response) => response.data);

    protected requestDelete = (): Promise<void> => {
        this.snack('Cannot delete plugin');
        throw new Error('Cannot delete plugin');
    };

    public getName = (id: number): string => {
        const plugin = this.getByIDOrUndefined(id);
        return id === -1 ? 'All Plugins' : plugin !== undefined ? plugin.name : 'unknown';
    };

    @action
    public installPlugin = async (
        file: File,
        verification?: {sha256?: string; signature?: string; publicKey?: string}
    ): Promise<IPluginInstallResult> => {
        const form = new FormData();
        form.append('plugin', file);
        if (verification?.sha256) form.append('sha256', verification.sha256);
        if (verification?.signature) form.append('signature', verification.signature);
        if (verification?.publicKey) form.append('publicKey', verification.publicKey);

        const response = await axios.post<IPluginInstallResult>(
            `${config.get('url')}plugin/install`,
            form
        );

        this.snack(`Installed plugin: ${response.data.name}`);
        if (response.data.warnings.length > 0) {
            this.snack(
                `Plugin installed with ${response.data.warnings.length} initialization warning${response.data.warnings.length === 1 ? '' : 's'}`
            );
        }
        await this.refresh();
        return response.data;
    };

    public getCatalog = async (): Promise<IPluginCatalogEntry[]> =>
        axios
            .get<IPluginCatalogEntry[]>(`${config.get('url')}plugin/catalog`)
            .then((response) => response.data);

    @action
    public installCatalogPlugin = async (entry: IPluginCatalogEntry): Promise<void> => {
        const response = await axios.post<{restartRequired?: boolean}>(
            `${config.get('url')}plugin/catalog/install`,
            {modulePath: entry.modulePath, version: entry.version}
        );
        this.snack(
            response.data.restartRequired
                ? 'Plugin update staged. Restart Gotify MU to load it.'
                : 'Plugin installed from catalog'
        );
        await this.refresh();
    };

    @action
    public uninstallPlugin = async (id: number): Promise<void> => {
        await axios.delete(`${config.get('url')}plugin/${id}/uninstall`);
        this.snack('Plugin uninstalled');
        await this.refresh();
    };

    @action
    public stagePluginUpdate = async (
        id: number,
        file: File,
        verification?: {sha256?: string; signature?: string; publicKey?: string}
    ): Promise<void> => {
        const form = new FormData();
        form.append('plugin', file);
        if (verification?.sha256) form.append('sha256', verification.sha256);
        if (verification?.signature) form.append('signature', verification.signature);
        if (verification?.publicKey) form.append('publicKey', verification.publicKey);
        await axios.post(`${config.get('url')}plugin/${id}/update`, form);
        this.snack('Plugin update staged. Restart Gotify MU to load it.');
    };

    @action
    public changeConfig = async (id: number, newConfig: string): Promise<void> => {
        await axios.post(`${config.get('url')}plugin/${id}/config`, newConfig, {
            headers: {'content-type': 'application/x-yaml'},
        });
        this.snack(`Plugin config updated`);
        await this.refresh();
    };

    @action
    public changeEnabledState = async (id: number, enabled: boolean): Promise<void> => {
        await axios.post(`${config.get('url')}plugin/${id}/${enabled ? 'enable' : 'disable'}`);
        this.snack(`Plugin ${enabled ? 'enabled' : 'disabled'}`);
        await this.refresh();
    };
}
