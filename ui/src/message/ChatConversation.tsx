import Archive from '@mui/icons-material/Archive';
import Forum from '@mui/icons-material/Forum';
import Refresh from '@mui/icons-material/Refresh';
import Restore from '@mui/icons-material/Restore';
import Search from '@mui/icons-material/Search';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import InputAdornment from '@mui/material/InputAdornment';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import React from 'react';

import * as config from '../config';
import {IApplication, IMessage} from '../types';
import ChatComposer from './ChatComposer';
import ChatMessage from './ChatMessage';

interface Props {
    app: IApplication;
    currentUserId: number;
    messages: IMessage[];
    loaded: boolean;
    hasMore: boolean;
    archivedView: boolean;
    canPost: boolean;
    typingLabel: string;
    query: string;
    onQuery: (value: string) => void;
    onLoadEarlier: () => Promise<void>;
    onRefresh: () => Promise<void>;
    onToggleArchive: () => void;
    onAdvancedSearch: () => void;
    onSend: (message: string, images: File[]) => Promise<void>;
    onTyping: (typing: boolean) => Promise<void> | void;
    onArchiveMessage: (message: IMessage) => void;
    onRestoreMessage: (message: IMessage) => void;
    onDeleteMessage: (message: IMessage) => void;
    canDeleteMessage: (message: IMessage) => boolean;
}

const dayLabel = (value: string) => {
    const date = new Date(value);
    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(today.getDate() - 1);

    const sameDay = (left: Date, right: Date) =>
        left.getFullYear() === right.getFullYear() &&
        left.getMonth() === right.getMonth() &&
        left.getDate() === right.getDate();

    if (sameDay(date, today)) return 'Today';
    if (sameDay(date, yesterday)) return 'Yesterday';
    return date.toLocaleDateString(undefined, {
        weekday: 'short',
        month: 'short',
        day: 'numeric',
        year: date.getFullYear() === today.getFullYear() ? undefined : 'numeric',
    });
};

