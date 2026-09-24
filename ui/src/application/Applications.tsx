import React, {ChangeEvent, useEffect, useRef, useState} from 'react';
import {
    Avatar,
    Box,
    Button,
    Chip,
    Grid,
    IconButton,
    Stack,
    Tooltip,
    Typography,
} from '@mui/material';
import Key from '@mui/icons-material/Key';
import Delete from '@mui/icons-material/Delete';
import Edit from '@mui/icons-material/Edit';
import Group from '@mui/icons-material/Group';
import NotificationsActive from '@mui/icons-material/NotificationsActive';
import NotificationsOff from '@mui/icons-material/NotificationsOff';
import DeleteSweep from '@mui/icons-material/DeleteSweep';
import CloudUpload from '@mui/icons-material/CloudUpload';
import ImageNotSupported from '@mui/icons-material/ImageNotSupported';
import DragIndicator from '@mui/icons-material/DragIndicator';
import Public from '@mui/icons-material/Public';
import Forum from '@mui/icons-material/Forum';
import {
    DndContext,
    closestCenter,
    KeyboardSensor,
    PointerSensor,
    useSensor,
    useSensors,
    DragEndEvent,
} from '@dnd-kit/core';
import {SortableContext, useSortable, rectSortingStrategy} from '@dnd-kit/sortable';
import {CSS} from '@dnd-kit/utilities';

import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import {AddApplicationDialog} from './AddApplicationDialog';
import * as config from '../config';
import {UpdateApplicationDialog} from './UpdateApplicationDialog';
import {IApplication} from '../types';
import {LastUsedCell} from '../common/LastUsedCell';
import {formatDate} from '../common/TimeAgoFormatter';
import {useStores} from '../stores';
import {observer} from 'mobx-react-lite';
import {TokenConfirmDialog} from '../common/TokenConfirmDialog';
import ChannelMembersDialog from './ChannelMembersDialog';

