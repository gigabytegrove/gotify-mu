import React, {useEffect, useState} from 'react';
import axios from 'axios';
import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    IconButton,
    MenuItem,
    Stack,
    TextField,
    Tooltip,
} from '@mui/material';
import Add from '@mui/icons-material/Add';
import Delete from '@mui/icons-material/Delete';
import Send from '@mui/icons-material/Send';
import {PriorityField} from '../common/NotificationFields';
import * as config from '../config';
import {
    IMessageExtras,
    IMessageTemplate,
    INotificationAction,
    INotificationField,
} from '../types';
import {useStores} from '../stores';

interface IProps {
    appId: number;
    appName: string;
    defaultPriority: number;
    fClose: VoidFunction;
    fOnSubmit: (
        message: string,
        title: string,
        priority: number,
        extras?: IMessageExtras
    ) => Promise<void>;
}

export const PushMessageDialog = ({appId, appName, defaultPriority, fClose, fOnSubmit}: IProps) => {
    const [title, setTitle] = useState('');
    const [message, setMessage] = useState('');
    const [priority, setPriority] = useState(defaultPriority);
    const [templates, setTemplates] = useState<IMessageTemplate[]>([]);
    const [templateId, setTemplateId] = useState(0);
    const [templateName, setTemplateName] = useState('');
    const [actions, setActions] = useState<INotificationAction[]>([]);
    const [fields, setFields] = useState<INotificationField[]>([]);
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
        const display = item.extras?.['gotify-mu::display'];
        setActions(Array.isArray(display?.actions) ? display.actions : []);
        setFields(Array.isArray(display?.fields) ? display.fields : []);
    };

    const saveTemplate = async () => {
        const name = templateName.trim();
        if (!name || !message.trim()) return;
        const extras =
            actions.length > 0 || fields.length > 0
                ? {
                      'gotify-mu::display': {
                          actions: actions.filter((item) => item.label.trim() && item.url.trim()),
                          fields: fields.filter((item) => item.label.trim() && item.value.trim()),
                      },
                  }
                : undefined;
        await axios.post(config.get('url') + 'message-template', {
            name,
            applicationId: appId,
            title,
            message,
            priority,
            extras,
        });
        setTemplateName('');
        await loadTemplates();
        snackManager.snack('Template saved');
    };

    const submitEnabled = message.trim().length !== 0;

    const submitAndClose = async () => {
        const cleanActions = actions.filter((item) => item.label.trim() && item.url.trim());
        const cleanFields = fields.filter((item) => item.label.trim() && item.value.trim());
        const extras: IMessageExtras | undefined =
            cleanActions.length > 0 || cleanFields.length > 0
                ? {
                      'gotify-mu::display': {
                          actions: cleanActions,
                          fields: cleanFields,
                      },
                  }
                : undefined;
        await fOnSubmit(message, title, priority, extras);
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
                    <PriorityField
                        className="priority"
                        value={priority}
                        onChange={setPriority}
                        fullWidth
                    />
                    <Box sx={{borderTop: 1, borderColor: 'divider', pt: 2}}>
                        <Stack spacing={1.5}>
                            <Stack
                                direction="row"
                                spacing={1}
                                sx={{justifyContent: 'space-between', alignItems: 'center'}}>
                                <Box>
                                    <strong>Details</strong>
                                    <Box sx={{fontSize: '0.8rem', color: 'text.secondary'}}>
                                        Add structured values that appear below the notification.
                                    </Box>
                                </Box>
                                <Button
                                    size="small"
                                    startIcon={<Add />}
                                    disabled={fields.length >= 20}
                                    onClick={() =>
                                        setFields([...fields, {label: '', value: ''}])
                                    }>
                                    Add Detail
                                </Button>
                            </Stack>
                            {fields.map((field, index) => (
                                <Stack key={index} direction="row" spacing={1}>
                                    <TextField
                                        size="small"
                                        label="Label"
                                        value={field.label}
                                        onChange={(event) => {
                                            const next = [...fields];
                                            next[index] = {...field, label: event.target.value};
                                            setFields(next);
                                        }}
                                        fullWidth
                                    />
                                    <TextField
                                        size="small"
                                        label="Value"
                                        value={field.value}
                                        onChange={(event) => {
                                            const next = [...fields];
                                            next[index] = {...field, value: event.target.value};
                                            setFields(next);
                                        }}
                                        fullWidth
                                    />
                                    <IconButton
                                        aria-label="Remove detail"
                                        onClick={() =>
                                            setFields(fields.filter((_item, i) => i !== index))
                                        }>
                                        <Delete />
                                    </IconButton>
                                </Stack>
                            ))}
                        </Stack>
                    </Box>

                    <Box sx={{borderTop: 1, borderColor: 'divider', pt: 2}}>
                        <Stack spacing={1.5}>
                            <Stack
                                direction="row"
                                spacing={1}
                                sx={{justifyContent: 'space-between', alignItems: 'center'}}>
                                <Box>
                                    <strong>Action Buttons</strong>
                                    <Box sx={{fontSize: '0.8rem', color: 'text.secondary'}}>
                                        Add links users can open directly from the Web UI.
                                    </Box>
                                </Box>
                                <Button
                                    size="small"
                                    startIcon={<Add />}
                                    disabled={actions.length >= 8}
                                    onClick={() =>
                                        setActions([...actions, {label: '', url: 'https://'}])
                                    }>
                                    Add Action
                                </Button>
                            </Stack>
                            {actions.map((action, index) => (
                                <Stack key={index} direction="row" spacing={1}>
                                    <TextField
                                        size="small"
                                        label="Button label"
                                        value={action.label}
                                        onChange={(event) => {
                                            const next = [...actions];
                                            next[index] = {...action, label: event.target.value};
                                            setActions(next);
                                        }}
                                        fullWidth
                                    />
                                    <TextField
                                        size="small"
                                        label="URL"
                                        value={action.url}
                                        onChange={(event) => {
                                            const next = [...actions];
                                            next[index] = {...action, url: event.target.value};
                                            setActions(next);
                                        }}
                                        fullWidth
                                    />
                                    <IconButton
                                        aria-label="Remove action"
                                        onClick={() =>
                                            setActions(actions.filter((_item, i) => i !== index))
                                        }>
                                        <Delete />
                                    </IconButton>
                                </Stack>
                            ))}
                        </Stack>
                    </Box>

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
