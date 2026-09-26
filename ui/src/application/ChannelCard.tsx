import React from 'react';
import {
    Avatar,
    Box,
    Button,
    Chip,
    IconButton,
    ListItemIcon,
    ListItemText,
    Menu,
    MenuItem,
    Paper,
    Stack,
    Tooltip,
    Typography,
} from '@mui/material';
import DragIndicator from '@mui/icons-material/DragIndicator';
import MoreVert from '@mui/icons-material/MoreVert';
import NotificationsActive from '@mui/icons-material/NotificationsActive';
import NotificationsOff from '@mui/icons-material/NotificationsOff';
import People from '@mui/icons-material/People';
import Public from '@mui/icons-material/Public';
import Forum from '@mui/icons-material/Forum';
import Edit from '@mui/icons-material/Edit';
import Key from '@mui/icons-material/Key';
import Delete from '@mui/icons-material/Delete';
import DeleteSweep from '@mui/icons-material/DeleteSweep';
import CloudUpload from '@mui/icons-material/CloudUpload';
import ImageNotSupported from '@mui/icons-material/ImageNotSupported';
import ArrowForward from '@mui/icons-material/ArrowForward';
import Person from '@mui/icons-material/Person';
import {Link} from 'react-router';
import {useSortable} from '@dnd-kit/sortable';
import {CSS} from '@dnd-kit/utilities';
import {IApplication} from '../types';
import * as config from '../config';
import {formatDate} from '../common/TimeAgoFormatter';

interface IProps {
    app: IApplication;
    canManage: boolean;
    isOwner: boolean;
    canDeleteChannel: boolean;
    canClearHistory: boolean;
    fEdit: VoidFunction;
    fMembers: VoidFunction;
    fToggleNotifications: VoidFunction;
    fRegenerateToken: VoidFunction;
    fUpload: VoidFunction;
    fDeleteImage: VoidFunction;
    fClearHistory: VoidFunction;
    fDelete: VoidFunction;
}