const Applications = observer(() => {
    const {appStore, currentUser} = useStores();
    const apps = appStore.getItems();
    const [toDeleteApp, setToDeleteApp] = useState<IApplication>();
    const [toDeleteImage, setToDeleteImage] = useState<IApplication>();
    const [toUpdateApp, setToUpdateApp] = useState<IApplication>();
    const [toRegenerateTokenApp, setToRegenerateTokenApp] = useState<IApplication>();
    const [toShowToken, setToShowToken] = useState('');
    const [createDialog, setCreateDialog] = useState(false);
    const [toManageMembersApp, setToManageMembersApp] = useState<IApplication>();
    const [toClearHistoryApp, setToClearHistoryApp] = useState<IApplication>();

    const fileInputRef = useRef<HTMLInputElement>(null);
    const uploadId = useRef(-1);

    const sensors = useSensors(useSensor(PointerSensor), useSensor(KeyboardSensor));

    useEffect(() => void appStore.refresh(), []);

    const handleImageUploadClick = (id: number) => {
        uploadId.current = id;
        fileInputRef.current?.click();
    };

    const onUploadImage = (event: ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        if (!file) return;
        void appStore.uploadImage(uploadId.current, file);
        event.target.value = '';
    };

    const handleDragEnd = (event: DragEndEvent) => {
        const {active, over} = event;
        if (over && active.id !== over.id) {
            void appStore.reorder(active.id as number, over.id as number);
        }
    };

    return (
        <DefaultPage
            title="Channels"
            description="Create shared notification destinations, control membership, and manage delivery."
            rightControl={
                <Button variant="contained" id="create-app" onClick={() => setCreateDialog(true)}>
                    Create Channel
                </Button>
            }>
            {apps.length === 0 ? (
                <SurfaceCard>
                    <Stack spacing={1} alignItems="flex-start">
                        <Typography variant="h6">No Channels yet</Typography>
                        <Typography color="text.secondary">
                            Create a Channel to start receiving notifications.
                        </Typography>
                        <Button variant="contained" onClick={() => setCreateDialog(true)}>
                            Create Channel
                        </Button>
                    </Stack>
                </SurfaceCard>
            ) : (
                <DndContext
                    sensors={sensors}
                    collisionDetection={closestCenter}
                    onDragEnd={handleDragEnd}>
                    <SortableContext items={apps.map((app) => app.id)} strategy={rectSortingStrategy}>
                        <Grid container spacing={2}>
                            {apps.map((app) => {
                                const canManage =
                                    currentUser.user.admin || app.ownerId === currentUser.user.id;
                                const canDeleteChannel =
                                    currentUser.user.admin || !Boolean(app.autoAssign);
                                const canClearHistory =
                                    currentUser.user.admin ||
                                    (!app.autoAssign && app.ownerId === currentUser.user.id);

                                return (
                                    <Grid key={app.id} size={{xs: 12, lg: 6}}>
                                        <ChannelCard
                                            app={app}
                                            canManage={canManage}
                                            canDeleteChannel={canDeleteChannel}
                                            canClearHistory={canClearHistory}
                                            fOpenMembers={() => setToManageMembersApp(app)}
                                            fEdit={() => setToUpdateApp(app)}
                                            fToken={() => setToRegenerateTokenApp(app)}
                                            fUpload={() => handleImageUploadClick(app.id)}
                                            fDeleteImage={() => setToDeleteImage(app)}
                                            fDelete={() => setToDeleteApp(app)}
                                            fClearHistory={() => setToClearHistoryApp(app)}
                                            fToggleNotifications={() =>
                                                void appStore.setNotifications(
                                                    app.id,
                                                    app.receiveNotifications === false
                                                )
                                            }
                                        />
                                    </Grid>
                                );
                            })}
                        </Grid>
                    </SortableContext>
                </DndContext>
            )}

            <input
                ref={fileInputRef}
                type="file"
                accept=".gif,.png,.jpg,.jpeg"
                style={{display: 'none'}}
                onChange={onUploadImage}
            />

            {toShowToken && (
                <TokenConfirmDialog token={toShowToken} fClose={() => setToShowToken('')} />
            )}

            {createDialog && (
                <AddApplicationDialog
                    fClose={(token) => {
                        setCreateDialog(false);
                        setToShowToken(token ?? '');
                    }}
                    fOnSubmit={appStore.create}
                />
            )}

            {toUpdateApp && (
                <UpdateApplicationDialog
                    fClose={() => setToUpdateApp(undefined)}
                    fOnSubmit={(name, description, defaultPriority) =>
                        appStore.update({...toUpdateApp, name, description, defaultPriority})
                    }
                    initialDescription={toUpdateApp.description}
                    initialName={toUpdateApp.name}
                    initialDefaultPriority={toUpdateApp.defaultPriority}
                />
            )}

            {toRegenerateTokenApp && (
                <ConfirmDialog
                    title="Regenerate Channel Token"
                    text={`Regenerate the publishing token for ${toRegenerateTokenApp.name}? Existing integrations using the current token will stop working.`}
                    fClose={() => setToRegenerateTokenApp(undefined)}
                    fOnSubmit={() =>
                        appStore.regenerateToken(toRegenerateTokenApp.id).then((token) => {
                            setToShowToken(token);
                        })
                    }
                    requireElevated
                />
            )}

            {toDeleteApp && (
                <ConfirmDialog
                    title="Delete Channel"
                    text={`Delete ${toDeleteApp.name}? The Channel and its stored messages will be removed.`}
                    fClose={() => setToDeleteApp(undefined)}
                    fOnSubmit={() => appStore.remove(toDeleteApp.id)}
                    requireElevated
                />
            )}

            {toManageMembersApp && (
                <ChannelMembersDialog
                    app={toManageMembersApp}
                    fClose={() => setToManageMembersApp(undefined)}
                />
            )}

            {toClearHistoryApp && (
                <ConfirmDialog
                    title="Clear History For Everyone"
                    text={`Permanently delete every message in ${toClearHistoryApp.name} for every member? This cannot be undone.`}
                    fClose={() => setToClearHistoryApp(undefined)}
                    fOnSubmit={() => appStore.clearHistoryForEveryone(toClearHistoryApp.id)}
                    requireElevated
                />
            )}

            {toDeleteImage && (
                <ConfirmDialog
                    title="Remove Channel Image"
                    text={`Remove the custom image from ${toDeleteImage.name}?`}
                    fClose={() => setToDeleteImage(undefined)}
                    fOnSubmit={() => appStore.deleteImage(toDeleteImage.id)}
                />
            )}
        </DefaultPage>
    );
});

interface IChannelCardProps {
    app: IApplication;
    canManage: boolean;
    canDeleteChannel: boolean;
    canClearHistory: boolean;
    fOpenMembers: VoidFunction;
    fEdit: VoidFunction;
    fToken: VoidFunction;
    fUpload: VoidFunction;
    fDeleteImage: VoidFunction;
    fDelete: VoidFunction;
    fClearHistory: VoidFunction;
    fToggleNotifications: VoidFunction;
}

