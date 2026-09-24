import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import React, {useState} from 'react';

interface IProps {
    channelName: string;
    fOnSubmit: (message: string) => Promise<void>;
}

const ChatComposer = ({channelName, fOnSubmit}: IProps) => {
    const [message, setMessage] = useState('');
    const [sending, setSending] = useState(false);

    const send = async () => {
        const trimmed = message.trim();
        if (!trimmed || sending) return;

        setSending(true);
        try {
            await fOnSubmit(trimmed);
            setMessage('');
        } finally {
            setSending(false);
        }
    };

    return (
        <Paper
            elevation={3}
            sx={{
                display: 'flex',
                gap: 1,
                alignItems: 'flex-end',
                padding: 1.5,
                marginBottom: 2,
            }}>
            <TextField
                autoFocus
                fullWidth
                multiline
                maxRows={5}
                label={`Message #${channelName}`}
                value={message}
                onChange={(event) => setMessage(event.target.value)}
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
