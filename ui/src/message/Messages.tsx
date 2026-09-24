import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import React from 'react';
import {useParams} from 'react-router';
import DefaultPage from '../common/DefaultPage';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import Message from './Message';
import {observer} from 'mobx-react-lite';
import {IMessage} from '../types';
import ConfirmDialog from '../common/ConfirmDialog';
import LoadingSpinner from '../common/LoadingSpinner';
import {useStores} from '../stores';
import {Virtuoso} from 'react-virtuoso';
import {PushMessageDialog} from './PushMessageDialog';
import {enqueueSnackbar} from 'notistack';

const UndoAutoHideMs = 5000;

const Messages = observer(() => {
    const {id} = useParams<{id: string}>();
    const appId = id == null ? -1 : parseInt(id as string, 10);

    const [deleteAll, setDeleteAll] = React.useState(false);
    const [pushMessageOpen, setPushMessageOpen] = React.useState(false);
    const [isLoadingMore, setLoadingMore] = React.useState(false);
    const [showArchived, setShowArchived] = React.useState(false);
    const [chatDraft, setChatDraft] = React.useState('');
    const {messagesStore, appStore, currentUser} = useStores();

    const app = appId === -1 ? undefined : appStore.getByIDOrUndefined(appId);
    const messages = showArchived ? messagesStore.getArchived(appId) : messagesStore.get(appId);
    const hasMore = messagesStore.canLoadMore(appId, showArchived);
    const name = appStore.getName(appId);
    const hasMessages = messages.length !== 0;
    const expandedState = React.useRef<Record<number, boolean>>({});

    const isGlobalNonAdmin = Boolean(app?.autoAssign && !currentUser.user.admin);
    const isChatChannel = Boolean(app?.membersCanPost);
    const canPost = Boolean(
        app &&
            (app.ownerId === currentUser.user.id ||
                (app.membersCanPost && !showArchived))
    );

    const deleteMessage = (message: IMessage) => {
        const key = enqueueSnackbar({
            message: 'Message deleted',
            variant: 'info',
            action: () => (
                <Button
                    color="inherit"
                    size="small"
                    onClick={() => messagesStore.cancelPendingDelete(message)}>
                    Undo
                </Button>
            ),
            disableWindowBlurListener: true,
            transitionDuration: {enter: 0, exit: 0},
            autoHideDuration: UndoAutoHideMs,
            onExited: () => messagesStore.removeSingle(message),
        });
        messagesStore.addPendingDelete({message, key});
    };

    const messageAction = (message: IMessage) => {
        if (showArchived) {
            return void messagesStore.restoreSingle(message);
        }
        if (isGlobalNonAdmin) {
            return void messagesStore.archiveSingle(message);
        }
        deleteMessage(message);
    };

    React.useEffect(() => {
        if (!messagesStore.loaded(appId, showArchived)) {
            void messagesStore.loadMore(appId, showArchived);
        }
    }, [appId, showArchived]);

    const sendChatMessage = async () => {
        const text = chatDraft.trim();
        if (!app || !text) return;
        await messagesStore.sendMessage(app.id, text, '', app.defaultPriority);
        setChatDraft('');
    };

    const renderMessage = (_index: number, message: IMessage) => (
        <Message
            key={message.id}
            fDelete={() => messageAction(message)}
            action={showArchived ? 'restore' : isGlobalNonAdmin ? 'archive' : 'delete'}
            chatMode={isChatChannel}
            ownMessage={message.senderUserId === currentUser.user.id}
            senderName={message.senderName}
            onExpand={(expanded) => (expandedState.current[message.id] = expanded)}
            title={message.title}
            date={message.date}
            appName={appStore.getName(message.appid)}
            expanded={expandedState.current[message.id] ?? false}
            content={message.message}
            image={message.image}
            extras={message.extras}
            priority={message.priority}
        />
    );

    const checkIfLoadMore = () => {
        if (!isLoadingMore && messagesStore.canLoadMore(appId, showArchived)) {
            setLoadingMore(true);
            messagesStore
                .loadMore(appId, showArchived)
                .then(() => setLoadingMore(false));
        }
    };

    const messageFooter = () => {
        if (hasMore) {
            return <LoadingSpinner />;
        }
        if (hasMessages) {
            return label("You've reached the end");
        }
        return null;
    };

    const renderMessages = () => (
        <Virtuoso
            id="messages"
            style={{width: '100%'}}
            useWindowScroll
            totalCount={messages.length}
            endReached={checkIfLoadMore}
            data={messages}
            itemContent={renderMessage}
            components={{
                Footer: messageFooter,
                EmptyPlaceholder: () => label(showArchived ? 'No archived messages' : 'No messages'),
            }}
        />
    );

    const label = (text: string) => (
        <Grid size={{xs: 12}}>
            <Typography variant="caption" component="div" gutterBottom align="center">
                {text}
            </Typography>
        </Grid>
    );

    const bulkActionLabel = showArchived
        ? 'Restore All'
        : isGlobalNonAdmin
          ? 'Archive All'
          : 'Delete All';

    const performBulkAction = async () => {
        if (showArchived) {
            await messagesStore.restoreByApp(appId);
        } else if (isGlobalNonAdmin) {
            await messagesStore.archiveByApp(appId);
        } else {
            await messagesStore.removeByApp(appId);
        }
    };

    return (
        <DefaultPage
            title={name + (isChatChannel ? ' · Chat' : '')}
            rightControl={
                <div>
                    <Button
                        variant={showArchived ? 'outlined' : 'contained'}
                        color="primary"
                        onClick={() => setShowArchived(false)}
                        style={{marginRight: 5}}>
                        Active
                    </Button>
                    <Button
                        variant={showArchived ? 'contained' : 'outlined'}
                        color="primary"
                        onClick={() => setShowArchived(true)}
                        style={{marginRight: 5}}>
                        Archived
                    </Button>
                    {app && canPost && !isChatChannel && !showArchived && (
                        <Button
                            id="push-message"
                            variant="contained"
                            color="primary"
                            onClick={() => setPushMessageOpen(true)}
                            style={{marginRight: 5}}>
                            Push Message
                        </Button>
                    )}
                    <Button
                        id="refresh-all"
                        variant="contained"
                        color="primary"
                        onClick={() => messagesStore.refreshByApp(appId, showArchived)}
                        style={{marginRight: 5}}>
                        Refresh
                    </Button>
                    <Button
                        id="delete-all"
                        variant="contained"
                        disabled={!hasMessages}
                        color="primary"
                        onClick={() => setDeleteAll(true)}>
                        {bulkActionLabel}
                    </Button>
                </div>
            }>
            {isChatChannel && canPost && !showArchived && (
                <Paper
                    elevation={4}
                    sx={{
                        p: 1.5,
                        mb: 2,
                        display: 'flex',
                        gap: 1,
                        alignItems: 'flex-end',
                        position: 'sticky',
                        top: 64,
                        zIndex: 2,
                    }}>
                    <TextField
                        fullWidth
                        multiline
                        maxRows={5}
                        label="Message"
                        placeholder={'Message ' + name}
                        value={chatDraft}
                        onChange={(event) => setChatDraft(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === 'Enter' && !event.shiftKey) {
                                event.preventDefault();
                                void sendChatMessage();
                            }
                        }}
                    />
                    <Button
                        variant="contained"
                        disabled={!chatDraft.trim()}
                        onClick={() => void sendChatMessage()}>
                        Send
                    </Button>
                </Paper>
            )}

            {!messagesStore.loaded(appId, showArchived) ? <LoadingSpinner /> : renderMessages()}

            {deleteAll && (
                <ConfirmDialog
                    title={
                        showArchived
                            ? 'Restore Archived Messages'
                            : isGlobalNonAdmin
                              ? 'Archive Messages'
                              : 'Confirm Delete'
                    }
                    text={
                        showArchived
                            ? 'Restore all archived messages?'
                            : isGlobalNonAdmin
                              ? 'Archive all messages from your view? Other users will not be affected.'
                              : 'Delete all messages?'
                    }
                    fClose={() => setDeleteAll(false)}
                    fOnSubmit={performBulkAction}
                />
            )}
            {pushMessageOpen && app && (
                <PushMessageDialog
                    appName={app.name}
                    defaultPriority={app.defaultPriority}
                    fClose={() => setPushMessageOpen(false)}
                    fOnSubmit={(message, title, priority) =>
                        messagesStore.sendMessage(app.id, message, title, priority)
                    }
                />
            )}
        </DefaultPage>
    );
});

export default Messages;
