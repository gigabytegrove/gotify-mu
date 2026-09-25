import React, {useEffect, useState} from 'react';
import axios from 'axios';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    MenuItem,
    Stack,
    TextField,
    Tooltip,
} from '@mui/material';
import Send from '@mui/icons-material/Send';
import {NumberField} from '../common/NumberField';
import * as config from '../config';
import {IMessageTemplate} from '../types';
import {useStores} from '../stores';

interface IProps {
    appId: number;
    appName: string;
    defaultPriority: number;
    fClose: VoidFunction;
    fOnSubmit: (message: string, title: string, priority: number) => Promise<void>;
}

export const PushMessageDialog = ({appId, appName, defaultPriority, fClose, fOnSubmit}: IProps) => {
    const [title, setTitle] = useState('');
    const [message, setMessage] = useState('');
    const [priority, setPriority] = useState(defaultPriority);
    const [templates, setTemplates] = useState<IMessageTemplate[]>([]);
    const [templateId, setTemplateId] = useState(0);
    const [templateName, setTemplateName] = useState('');
    const {snackManager} = useStores();

    const loadTemplates = async () => {
        const response = await axios.get<IMessageTemplate[]>(config.get('url') + 'message-template');
        setTemplates(response.data);
    };

    useEffect(() => {
        void loadTemplates();
    }, []);

    const applyTemplate = (id: number) => {
        setTemplateId(id);
        const item = templates.find((template) => template.id === id);
        if (!item) return;
        setTitle(item.title);
        setMessage(item.message);
        setPriority(item.priority);
    };

    const saveTemplate = async () => {
        const name = templateName.trim();
        if (!name || !message.trim()) return;
        await axios.post(config.get('url') + 'message-template', {
            name,
            applicationId: appId,
            title,
            message,
            priority,
        });
        setTemplateName('');
        await loadTemplates();
        snackManager.snack('Template saved');
    };

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
                        select
                        label="Template"
                        value={templateId}
                        onChange={(event) => applyTemplate(Number(event.target.value))}>
                        <MenuItem value={0}>No template</MenuItem>
                        {templates
                            .filter(
                                (template) =>
                                    !template.applicationId || template.applicationId === appId
                            )
                            .map((template) => (
                                <MenuItem key={template.id} value={template.id}>
                                    {template.name}
                                </MenuItem>
                            ))}
                    </TextField>
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
                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={1}>
                        <TextField
                            size="small"
                            label="Save as template"
                            value={templateName}
                            onChange={(event) => setTemplateName(event.target.value)}
                            fullWidth
                        />
                        <Button
                            variant="outlined"
                            disabled={!templateName.trim() || !message.trim()}
                            onClick={() => void saveTemplate()}>
                            Save Template
                        </Button>
                    </Stack>
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
