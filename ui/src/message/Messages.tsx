import React from 'react';
import {
    Box,
    Button,
    ButtonGroup,
    Chip,
    Stack,
    Tooltip,
    Typography,
} from '@mui/material';
import Archive from '@mui/icons-material/Archive';
import DeleteSweep from '@mui/icons-material/DeleteSweep';
import Refresh from '@mui/icons-material/Refresh';
import Restore from '@mui/icons-material/Restore';
import Send from '@mui/icons-material/Send';
import {useParams} from 'react-router';
import DefaultPage from '../common/DefaultPage';
import Message from './Message';
import {observer} from 'mobx-react-lite';
import {IMessage} from '../types';
import ConfirmDialog from '../common/ConfirmDialog';
import LoadingSpinner from '../common/LoadingSpinner';
import {useStores} from '../stores';
import {Virtuoso} from 'react-virtuoso';
import {PushMessageDialog} from './PushMessageDialog';
import ChatComposer from './ChatComposer';
import {enqueueSnackbar} from 'notistack';

const UndoAutoHideMs = 5000;

const Messages = observer(() => {
    const {id} = useParams<{id: string}>();
    const appId = id == null ? -1 : parseInt(id, 10);

    const [deleteAll, setDeleteAll] = React.useState(false);
    const [pushMessageOpen, setPushMessageOpen] = React.useState(false);
    const [archivedView, setArchivedView] = React.useState(false);
    const [isLoadingMore, setLoadingMore] = React.useState(false);
    const {messagesStore, appStore, currentUser} = useStores();

    const messages = archivedView ? messagesStore.getArchived(appId) : messagesStore.get(appId);
    const hasMore = messagesStore.canLoadMore(appId, archivedView);
    const name = appStore.getName(appId);
    const hasMessages = messages.length !== 0;
    const expandedState = React.useRef<Record<number, boolean>>({});
    const app = appId === -1 ? undefined : appStore.getByIDOrUndefined(appId);

    const canPost =
        app != null && (app.ownerId === currentUser.user.id || Boolean(app.allowMemberPost));

    const canDeleteAll =
        !archivedView &&
        (appId === -1 ? currentUser.user.admin : !app?.autoAssign || currentUser.user.admin);

    const canDeleteMessage = (message: IMessage): boolean => {
        const messageApp = appStore.getByIDOrUndefined(message.appid);
        return !messageApp?.autoAssign || currentUser.user.admin;
    };

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

    React.useEffect(() => {
        if (!messagesStore.loaded(appId, archivedView)) {
            void messagesStore.loadMore(appId, archivedView);
        }
    }, [appId, archivedView, messagesStore]);

    const renderMessage = (_index: number, item: IMessage) => (
        <Message
            key={item.id}
            fDelete={
                !archivedView && canDeleteMessage(item)
                    ? () => deleteMessage(item)
                    : undefined
            }
            fArchive={
                !archivedView ? () => void messagesStore.archiveSingle(item) : undefined
            }
            fRestore={
                archivedView ? () => void messagesStore.restoreSingle(item) : undefined
            }
            senderName={item.senderName}
            onExpand={(expanded) => (expandedState.current[item.id] = expanded)}
            title={item.title}
            date={item.date}
            appName={appStore.getName(item.appid)}
            expanded={expandedState.current[item.id] ?? false}
            content={item.message}
            image={item.image}
            extras={item.extras}
            priority={item.priority}
        />
    );

    const checkIfLoadMore = () => {
        if (!isLoadingMore && messagesStore.canLoadMore(appId, archivedView)) {
            setLoadingMore(true);
            messagesStore
                .loadMore(appId, archivedView)
                .finally(() => setLoadingMore(false));
        }
    };

    const footer = () => {
        if (hasMore) return <LoadingSpinner />;
        if (!hasMessages) return null;
        return (
            <Typography
                variant="caption"
                color="text.secondary"
                component="div"
                align="center"
                sx={{py: 2}}>
                End of messages
            </Typography>
        );
    };

    const empty = () => (
        <Box sx={{py: 8, textAlign: 'center'}}>
            <Typography variant="h6">
                {archivedView ? 'Archive is empty' : 'No messages'}
            </Typography>
            <Typography color="text.secondary">
                {archivedView
                    ? 'Messages you archive will appear here.'
                    : 'New notifications will appear here when they arrive.'}
            </Typography>
        </Box>
    );

    const title = appId === -1 ? 'Messages' : name;
    const description =
        appId === -1
            ? 'Notifications from every Channel available to your account.'
            : archivedView
              ? 'Archived messages are private to your account and can be restored.'
              : app?.autoAssign
                ? 'Global Channel · available to all users.'
                : 'Channel message history.';

    return (
        <DefaultPage
            title={title}
            description={description}
            rightControl={
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                    {!archivedView && canPost && app && !app.allowMemberPost && (
                        <Button
                            id="push-message"
                            variant="contained"
                            startIcon={<Send />}
                            onClick={() => setPushMessageOpen(true)}>
                            Send Notification
                        </Button>
                    )}

                    <ButtonGroup variant="outlined">
                        <Button
                            variant={!archivedView ? 'contained' : 'outlined'}
                            onClick={() => setArchivedView(false)}>
                            Active
                        </Button>
                        <Button
                            variant={archivedView ? 'contained' : 'outlined'}
                            onClick={() => setArchivedView(true)}>
                            Archive
                        </Button>
                    </ButtonGroup>

                    <Tooltip title="Refresh">
                        <Button
                            id="refresh-all"
                            variant="outlined"
                            onClick={() => messagesStore.refreshByApp(appId, archivedView)}>
                            <Refresh />
                        </Button>
                    </Tooltip>

                    {!archivedView && (
                        <Button
                            id="archive-all"
                            variant="outlined"
                            startIcon={<Archive />}
                            disabled={!hasMessages}
                            onClick={() => void messagesStore.archiveByApp(appId)}>
                            Archive All
                        </Button>
                    )}

                    {archivedView && (
                        <Button
                            id="restore-all"
                            variant="outlined"
                            startIcon={<Restore />}
                            disabled={!hasMessages}
                            onClick={() => void messagesStore.restoreByApp(appId)}>
                            Restore All
                        </Button>
                    )}

                    {canDeleteAll && (
                        <Button
                            id="delete-all"
                            variant="outlined"
                            startIcon={<DeleteSweep />}
                            disabled={!hasMessages}
                            onClick={() => setDeleteAll(true)}>
                            Delete All
                        </Button>
                    )}
                </Stack>
            }>
            <Stack direction="row" spacing={1}>
                {app?.autoAssign && <Chip size="small" label="Global Channel" />}
                {app?.receiveNotifications === false && (
                    <Chip size="small" variant="outlined" label="Notifications Muted" />
                )}
                {archivedView && <Chip size="small" variant="outlined" label="My Archive" />}
            </Stack>

            {!archivedView && app?.allowMemberPost && canPost && (
                <ChatComposer
                    channelName={app.name}
                    fOnSubmit={(text) =>
                        messagesStore.sendMessage(app.id, text, '', app.defaultPriority)
                    }
                />
            )}

            {!messagesStore.loaded(appId, archivedView) ? (
                <LoadingSpinner />
            ) : (
                <Virtuoso
                    id="messages"
                    style={{width: '100%'}}
                    useWindowScroll
                    totalCount={messages.length}
                    endReached={checkIfLoadMore}
                    data={messages}
                    itemContent={renderMessage}
                    components={{
                        Footer: footer,
                        EmptyPlaceholder: empty,
                    }}
                />
            )}

            {deleteAll && (
                <ConfirmDialog
                    title="Delete Messages"
                    text={
                        app?.autoAssign
                            ? 'Permanently delete all messages from this Global Channel for everyone?'
                            : 'Delete all visible messages?'
                    }
                    fClose={() => setDeleteAll(false)}
                    fOnSubmit={() => messagesStore.removeByApp(appId)}
                />
            )}

            {pushMessageOpen && app && (
                <PushMessageDialog
                    appName={app.name}
                    defaultPriority={app.defaultPriority}
                    fClose={() => setPushMessageOpen(false)}
                    fOnSubmit={(text, messageTitle, priority) =>
                        messagesStore.sendMessage(app.id, text, messageTitle, priority)
                    }
                />
            )}
        </DefaultPage>
    );
});

export default Messages;
