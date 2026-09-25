import {IMessageExtras, INotificationAction, INotificationField} from '../types';

export enum RenderMode {
    Markdown = 'text/markdown',
    Plain = 'text/plain',
}

export const contentType = (extras?: IMessageExtras): RenderMode => {
    const type = extract(extras, 'client::display', 'contentType');
    const valid = Object.values(RenderMode).includes(type);
    return valid ? type : RenderMode.Plain;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const extract = (extras: IMessageExtras | undefined, key: string, path: string): any => {
    if (!extras) {
        return null;
    }

    if (!extras[key]) {
        return null;
    }

    if (!extras[key][path]) {
        return null;
    }

    return extras[key][path];
};


export const notificationActions = (extras?: IMessageExtras): INotificationAction[] => {
    const value = extras?.['gotify-mu::display']?.actions;
    if (!Array.isArray(value)) return [];
    return value
        .filter(
            (item): item is INotificationAction =>
                Boolean(
                    item &&
                        typeof item === 'object' &&
                        typeof item.label === 'string' &&
                        typeof item.url === 'string'
                )
        )
        .slice(0, 8);
};

export const notificationFields = (extras?: IMessageExtras): INotificationField[] => {
    const value = extras?.['gotify-mu::display']?.fields;
    if (!Array.isArray(value)) return [];
    return value
        .filter(
            (item): item is INotificationField =>
                Boolean(
                    item &&
                        typeof item === 'object' &&
                        typeof item.label === 'string' &&
                        typeof item.value === 'string'
                )
        )
        .slice(0, 20);
};
