import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import React from 'react';
import {useParams} from 'react-router';
import DefaultPage from '../common/DefaultPage';
import Button from '@mui/material/Button';
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
    const appId = id == null ? -1 : parseInt(id as string, 10);

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
        app != null &&
        (app.ownerId === currentUser.user.id || Boolean(app.allowMemberPost));

    const canDeleteAll =
        !archivedView &&
        (appId === -1
            ? currentUser.user.admin
            : !app?.autoAssign || currentUser.user.admin);

    const canDeleteMessage = (message: IMessage): boolean => {
        const messageApp = appStore.getByIDOrUndefined(message.appid);
        if (messageApp?.autoAssign) {
            return currentUser.user.admin;
        }
        return true;
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
            messagesStore.loadMore(appId, archivedView);
        }
    }, [appId, archivedView]);

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
        if (!isLoadingMore && messagesStore.canLoadMore(appId, archivedView)) {
            setLoadingMore(true);
            messagesStore.loadMore(appId, archivedView).then(() => setLoadingMore(false));
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
                EmptyPlaceholder: () =>
                    label(archivedView ? 'No archived messages' : 'No messages'),
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

    return (
        <DefaultPage
            title={
                app?.allowMemberPost
                    ? `${name} · Chat${archivedView ? ' · Archived' : ''}`
                    : `${name}${archivedView ? ' · Archived' : ''}`
            }
            rightControl={
                <div>
                    {!archivedView && canPost && app && !app.allowMemberPost && (
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
                        id="toggle-archive-view"
                        variant="contained"
                        color="primary"
                        onClick={() => setArchivedView((current) => !current)}
                        style={{marginRight: 5}}>
                        {archivedView ? 'Active' : 'Archived'}
                    </Button>
                    <Button
                        id="refresh-all"
                        variant="contained"
                        color="primary"
                        onClick={() => messagesStore.refreshByApp(appId, archivedView)}
                        style={{marginRight: 5}}>
                        Refresh
                    </Button>
                    {!archivedView && (
                        <Button
                            id="archive-all"
                            variant="contained"
                            disabled={!hasMessages}
                            color="primary"
                            onClick={() => void messagesStore.archiveByApp(appId)}
                            style={{marginRight: 5}}>
                            Archive All
                        </Button>
                    )}
                    {archivedView && (
                        <Button
                            id="restore-all"
                            variant="contained"
                            disabled={!hasMessages}
                            color="primary"
                            onClick={() => void messagesStore.restoreByApp(appId)}
                            style={{marginRight: 5}}>
                            Restore All
                        </Button>
                    )}
                    {canDeleteAll && (
                        <Button
                            id="delete-all"
                            variant="contained"
                            disabled={!hasMessages}
                            color="primary"
                            onClick={() => setDeleteAll(true)}>
                            Delete All
                        </Button>
                    )}
                </div>
            }>
            {!archivedView && app?.allowMemberPost && canPost && (
                <ChatComposer
                    channelName={app.name}
                    fOnSubmit={(message) =>
                        messagesStore.sendMessage(app.id, message, '', app.defaultPriority)
                    }
                />
            )}

            {!messagesStore.loaded(appId, archivedView) ? <LoadingSpinner /> : renderMessages()}

            {deleteAll && (
                <ConfirmDialog
                    title="Confirm Delete"
                    text={
                        app?.autoAssign
                            ? 'Delete all messages from this Global Channel for everyone?'
                            : 'Delete all messages?'
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