const ChannelCard = ({
    app,
    canManage,
    isOwner,
    canDeleteChannel,
    canClearHistory,
    fEdit,
    fMembers,
    fToggleNotifications,
    fRegenerateToken,
    fUpload,
    fDeleteImage,
    fClearHistory,
    fDelete,
}: IProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const isChat =
        app.channelType === 'chat' ||
        (app.channelType == null && Boolean(app.allowMemberPost));
    const {attributes, listeners, setNodeRef, transform, transition, isDragging} = useSortable({
        id: app.id,
        disabled: !canManage,
    });

    const closeMenu = () => setAnchorEl(null);
    const action = (fn: VoidFunction) => {
        closeMenu();
        fn();
    };

    return (
        <Paper
            ref={setNodeRef}
            className="channel-card"
            data-channel-id={app.id}
            variant="outlined"
            sx={{
                p: {xs: 1.5, sm: 1.75},
                borderRadius: 2.5,
                opacity: isDragging ? 0.55 : 1,
                transform: CSS.Transform.toString(transform),
                transition:
                    transition || 'transform 140ms ease, box-shadow 140ms ease, border-color 140ms ease',
                '&:hover': {
                    boxShadow: 2,
                    borderColor: 'action.selected',
                },
            }}>
            <Stack direction="row" spacing={{xs: 1, sm: 1.5}} sx={{alignItems: 'center'}}>
                <Tooltip title={canManage ? 'Drag to reorder' : ''}>
                    <Box
                        {...attributes}
                        {...listeners}
                        sx={{
                            display: {xs: 'none', sm: 'flex'},
                            alignItems: 'center',
                            alignSelf: 'stretch',
                            cursor: canManage ? 'grab' : 'default',
                            color: 'text.disabled',
                            touchAction: 'none',
                            mx: -0.5,
                        }}>
                        <DragIndicator fontSize="small" />
                    </Box>
                </Tooltip>

                <Avatar
                    src={config.get('url') + app.image}
                    variant="rounded"
                    sx={{width: 46, height: 46, flexShrink: 0}}
                />

                <Box sx={{minWidth: 0, flex: 1}}>
                    <Stack
                        direction="row"
                        spacing={0.75}
                        useFlexGap
                        sx={{alignItems: 'center', flexWrap: 'wrap'}}>
                        <Typography
                            className="channel-name"
                            variant="h6"
                            noWrap
                            sx={{fontSize: '1rem', maxWidth: '100%'}}>
                            {app.name}
                        </Typography>
                        {isOwner && (
                            <Chip
                                size="small"
                                variant="outlined"
                                icon={<Person fontSize="small" />}
                                label="Owner"
                            />
                        )}
                        {app.autoAssign && (
                            <Chip size="small" icon={<Public fontSize="small" />} label="Global" />
                        )}
                        {isChat && (
                            <Chip
                                size="small"
                                variant="outlined"
                                icon={<Forum fontSize="small" />}
                                label="Chat"
                            />
                        )}
                        {app.receiveNotifications === false && (
                            <Chip
                                size="small"
                                variant="outlined"
                                icon={<NotificationsOff fontSize="small" />}
                                label="Muted"
                            />
                        )}
                    </Stack>

                    <Typography
                        className="channel-description"
                        variant="body2"
                        color="text.secondary"
                        noWrap
                        sx={{mt: 0.15}}>
                        {app.description || 'No description'}
                    </Typography>

                    <Stack
                        direction="row"
                        spacing={1.5}
                        useFlexGap
                        sx={{mt: 0.4, flexWrap: 'wrap', color: 'text.secondary'}}>
                        <Typography variant="caption">Priority {app.defaultPriority}</Typography>
                        <Typography variant="caption">
                            {app.lastUsed ? `Used ${formatDate(app.lastUsed)}` : 'Never used'}
                        </Typography>
                    </Stack>
                </Box>

                <Stack direction="row" spacing={0.25} sx={{alignItems: 'center', flexShrink: 0}}>
                    <Tooltip
                        title={
                            app.receiveNotifications === false
                                ? 'Enable notifications'
                                : 'Mute notifications'
                        }>
                        <IconButton
                            size="small"
                            onClick={fToggleNotifications}
                            className="toggle-notifications"
                            aria-label={
                                app.receiveNotifications === false
                                    ? 'Enable notifications'
                                    : 'Mute notifications'
                            }>
                            {app.receiveNotifications === false ? (
                                <NotificationsOff fontSize="small" />
                            ) : (
                                <NotificationsActive fontSize="small" />
                            )}
                        </IconButton>
                    </Tooltip>

                    <Button
                        size="small"
                        component={Link}
                        to={`/channels/${app.id}`}
                        endIcon={<ArrowForward fontSize="small" />}
                        sx={{display: {xs: 'none', sm: 'inline-flex'}}}>
                        Open
                    </Button>

                    <IconButton
                        size="small"
                        className="channel-actions"
                        aria-label="Channel actions"
                        onClick={(event) => setAnchorEl(event.currentTarget)}>
                        <MoreVert fontSize="small" />
                    </IconButton>
                </Stack>
            </Stack>

            <Menu
                anchorEl={anchorEl}
                open={Boolean(anchorEl)}
                onClose={closeMenu}
                anchorOrigin={{vertical: 'bottom', horizontal: 'right'}}
                transformOrigin={{vertical: 'top', horizontal: 'right'}}>
                <MenuItem onClick={() => action(fMembers)} disabled={!canManage}>
                    <ListItemIcon>
                        <People fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Members & permissions</ListItemText>
                </MenuItem>
                <MenuItem className="edit" onClick={() => action(fEdit)} disabled={!canManage}>
                    <ListItemIcon>
                        <Edit fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Edit channel</ListItemText>
                </MenuItem>
                <MenuItem onClick={() => action(fUpload)} disabled={!canManage}>
                    <ListItemIcon>
                        <CloudUpload fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Upload image</ListItemText>
                </MenuItem>
                <MenuItem
                    onClick={() => action(fDeleteImage)}
                    disabled={!canManage || app.image === 'static/defaultapp.png'}>
                    <ListItemIcon>
                        <ImageNotSupported fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Remove image</ListItemText>
                </MenuItem>
                <MenuItem
                    className="regenerate-token"
                    onClick={() => action(fRegenerateToken)}
                    disabled={!canManage}>
                    <ListItemIcon>
                        <Key fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Regenerate token</ListItemText>
                </MenuItem>
                <MenuItem onClick={() => action(fClearHistory)} disabled={!canClearHistory}>
                    <ListItemIcon>
                        <DeleteSweep fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Clear history for everyone</ListItemText>
                </MenuItem>
                <MenuItem
                    className="delete"
                    onClick={() => action(fDelete)}
                    disabled={app.internal || !canManage || !canDeleteChannel}>
                    <ListItemIcon>
                        <Delete fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Delete channel</ListItemText>
                </MenuItem>
            </Menu>
        </Paper>
    );
};

export default ChannelCard;