const ChatConversation = ({
    app,
    currentUserId,
    messages,
    loaded,
    hasMore,
    archivedView,
    canPost,
    typingLabel,
    query,
    onQuery,
    onLoadEarlier,
    onRefresh,
    onToggleArchive,
    onAdvancedSearch,
    onSend,
    onTyping,
    onArchiveMessage,
    onRestoreMessage,
    onDeleteMessage,
    canDeleteMessage,
}: Props) => {
    const scrollRef = React.useRef<HTMLDivElement | null>(null);
    const firstPaint = React.useRef(true);
    const normalized = query.trim().toLowerCase();
    const filtered = normalized
        ? messages.filter(
              (message) =>
                  message.message.toLowerCase().includes(normalized) ||
                  message.title.toLowerCase().includes(normalized) ||
                  message.senderName?.toLowerCase().includes(normalized)
          )
        : messages;
    const chronological = [...filtered].reverse();

    React.useEffect(() => {
        const element = scrollRef.current;
        if (!element || archivedView) return;
        const nearBottom =
            firstPaint.current || element.scrollHeight - element.scrollTop - element.clientHeight < 180;
        if (nearBottom) {
            window.requestAnimationFrame(() => {
                element.scrollTop = element.scrollHeight;
                firstPaint.current = false;
            });
        }
    }, [app.id, archivedView, chronological.length]);

    React.useEffect(() => {
        firstPaint.current = true;
    }, [app.id]);

    let previousDay = '';

    return (
        <Box sx={{width: '100%', maxWidth: 1220, mx: 'auto'}}>
            <Paper
                variant="outlined"
                sx={{
                    height: {
                        xs: 'calc(100dvh - 98px)',
                        sm: 'calc(100dvh - 126px)',
                    },
                    minHeight: {sm: 600},
                    display: 'flex',
                    flexDirection: 'column',
                    overflow: 'hidden',
                    borderRadius: {xs: 2, sm: 3.5},
                    boxShadow: {sm: '0 18px 50px rgba(15,23,42,0.08)'},
                }}>
                <Stack
                    direction="row"
                    spacing={1.25}
                    sx={{
                        px: {xs: 1.25, sm: 2},
                        py: 1.25,
                        alignItems: 'center',
                        borderBottom: 1,
                        borderColor: 'divider',
                        bgcolor: 'background.paper',
                    }}>
                    <Avatar
                        src={config.get('url') + app.image}
                        variant="rounded"
                        sx={{width: 42, height: 42}}
                    />
                    <Box sx={{minWidth: 0, flex: 1}}>
                        <Stack direction="row" spacing={0.75} sx={{alignItems: 'center'}}>
                            <Typography variant="h6" noWrap>
                                {app.name}
                            </Typography>
                            <Chip size="small" icon={<Forum />} label="Chat" />
                            {app.receiveNotifications === false && (
                                <Chip size="small" variant="outlined" label="Muted" />
                            )}
                        </Stack>
                        <Typography variant="caption" color="text.secondary" noWrap>
                            {app.description || 'Conversation'}
                        </Typography>
                    </Box>
                    <Stack direction="row" spacing={0.5}>
                        <Button
                            size="small"
                            variant="text"
                            startIcon={<Search />}
                            onClick={onAdvancedSearch}
                            sx={{display: {xs: 'none', md: 'inline-flex'}}}>
                            Search
                        </Button>
                        <Button
                            size="small"
                            variant="text"
                            startIcon={archivedView ? <Restore /> : <Archive />}
                            onClick={onToggleArchive}>
                            {archivedView ? 'Active' : 'Archive'}
                        </Button>
                        <Button
                            size="small"
                            variant="text"
                            startIcon={<Refresh />}
                            onClick={() => void onRefresh()}
                            sx={{display: {xs: 'none', sm: 'inline-flex'}}}>
                            Refresh
                        </Button>
                    </Stack>
                </Stack>

                <Box
                    sx={{
                        px: {xs: 1, sm: 2},
                        py: 1,
                        borderBottom: 1,
                        borderColor: 'divider',
                        bgcolor: 'action.hover',
                    }}>
                    <TextField
                        fullWidth
                        value={query}
                        onChange={(event) => onQuery(event.target.value)}
                        placeholder="Search this conversation"
                        aria-label="Search this conversation"
                        slotProps={{
                            input: {
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <Search fontSize="small" />
                                    </InputAdornment>
                                ),
                            },
                        }}
                    />
                </Box>

                <Box
                    ref={scrollRef}
                    sx={{
                        flex: 1,
                        minHeight: 0,
                        overflowY: 'auto',
                        py: 1.25,
                        bgcolor: (theme) =>
                            theme.palette.mode === 'dark' ? '#0b1220' : '#f8fafc',
                        backgroundImage: (theme) =>
                            theme.palette.mode === 'dark'
                                ? 'radial-gradient(circle at 20px 20px, rgba(148,163,184,0.035) 1px, transparent 0)'
                                : 'radial-gradient(circle at 20px 20px, rgba(37,99,235,0.045) 1px, transparent 0)',
                        backgroundSize: '28px 28px',
                    }}>
                    {hasMore && !normalized && (
                        <Box sx={{display: 'flex', justifyContent: 'center', pb: 1}}>
                            <Button size="small" variant="outlined" onClick={() => void onLoadEarlier()}>
                                Load earlier messages
                            </Button>
                        </Box>
                    )}

                    {!loaded ? (
                        <Typography color="text.secondary" align="center" sx={{py: 5}}>
                            Loading conversation…
                        </Typography>
                    ) : chronological.length === 0 ? (
                        <Typography color="text.secondary" align="center" sx={{py: 8}}>
                            {normalized
                                ? 'No messages match your search.'
                                : archivedView
                                  ? 'No archived chat messages.'
                                  : 'No messages yet. Start the conversation below.'}
                        </Typography>
                    ) : (
                        chronological.map((message) => {
                            const currentDay = dayLabel(message.date);
                            const showDay = currentDay !== previousDay;
                            previousDay = currentDay;
                            return (
                                <React.Fragment key={message.id}>
                                    {showDay && (
                                        <Box sx={{display: 'flex', justifyContent: 'center', py: 1}}>
                                            <Chip
                                                size="small"
                                                variant="outlined"
                                                label={currentDay}
                                                sx={{bgcolor: 'background.paper'}}
                                            />
                                        </Box>
                                    )}
                                    <ChatMessage
                                        message={message}
                                        mine={message.senderUserId === currentUserId}
                                        onRefresh={() => onRefresh()}
                                        onArchive={
                                            archivedView
                                                ? undefined
                                                : () => onArchiveMessage(message)
                                        }
                                        onRestore={
                                            archivedView
                                                ? () => onRestoreMessage(message)
                                                : undefined
                                        }
                                        onDelete={
                                            !archivedView && canDeleteMessage(message)
                                                ? () => onDeleteMessage(message)
                                                : undefined
                                        }
                                    />
                                </React.Fragment>
                            );
                        })
                    )}
                </Box>

                {!archivedView && canPost && (
                    <Box
                        sx={{
                            borderTop: 1,
                            borderColor: 'divider',
                            bgcolor: 'background.paper',
                            px: {xs: 1, sm: 1.5},
                            pt: 0.75,
                        }}>
                        <Typography
                            variant="caption"
                            color="text.secondary"
                            sx={{
                                display: 'block',
                                minHeight: 20,
                                px: 1,
                                fontStyle: typingLabel ? 'italic' : 'normal',
                            }}>
                            {typingLabel}
                        </Typography>
                        <ChatComposer
                            appId={app.id}
                            channelName={app.name}
                            fOnSubmit={onSend}
                            fOnTyping={onTyping}
                        />
                    </Box>
                )}
            </Paper>
        </Box>
    );
};

export default ChatConversation;
