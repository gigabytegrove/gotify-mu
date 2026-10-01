import Button from '@mui/material/Button';
import React from 'react';
import {closeSnackbar, enqueueSnackbar, SnackbarKey} from 'notistack';

export interface SnackReporter {
    (message: string): void;
}

export interface IncomingMessageToast {
    appId: number;
    channelName: string;
    channelType: 'chat' | 'notification';
    senderName?: string;
    message: string;
    mentioned: boolean;
}

const preview = (message: string): string => {
    const value = message.trim().replace(/\s+/g, ' ');
    if (!value) return 'New image or attachment';
    return value.length > 140 ? value.slice(0, 137) + '…' : value;
};

const openChannel = (appId: number, key: SnackbarKey) => {
    window.location.hash = appId > 0 ? `/channels/${appId}` : '/messages';
    if (window.parent) window.parent.focus();
    window.focus();
    closeSnackbar(key);
};

export class SnackManager {
    public snack: SnackReporter = (message: string): void => {
        enqueueSnackbar({message, variant: 'info'});
    };

    public incomingMessage = ({
        appId,
        channelName,
        channelType,
        senderName,
        message,
        mentioned,
    }: IncomingMessageToast): void => {
        const location =
            channelType === 'chat' ? `Chat · ${channelName}` : `Channel · ${channelName}`;
        const sender = senderName?.trim();
        const heading = mentioned
            ? `${sender || 'Someone'} mentioned you in ${channelName}`
            : sender
              ? `${sender} · ${location}`
              : location;

        enqueueSnackbar({
            message: `${heading} — ${preview(message)}`,
            variant: mentioned ? 'warning' : 'info',
            autoHideDuration: mentioned ? 10000 : 7000,
            anchorOrigin: {vertical: 'top', horizontal: 'right'},
            preventDuplicate: false,
            action: (key) =>
                React.createElement(
                    Button,
                    {
                        size: 'small',
                        color: 'inherit',
                        onClick: () => openChannel(appId, key),
                        sx: {fontWeight: 800},
                    },
                    'Open'
                ),
        });
    };
}
