import React, {ChangeEvent, useEffect, useRef, useState} from 'react';
import Grid from '@mui/material/Grid';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Alert from '@mui/material/Alert';
import {
    DndContext,
    closestCenter,
    KeyboardSensor,
    PointerSensor,
    useSensor,
    useSensors,
    DragEndEvent,
} from '@dnd-kit/core';
import {SortableContext, verticalListSortingStrategy} from '@dnd-kit/sortable';

import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import {AddApplicationDialog} from './AddApplicationDialog';
import {UpdateApplicationDialog} from './UpdateApplicationDialog';
import {IApplication} from '../types';
import {useStores} from '../stores';
import {observer} from 'mobx-react-lite';
import {TokenConfirmDialog} from '../common/TokenConfirmDialog';
import ChannelMembersDialog from './ChannelMembersDialog';
import ChannelCard from './ChannelCard';

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
            description="Create notification destinations, control membership, and manage delivery behavior."
            rightControl={
                <Button
                    id="create-app"
                    variant="contained"
                    onClick={() => setCreateDialog(true)}>
                    Create Channel
                </Button>
            }>
            <SurfaceCard
                title="Channel Directory"
                subtitle={
                    apps.length === 0
                        ? 'No Channels are available yet.'
                        : `${apps.length} Channel${apps.length === 1 ? '' : 's'} available to your account`
                }>
                {apps.length === 0 ? (
                    <Stack spacing={2} sx={{alignItems: 'flex-start'}}>
                        <Alert severity="info">
                            Create a Channel to start receiving notifications. Administrators can
                            also make Channels Global so every current and future user is assigned.
                        </Alert>
                        <Button variant="contained" onClick={() => setCreateDialog(true)}>
                            Create your first Channel
                        </Button>
                    </Stack>
                ) : (
                    <>
                        <Typography variant="body2" color="text.secondary" sx={{mb: 2}}>
                            Owners and administrators can drag Channels to change their order.
                            Destructive actions are grouped under each Channel's action menu.
                        </Typography>
                        <DndContext
                            sensors={sensors}
                            collisionDetection={closestCenter}
                            onDragEnd={handleDragEnd}>
                            <SortableContext
                                items={apps.map((app) => app.id)}
                                strategy={verticalListSortingStrategy}>
                                <Grid container spacing={1.5}>
                                    {apps.map((app) => {
                                        const canManage =
                                            currentUser.user.admin ||
                                            app.ownerId === currentUser.user.id;
                                        const canDeleteChannel =
                                            currentUser.user.admin || !app.autoAssign;
                                        const canClearHistory =
                                            currentUser.user.admin ||
                                            (!app.autoAssign &&
                                                app.ownerId === currentUser.user.id);

                                        return (
                                            <Grid key={app.id} size={12}>
                                                <ChannelCard
                                                    app={app}
                                                    canManage={canManage}
                                                    canDeleteChannel={canDeleteChannel}
                                                    canClearHistory={canClearHistory}
                                                    fEdit={() => setToUpdateApp(app)}
                                                    fMembers={() => setToManageMembersApp(app)}
                                                    fToggleNotifications={() =>
                                                        void appStore.setNotifications(
                                                            app.id,
                                                            app.receiveNotifications === false
                                                        )
                                                    }
                                                    fRegenerateToken={() =>
                                                        setToRegenerateTokenApp(app)
                                                    }
                                                    fUpload={() =>
                                                        handleImageUploadClick(app.id)
                                                    }
                                                    fDeleteImage={() =>
                                                        setToDeleteImage(app)
                                                    }
                                                    fClearHistory={() =>
                                                        setToClearHistoryApp(app)
                                                    }
                                                    fDelete={() => setToDeleteApp(app)}
                                                />
                                            </Grid>
                                        );
                                    })}
                                </Grid>
                            </SortableContext>
                        </DndContext>
                    </>
                )}

                <input
                    ref={fileInputRef}
                    type="file"
                    accept=".gif,.png,.jpg,.jpeg"
                    style={{display: 'none'}}
                    onChange={onUploadImage}
                />
            </SurfaceCard>

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
                    text={
                        'Regenerate the token for ' +
                        toRegenerateTokenApp.name +
                        '? The current application token will stop working immediately.'
                    }
                    fClose={() => setToRegenerateTokenApp(undefined)}
                    fOnSubmit={() =>
                        appStore.regenerateToken(toRegenerateTokenApp.id).then((token) => {
                            setToRegenerateTokenApp(undefined);
                            setToShowToken(token);
                        })
                    }
                    requireElevated
                />
            )}

            {toDeleteApp && (
                <ConfirmDialog
                    title="Delete Channel"
                    text={
                        'Delete ' +
                        toDeleteApp.name +
                        '? The Channel and its stored messages will be permanently removed.'
                    }
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
                    text={
                        'Permanently delete every message in ' +
                        toClearHistoryApp.name +
                        ' for all Channel members? This cannot be undone.'
                    }
                    fClose={() => setToClearHistoryApp(undefined)}
                    fOnSubmit={() => appStore.clearHistoryForEveryone(toClearHistoryApp.id)}
                    requireElevated
                />
            )}

            {toDeleteImage && (
                <ConfirmDialog
                    title="Remove Channel Image"
                    text={'Remove the custom image from ' + toDeleteImage.name + '?'}
                    fClose={() => setToDeleteImage(undefined)}
                    fOnSubmit={() => appStore.deleteImage(toDeleteImage.id)}
                />
            )}
        </DefaultPage>
    );
});

export default Applications;
