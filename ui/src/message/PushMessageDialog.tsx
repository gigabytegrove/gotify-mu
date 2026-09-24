import React, {useState} from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    Stack,
    TextField,
    Tooltip,
} from '@mui/material';
import Send from '@mui/icons-material/Send';
import {NumberField} from '../common/NumberField';

interface IProps {
    appName: string;
    defaultPriority: number;
    fClose: VoidFunction;
    fOnSubmit: (message: string, title: string, priority: number) => Promise<void>;
}

export const PushMessageDialog = ({appName, defaultPriority, fClose, fOnSubmit}: IProps) => {
    const [title, setTitle] = useState('');
    const [message, setMessage] = useState('');
    const [priority, setPriority] = useState(defaultPriority);

    const submitEnabled = message.trim().length !== 0;

    const submitAndClose = async () => {
        await fOnSubmit(message, title, priority);
        fClose();
    };

    return (
        <Dialog id="push-message-dialog" open onClose={fClose} fullWidth maxWidth="sm">
            <DialogTitle>Send Notification</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Send a notification to {appName}. Leave the title blank to use the Channel
                    name.
                </DialogContentText>
                <Stack spacing={2}>
                    <TextField
                        className="title"
                        label="Title"
                        value={title}
                        onChange={(event) => setTitle(event.target.value)}
                        fullWidth
                    />
                    <TextField
                        autoFocus
                        className="message"
                        label="Message"
                        value={message}
                        onChange={(event) => setMessage(event.target.value)}
                        fullWidth
                        required
                        multiline
                        minRows={5}
                    />
                    <NumberField
                        className="priority"
                        label="Priority"
                        value={priority}
                        onChange={setPriority}
                        fullWidth
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={fClose}>Cancel</Button>
                <Tooltip title={submitEnabled ? '' : 'Message is required'}>
                    <span>
                        <Button
                            className="send"
                            disabled={!submitEnabled}
                            onClick={() => void submitAndClose()}
                            variant="contained"
                            startIcon={<Send />}>
                            Send
                        </Button>
                    </span>
                </Tooltip>
            </DialogActions>
        </Dialog>
    );
};
