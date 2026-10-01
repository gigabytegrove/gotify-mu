import Notify from 'notifyjs';
import removeMarkdown from 'remove-markdown';
import {IMessage} from '../types';

export function mayAllowPermission(): boolean {
    return Notify.needsPermission && Notify.isSupported() && Notification.permission !== 'denied';
}

export function requestPermission() {
    if (Notify.needsPermission && Notify.isSupported()) {
        Notify.requestPermission(
            () => console.log('granted notification permissions'),
            () => console.log('notification permission denied')
        );
    }
}

const mentionUserIds = (msg: IMessage): number[] => {
    const extras = msg.extras || {};
    const raw = extras['monita::mentionUserIds'] ?? extras['gotify::mu::mentionUserIds'] ?? [];
    if (!Array.isArray(raw)) return [];
    return raw.map((value) => Number(value)).filter((value) => Number.isFinite(value) && value > 0);
};

export const isMentionForUser = (msg: IMessage, userId: number): boolean =>
    Boolean(msg.collaboration?.mentioned) || mentionUserIds(msg).includes(userId);

export function notifyNewMessage(msg: IMessage, userId: number, channelName: string) {
    const mentioned = isMentionForUser(msg, userId);
    const sender = msg.senderName || msg.title || channelName || 'Monita';
    const title = mentioned ? `${sender} mentioned you` : msg.title;
    const plain = removeMarkdown(msg.message);
    const body = mentioned && channelName ? `${channelName} · ${plain}` : plain;

    const notify = new Notify(title, {
        body,
        icon: msg.image,
        silent: !mentioned,
        notifyClick: closeAndFocus(msg.appid),
        notifyShow: closeAfterTimeout,
    });
    notify.show();
}

function closeAndFocus(appId: number) {
    return (event: Event) => {
        if (window.parent) {
            window.parent.focus();
        }
        window.focus();
        if (appId > 0) {
            window.location.hash = `/channels/${appId}`;
        } else {
            window.location.hash = '/messages';
        }
        const target = event.target as Notification;
        target.close();
    };
}

function closeAfterTimeout(event: Event) {
    setTimeout(() => {
        const target = event.target as Notification;
        target.close();
    }, 7000);
}
