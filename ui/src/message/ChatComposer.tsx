import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import React, {useEffect, useRef, useState} from 'react';

interface IProps {
    channelName: string;
    fOnSubmit: (message: string) => Promise<void>;
    fOnTyping?: (typing: boolean) => Promise<void> | void;
}

const ChatComposer = ({channelName, fOnSubmit, fOnTyping}: IProps) => {
    const [message, setMessage] = useState('');
    const [sending, setSending] = useState(false);
    const stopTimer = useRef<number | null>(null);
    const lastTypingSentAt = useRef(0);

    const clearStopTimer = () => {
        if (stopTimer.current != null) {
            window.clearTimeout(stopTimer.current);
            stopTimer.current = null;
        }
    };

    const setTyping = (typing: boolean) => {
        if (!fOnTyping) return;
        void Promise.resolve(fOnTyping(typing)).catch(() => {});
        if (!typing) lastTypingSentAt.current = 0;
    };

    const noteInput = (value: string) => {
        setMessage(value);
        clearStopTimer();

        if (value.trim().length === 0) {
            setTyping(false);
            return;
        }

        const now = Date.now();
        if (now - lastTypingSentAt.current > 2200) {
            lastTypingSentAt.current = now;
            setTyping(true);
        }

        stopTimer.current = window.setTimeout(() => setTyping(false), 2600);
    };

    const send = async () => {
        const trimmed = message.trim();
        if (!trimmed || sending) return;

        clearStopTimer();
        setTyping(false);
        setSending(true);
        try {
            await fOnSubmit(trimmed);
            setMessage('');
        } finally {
            setSending(false);
        }
    };

    useEffect(
        () => () => {
            clearStopTimer();
            setTyping(false);
        },
        []
    );

    return (
        <Paper
            elevation={0}
            variant="outlined"
            sx={{
                display: 'flex',
                gap: 1,
                alignItems: 'flex-end',
                padding: 1,
                marginBottom: 1,
            }}>
            <TextField
                autoFocus
                fullWidth
                multiline
                maxRows={5}
                label={`Message #${channelName}`}
                value={message}
                onChange={(event) => noteInput(event.target.value)}
                onBlur={() => {
                    clearStopTimer();
                    setTyping(false);
                }}
                onKeyDown={(event) => {
                    if (event.key === 'Enter' && !event.shiftKey) {
                        event.preventDefault();
                        void send();
                    }
                }}
            />
            <Button
                variant="contained"
                disabled={sending || message.trim().length === 0}
                onClick={() => void send()}>
                Send
            </Button>
        </Paper>
    );
};

export default ChatComposer;
