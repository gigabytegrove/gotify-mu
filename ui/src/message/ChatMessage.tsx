import Archive from '@mui/icons-material/Archive';
import Delete from '@mui/icons-material/Delete';
import Unarchive from '@mui/icons-material/Unarchive';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import React from 'react';
import TimeAgo from 'react-timeago';

import {IMessage} from '../types';
import {TimeAgoFormatter} from '../common/TimeAgoFormatter';
import MessageCollaboration from './MessageCollaboration';

interface Props {
    message: IMessage;
    mine: boolean;
    onRefresh: () => Promise<void>;
    onArchive?: VoidFunction;
    onRestore?: VoidFunction;
    onDelete?: VoidFunction;
}

const initial = (value: string) => value.trim().slice(0, 1).toUpperCase() || '?';

const ChatMessage = ({
    message,
    mine,
    onRefresh,
    onArchive,
    onRestore,
    onDelete,
}: Props) => {
    const sender = message.senderName || message.title || 'Member';
    const parts = message.message.split(/(@[A-Za-z0-9._-]+)/g);

    return (
        <Box
            className="chat-message"
            sx={{
                display: 'flex',
                justifyContent: mine ? 'flex-end' : 'flex-start',
                gap: 1,
                px: {xs: 1, sm: 2},
                py: 0.65,
            }}>
            {!mine && (
                <Avatar
                    sx={{
                        width: 34,
                        height: 34,
                        mt: 0.25,
                        bgcolor: 'primary.main',
                        fontSize: '0.82rem',
                        fontWeight: 800,
                    }}>
                    {initial(sender)}
                </Avatar>
            )}
            <Box sx={{maxWidth: {xs: '88%', sm: '76%', lg: '68%'}, minWidth: 0}}>
                {!mine && (
                    <Typography
                        variant="caption"
                        sx={{display: 'block', mb: 0.35, ml: 0.5, fontWeight: 750}}>
                        {sender}
                    </Typography>
                )}
                <Box
                    sx={{
                        borderRadius: 3,
                        borderBottomRightRadius: mine ? 0.75 : 3,
                        borderBottomLeftRadius: mine ? 3 : 0.75,
                        bgcolor: mine ? 'primary.main' : 'background.paper',
                        color: mine ? 'primary.contrastText' : 'text.primary',
                        border: 1,
                        borderColor: mine ? 'primary.main' : 'divider',
                        boxShadow: mine ? '0 7px 22px rgba(37,99,235,0.18)' : 1,
                        px: 1.6,
                        py: 1.15,
                    }}>
                    {message.message && (
                        <Typography
                            component="div"
                            sx={{
                                whiteSpace: 'pre-wrap',
                                overflowWrap: 'anywhere',
                                fontSize: '0.95rem',
                                lineHeight: 1.55,
                            }}>
                            {parts.map((part, index) =>
                                /^@[A-Za-z0-9._-]+$/.test(part) ? (
                                    <Box
                                        component="span"
                                        key={part + index}
                                        sx={{
                                            display: 'inline',
                                            fontWeight: 850,
                                            borderRadius: 0.75,
                                            px: 0.35,
                                            bgcolor: mine ? 'rgba(255,255,255,0.16)' : 'action.hover',
                                            color: mine ? 'inherit' : 'primary.main',
                                        }}>
                                        {part}
                                    </Box>
                                ) : (
                                    <React.Fragment key={index}>{part}</React.Fragment>
                                )
                            )}
                        </Typography>
                    )}
                    <Stack
                        direction="row"
                        spacing={0.75}
                        useFlexGap
                        sx={{
                            mt: message.message ? 0.75 : 0,
                            alignItems: 'center',
                            justifyContent: 'flex-end',
                            flexWrap: 'wrap',
                        }}>
                        {message.collaboration?.mentioned && (
                            <Chip
                                size="small"
                                label="Mentioned you"
                                sx={{
                                    bgcolor: mine ? 'rgba(255,255,255,0.14)' : 'info.main',
                                    color: mine ? 'inherit' : 'info.contrastText',
                                    height: 20,
                                }}
                            />
                        )}
                        <Typography
                            variant="caption"
                            title={message.date}
                            sx={{color: mine ? 'rgba(255,255,255,0.72)' : 'text.secondary'}}>
                            <TimeAgo date={message.date} formatter={TimeAgoFormatter.long} />
                        </Typography>
                    </Stack>
                </Box>

                <Box
                    sx={{
                        mt: 0.55,
                        px: 0.25,
                        '& .MuiButton-root': {minHeight: 27, fontSize: '0.72rem'},
                        '& .MuiChip-root': {height: 23},
                    }}>
                    <MessageCollaboration message={message} onChanged={onRefresh} />
                </Box>

                <Stack
                    direction="row"
                    spacing={0.25}
                    sx={{justifyContent: mine ? 'flex-end' : 'flex-start', mt: 0.15}}>
                    {onRestore && (
                        <Tooltip title="Restore">
                            <IconButton size="small" onClick={onRestore}>
                                <Unarchive fontSize="small" />
                            </IconButton>
                        </Tooltip>
                    )}
                    {onArchive && (
                        <Tooltip title="Archive for me">
                            <IconButton size="small" onClick={onArchive}>
                                <Archive fontSize="small" />
                            </IconButton>
                        </Tooltip>
                    )}
                    {onDelete && (
                        <Tooltip title="Delete">
                            <IconButton size="small" onClick={onDelete}>
                                <Delete fontSize="small" />
                            </IconButton>
                        </Tooltip>
                    )}
                </Stack>
            </Box>
        </Box>
    );
};

export default ChatMessage;
