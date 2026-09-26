import React from 'react';
import axios from 'axios';
import {
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Stack,
    TextField,
    Typography,
} from '@mui/material';
import Reply from '@mui/icons-material/Reply';
import Forum from '@mui/icons-material/Forum';
import AssignmentInd from '@mui/icons-material/AssignmentInd';
import TaskAlt from '@mui/icons-material/TaskAlt';
import MarkEmailRead from '@mui/icons-material/MarkEmailRead';
import MarkEmailUnread from '@mui/icons-material/MarkEmailUnread';
import AttachFile from '@mui/icons-material/AttachFile';
import * as config from '../config';
import {IMessage} from '../types';
import {useStores} from '../stores';

const api = (path: string) => config.get('url') + path;
const reactions = ['👍', '❤️', '👀', '✅', '⚠️'];

interface Props {
    message: IMessage;
    onChanged: () => Promise<void>;
}

const MessageCollaboration = ({message, onChanged}: Props) => {
    const {currentUser, snackManager} = useStores();
    const collaboration = message.collaboration || {};
    const [replyOpen, setReplyOpen] = React.useState(false);
    const [replyText, setReplyText] = React.useState('');
    const [thread, setThread] = React.useState<IMessage[]>();
    const [busy, setBusy] = React.useState(false);
    const uploadRef = React.useRef<HTMLInputElement | null>(null);

    const mutate = async (action: () => Promise<unknown>, notice?: string) => {
        if (busy) return;
        setBusy(true);
        try {
            await action();
            await onChanged();
            if (notice) snackManager.snack(notice);
        } finally {
            setBusy(false);
        }
    };

    const sendReply = async () => {
        const body = replyText.trim();
        if (!body) return;
        await mutate(
            () => axios.post(api('message/' + message.id + '/reply'), {message: body}),
            'Reply sent'
        );
        setReplyText('');
        setReplyOpen(false);
    };

    const openThread = async () => {
        const response = await axios.get<IMessage[]>(api('message/' + message.id + '/thread'));
        setThread(response.data);
    };

    const react = async (emoji: string, reactedByMe: boolean) => {
        await mutate(
            () =>
                reactedByMe
                    ? axios.delete(api('message/' + message.id + '/reaction'), {
                          params: {emoji},
                      })
                    : axios.post(api('message/' + message.id + '/reaction'), {emoji})
        );
    };

    const upload = async (file?: File) => {
        if (!file) return;
        const form = new FormData();
        form.append('attachment', file);
        await mutate(
            () =>
                axios.post(api('message/' + message.id + '/attachment'), form, {
                    headers: {'Content-Type': 'multipart/form-data'},
                }),
            'Attachment added'
        );
        if (uploadRef.current) uploadRef.current.value = '';
    };

    const status = collaboration.status || 'open';
    const assignedToMe = collaboration.assignedUserId === currentUser.user.id;

    return (
        <>
            <Stack spacing={1}>
                <Stack direction="row" spacing={0.75} useFlexGap sx={{flexWrap: 'wrap'}}>
                    <Button
                        size="small"
                        startIcon={<Reply />}
                        onClick={() => setReplyOpen(true)}>
                        Reply
                    </Button>
                    {(collaboration.replyCount || message.threadRootMessageId || message.replyToMessageId) && (
                        <Button size="small" startIcon={<Forum />} onClick={() => void openThread()}>
                            Thread
                            {collaboration.replyCount ? ' (' + collaboration.replyCount + ')' : ''}
                        </Button>
                    )}
                    <Button
                        size="small"
                        startIcon={<AssignmentInd />}
                        variant={assignedToMe ? 'contained' : 'text'}
                        onClick={() =>
                            void mutate(
                                () =>
                                    axios.put(api('message/' + message.id + '/assignment'), {
                                        userId: assignedToMe ? 0 : currentUser.user.id,
                                    }),
                                assignedToMe ? 'Assignment cleared' : 'Assigned to you'
                            )
                        }>
                        {assignedToMe
                            ? 'Assigned to me'
                            : collaboration.assignedUserName
                              ? 'Assigned: ' + collaboration.assignedUserName
                              : 'Assign to me'}
                    </Button>
                    <Button
                        size="small"
                        color={status === 'resolved' ? 'success' : 'inherit'}
                        startIcon={<TaskAlt />}
                        onClick={() =>
                            void mutate(
                                () =>
                                    axios.put(api('message/' + message.id + '/status'), {
                                        status: status === 'resolved' ? 'open' : 'resolved',
                                    }),
                                status === 'resolved' ? 'Message reopened' : 'Message resolved'
                            )
                        }>
                        {status === 'resolved' ? 'Resolved' : 'Resolve'}
                    </Button>
                    <Button
                        size="small"
                        startIcon={collaboration.read ? <MarkEmailUnread /> : <MarkEmailRead />}
                        onClick={() =>
                            void mutate(() =>
                                collaboration.read
                                    ? axios.delete(api('message/' + message.id + '/read'))
                                    : axios.post(api('message/' + message.id + '/read'))
                            )
                        }>
                        {collaboration.read ? 'Mark unread' : 'Mark read'}
                    </Button>
                    <input
                        ref={uploadRef}
                        type="file"
                        hidden
                        onChange={(event) => void upload(event.target.files?.[0])}
                    />
                    <Button
                        size="small"
                        startIcon={<AttachFile />}
                        onClick={() => uploadRef.current?.click()}>
                        Attach
                    </Button>
                </Stack>

                <Stack direction="row" spacing={0.5} useFlexGap sx={{flexWrap: 'wrap'}}>
                    {reactions.map((emoji) => {
                        const existing = collaboration.reactions?.find((item) => item.emoji === emoji);
                        return (
                            <Chip
                                key={emoji}
                                size="small"
                                clickable
                                variant={existing?.reactedByMe ? 'filled' : 'outlined'}
                                label={emoji + (existing?.count ? ' ' + existing.count : '')}
                                onClick={() => void react(emoji, Boolean(existing?.reactedByMe))}
                            />
                        );
                    })}
                </Stack>

                {collaboration.attachments && collaboration.attachments.length > 0 && (
                    <Stack direction="row" spacing={0.75} useFlexGap sx={{flexWrap: 'wrap'}}>
                        {collaboration.attachments.map((attachment) => (
                            <Chip
                                key={attachment.id}
                                icon={<AttachFile fontSize="small" />}
                                label={
                                    attachment.filename +
                                    ' · ' +
                                    formatBytes(attachment.size)
                                }
                                component="a"
                                clickable
                                href={api(attachment.url.replace(/^\//, ''))}
                            />
                        ))}
                    </Stack>
                )}

                {(collaboration.assignedUserName ||
                    collaboration.status === 'resolved' ||
                    collaboration.mentioned) && (
                    <Stack direction="row" spacing={0.75} useFlexGap sx={{flexWrap: 'wrap'}}>
                        {collaboration.assignedUserName && (
                            <Chip
                                size="small"
                                variant="outlined"
                                label={'Assigned to ' + collaboration.assignedUserName}
                            />
                        )}
                        {collaboration.status === 'resolved' && (
                            <Chip
                                size="small"
                                color="success"
                                variant="outlined"
                                label={
                                    collaboration.resolvedByName
                                        ? 'Resolved by ' + collaboration.resolvedByName
                                        : 'Resolved'
                                }
                            />
                        )}
                        {collaboration.mentioned && (
                            <Chip size="small" color="info" variant="outlined" label="Mentioned you" />
                        )}
                    </Stack>
                )}
            </Stack>

            <Dialog open={replyOpen} onClose={() => setReplyOpen(false)} fullWidth maxWidth="sm">
                <DialogTitle>Reply</DialogTitle>
                <DialogContent>
                    <Stack spacing={1.5} sx={{pt: 1}}>
                        <Typography variant="body2" color="text.secondary">
                            Replying to {message.title || 'message'}. Use @username to mention a
                            Channel member.
                        </Typography>
                        <TextField
                            autoFocus
                            multiline
                            minRows={4}
                            label="Reply"
                            value={replyText}
                            onChange={(event) => setReplyText(event.target.value)}
                        />
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setReplyOpen(false)}>Cancel</Button>
                    <Button
                        variant="contained"
                        disabled={busy || !replyText.trim()}
                        onClick={() => void sendReply()}>
                        Send Reply
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog open={thread !== undefined} onClose={() => setThread(undefined)} fullWidth maxWidth="md">
                <DialogTitle>Conversation Thread</DialogTitle>
                <DialogContent>
                    <Stack spacing={1.25} sx={{pt: 1}}>
                        {thread?.map((item) => (
                            <Box
                                key={item.id}
                                sx={{border: 1, borderColor: 'divider', borderRadius: 2, p: 1.25}}>
                                <Stack
                                    direction="row"
                                    spacing={1}
                                    sx={{justifyContent: 'space-between'}}>
                                    <Typography sx={{fontWeight: 700}}>
                                        {item.senderName || item.title}
                                    </Typography>
                                    <Typography variant="caption" color="text.secondary">
                                        {new Date(item.date).toLocaleString()}
                                    </Typography>
                                </Stack>
                                <Typography sx={{whiteSpace: 'pre-wrap', mt: 0.5}}>
                                    {item.message}
                                </Typography>
                            </Box>
                        ))}
                        {thread?.length === 0 && (
                            <Typography color="text.secondary">No thread messages.</Typography>
                        )}
                    </Stack>
                </DialogContent>
            </Dialog>
        </>
    );
};

const formatBytes = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KiB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MiB';
};

export default MessageCollaboration;
