import React from 'react';
import {
    Button,
    Chip,
    Grid,
    Stack,
    Typography,
} from '@mui/material';
import Archive from '@mui/icons-material/Archive';
import Delete from '@mui/icons-material/Delete';
import Refresh from '@mui/icons-material/Refresh';
import Restore from '@mui/icons-material/Restore';
import Public from '@mui/icons-material/Public';
import Forum from '@mui/icons-material/Forum';
import NotificationsOff from '@mui/icons-material/NotificationsOff';
import {useParams} from 'react-router';
import {observer} from 'mobx-react-lite';
import {Virtuoso} from 'react-virtuoso';
import {enqueueSnackbar} from 'notistack';

import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import LoadingSpinner from '../common/LoadingSpinner';
import Message from './Message';
import {IMessage} from '../types';
import {useStores} from '../stores';
import {PushMessageDialog} from './PushMessageDialog';
import ChatComposer from './ChatComposer';

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
    const app = appId === -1 ? undefined : appStore.getByIDOrUndefined(appId);
    const name = appStore.getName(appId);
    const expandedState = React.useRef<Record<number, boolean>>({});

    const canPost =
        app != null &&
        (app.ownerId === currentUser.user.id || Boolean(app.allowMemberPost));

    const canDeleteAll =
        !archivedView &&
        (appId === -1
            ? currentUser.user.admin
            : !app?.autoAssign || currentUser.user.admin);

    const canDeleteMessage = (message: IMessage) => {
        const messageApp = appStore.getByIDOrUndefined(message.appid);
        return !messageApp?.autoAssign || currentUser.user.admin;
    };

    React.useEffect(() => {
        if (!messagesStore.loaded(appId, archivedView)) {
            void messagesStore.loadMore(appId, archivedView);
        }
    }, [appId, archivedView, messagesStore]);

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

    const renderMessage = (_index: number, message: IMessage) => (
        <Message
            key={message.id}
            fDelete={
                !archivedView && canDeleteMessage(message)
                    ? () => deleteMessage(message)
                    : undefined
            }
            fArchive={
                !archivedView ? () => void messagesStore.archiveSingle(message) : undefined
            }
            fRestore={
                archivedView ? () => void messagesStore.restoreSingle(message) : undefined
            }
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
        if (isLoadingMore || !messagesStore.canLoadMore(appId, archivedView)) return;
        setLoadingMore(true);
        messagesStore
            .loadMore(appId, archivedView)
            .finally(() => setLoadingMore(false));
    };

    const emptyLabel = archivedView ? 'No archived messages' : 'No messages';

    const pageDescription =
        appId === -1
            ? 'Messages from every Channel available to your account.'
            : app?.description || 'Review and manage this Channel’s message history.';

    return (
        <DefaultPage
            title={appId === -1 ? 'Messages' : name}
            description={pageDescription}
            rightControl={
                <Stack direction="row" spacing={1} sx={{flexWrap: 'wrap'}} useFlexGap>
                    {app?.autoAssign && <Chip size="small" icon={<Public />} label="Global" />}
                    {app?.allowMemberPost && (
                        <Chip size="small" icon={<Forum />} label="Chat · Experimental" />
                    )}
                    {app?.receiveNotifications === false && (
                        <Chip size="small" icon={<NotificationsOff />} label="Muted" />
                    )}
                    <Chip
                        size="small"
                        variant={archivedView ? 'filled' : 'outlined'}
                        label={archivedView ? 'Archive' : 'Active'}
                    />
                </Stack>
            }>
            {app?.allowMemberPost && !archivedView && canPost && (
                <SurfaceCard
                    title="Conversation"
                    subtitle="Experimental Web-only posting. Standard Gotify mobile clients remain receive-only.">
                    <ChatComposer
                        channelName={app.name}
                        fOnSubmit={(message) =>
                            messagesStore.sendMessage(app.id, message, '', app.defaultPriority)
                        }
                    />
                </SurfaceCard>
            )}

            <SurfaceCard
                title={archivedView ? 'Archived Messages' : 'Message History'}
                subtitle={
                    appId === -1
                        ? 'Your combined message stream.'
                        : `${messages.length} message${messages.length === 1 ? '' : 's'} currently loaded`
                }
                action={
                    <Stack direction="row" spacing={1} sx={{flexWrap: 'wrap'}} useFlexGap>
                        {!archivedView && canPost && app && !app.allowMemberPost && (
                            <Button
                                id="push-message"
                                variant="contained"
                                onClick={() => setPushMessageOpen(true)}>
                                Push Message
                            </Button>
                        )}
                        <Button
                            id="toggle-archive-view"
                            variant="outlined"
                            startIcon={archivedView ? <Restore /> : <Archive />}
                            onClick={() => setArchivedView((current) => !current)}>
                            {archivedView ? 'Active Messages' : 'Archive'}
                        </Button>
                        <Button
                            id="refresh-all"
                            variant="outlined"
                            startIcon={<Refresh />}
                            onClick={() => messagesStore.refreshByApp(appId, archivedView)}>
                            Refresh
                        </Button>
                        {!archivedView && (
                            <Button
                                id="archive-all"
                                variant="outlined"
                                startIcon={<Archive />}
                                disabled={messages.length === 0}
                                onClick={() => void messagesStore.archiveByApp(appId)}>
                                Archive All
                            </Button>
                        )}
                        {archivedView && (
                            <Button
                                id="restore-all"
                                variant="outlined"
                                startIcon={<Restore />}
                                disabled={messages.length === 0}
                                onClick={() => void messagesStore.restoreByApp(appId)}>
                                Restore All
                            </Button>
                        )}
                        {canDeleteAll && (
                            <Button
                                id="delete-all"
                                color="error"
                                variant="outlined"
                                startIcon={<Delete />}
                                disabled={messages.length === 0}
                                onClick={() => setDeleteAll(true)}>
                                Delete All
                            </Button>
                        )}
                    </Stack>
                }>
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
                            Footer: () =>
                                hasMore ? (
                                    <LoadingSpinner />
                                ) : messages.length > 0 ? (
                                    <Grid size={12}>
                                        <Typography
                                            variant="caption"
                                            component="div"
                                            sx={{py: 1}}
                                            align="center"
                                            color="text.secondary">
                                            You've reached the end
                                        </Typography>
                                    </Grid>
                                ) : null,
                            EmptyPlaceholder: () => (
                                <Typography
                                    color="text.secondary"
                                    align="center"
                                    sx={{py: 5}}>
                                    {emptyLabel}
                                </Typography>
                            ),
                        }}
                    />
                )}
            </SurfaceCard>

            {deleteAll && (
                <ConfirmDialog
                    title="Delete Messages"
                    text={
                        app?.autoAssign
                            ? 'Delete all messages from this Global Channel for everyone?'
                            : 'Delete all messages in this view?'
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
                    fOnSubmit={(message, title, priority) =>
                        messagesStore.sendMessage(app.id, message, title, priority)
                    }
                />
            )}
        </DefaultPage>
    );
});

export default Messages;