const ChannelCard = ({
    app,
    canManage,
    canDeleteChannel,
    canClearHistory,
    fOpenMembers,
    fEdit,
    fToken,
    fUpload,
    fDeleteImage,
    fDelete,
    fClearHistory,
    fToggleNotifications,
}: IChannelCardProps) => {
    const {attributes, listeners, setNodeRef, transform, transition, isDragging} = useSortable({
        id: app.id,
        disabled: !canManage,
    });

    const isDefaultImage = app.image === 'static/defaultapp.png';

    return (
        <Box
            ref={setNodeRef}
            sx={{
                height: '100%',
                transform: CSS.Transform.toString(transform),
                transition,
                opacity: isDragging ? 0.6 : 1,
            }}>
            <SurfaceCard>
                <Stack spacing={2}>
                    <Stack direction="row" spacing={1.5} alignItems="flex-start">
                        <IconButton
                            {...attributes}
                            {...listeners}
                            disabled={!canManage}
                            aria-label="Reorder Channel"
                            sx={{cursor: canManage ? 'grab' : 'default', mt: 0.25}}>
                            <DragIndicator />
                        </IconButton>

                        <Avatar
                            src={config.get('url') + app.image}
                            variant="rounded"
                            sx={{width: 52, height: 52}}
                        />

                        <Box sx={{flex: 1, minWidth: 0}}>
                            <Typography variant="h6" noWrap>
                                {app.name}
                            </Typography>
                            <Typography
                                variant="body2"
                                color="text.secondary"
                                sx={{minHeight: 20}}>
                                {app.description || 'No description'}
                            </Typography>
                            <Stack direction="row" spacing={0.75} flexWrap="wrap" sx={{mt: 1}}>
                                {app.autoAssign && (
                                    <Chip
                                        size="small"
                                        icon={<Public fontSize="small" />}
                                        label="Global"
                                    />
                                )}
                                {app.allowMemberPost && (
                                    <Chip
                                        size="small"
                                        icon={<Forum fontSize="small" />}
                                        label="Chat"
                                        variant="outlined"
                                    />
                                )}
                                {app.receiveNotifications === false && (
                                    <Chip size="small" label="Muted" variant="outlined" />
                                )}
                            </Stack>
                        </Box>

                        <Tooltip
                            title={
                                app.receiveNotifications === false
                                    ? 'Enable notifications'
                                    : 'Mute notifications'
                            }>
                            <IconButton onClick={fToggleNotifications}>
                                {app.receiveNotifications === false ? (
                                    <NotificationsOff />
                                ) : (
                                    <NotificationsActive />
                                )}
                            </IconButton>
                        </Tooltip>
                    </Stack>

                    <Grid container spacing={1.5}>
                        <Grid size={{xs: 4}}>
                            <Typography variant="caption" color="text.secondary">
                                Priority
                            </Typography>
                            <Typography>{app.defaultPriority}</Typography>
                        </Grid>
                        <Grid size={{xs: 4}}>
                            <Typography variant="caption" color="text.secondary">
                                Last Used
                            </Typography>
                            <Box>
                                <LastUsedCell lastUsed={app.lastUsed} />
                            </Box>
                        </Grid>
                        <Grid size={{xs: 4}}>
                            <Typography variant="caption" color="text.secondary">
                                Created
                            </Typography>
                            <Typography>{formatDate(app.createdAt)}</Typography>
                        </Grid>
                    </Grid>

                    <Stack
                        direction="row"
                        spacing={0.75}
                        alignItems="center"
                        flexWrap="wrap"
                        useFlexGap>
                        {canManage && (
                            <Button size="small" startIcon={<Group />} onClick={fOpenMembers}>
                                Members
                            </Button>
                        )}
                        {canManage && (
                            <Button size="small" startIcon={<Edit />} onClick={fEdit}>
                                Edit
                            </Button>
                        )}
                        {canManage && (
                            <Tooltip title="Regenerate publishing token">
                                <IconButton onClick={fToken} size="small">
                                    <Key />
                                </IconButton>
                            </Tooltip>
                        )}
                        {canManage && (
                            <Tooltip title="Upload Channel image">
                                <IconButton onClick={fUpload} size="small">
                                    <CloudUpload />
                                </IconButton>
                            </Tooltip>
                        )}
                        {canManage && !isDefaultImage && (
                            <Tooltip title="Remove custom Channel image">
                                <IconButton onClick={fDeleteImage} size="small">
                                    <ImageNotSupported />
                                </IconButton>
                            </Tooltip>
                        )}
                        {canClearHistory && (
                            <Tooltip title="Clear history for everyone">
                                <IconButton onClick={fClearHistory} size="small">
                                    <DeleteSweep />
                                </IconButton>
                            </Tooltip>
                        )}
                        {canManage && (
                            <Box sx={{flex: 1}} />
                        )}
                        {canManage && (
                            <Tooltip
                                title={
                                    canDeleteChannel
                                        ? 'Delete Channel'
                                        : 'Only an administrator can delete a Global Channel'
                                }>
                                <span>
                                    <IconButton
                                        onClick={fDelete}
                                        disabled={app.internal || !canDeleteChannel}
                                        size="small">
                                        <Delete />
                                    </IconButton>
                                </span>
                            </Tooltip>
                        )}
                    </Stack>
                </Stack>
            </SurfaceCard>
        </Box>
    );
};

export default Applications;
