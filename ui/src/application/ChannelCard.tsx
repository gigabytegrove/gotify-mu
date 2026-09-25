import React from 'react';
import {
    Avatar,
    Box,
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
import OpenInNew from '@mui/icons-material/OpenInNew';
import {Link} from 'react-router';
import {useSortable} from '@dnd-kit/sortable';
import {CSS} from '@dnd-kit/utilities';
import {IApplication} from '../types';
import * as config from '../config';
import {formatDate} from '../common/TimeAgoFormatter';

interface IProps {
    app: IApplication;
    canManage: boolean;
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
                p: 2,
                borderRadius: 3,
                opacity: isDragging ? 0.55 : 1,
                transform: CSS.Transform.toString(transform),
                transition,
            }}>
            <Stack direction="row" spacing={2} alignItems="flex-start">
                <Box
                    {...attributes}
                    {...listeners}
                    sx={{
                        display: 'flex',
                        alignItems: 'center',
                        minHeight: 52,
                        cursor: canManage ? 'grab' : 'default',
                        color: 'text.disabled',
                        touchAction: 'none',
                    }}>
                    <DragIndicator />
                </Box>

                <Avatar
                    src={config.get('url') + app.image}
                    variant="rounded"
                    sx={{width: 52, height: 52, flexShrink: 0}}
                />

                <Stack spacing={1} sx={{minWidth: 0, flex: 1}}>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={1}
                        alignItems={{xs: 'flex-start', sm: 'center'}}>
                        <Typography
                            className="channel-name"
                            variant="h6"
                            noWrap
                            sx={{maxWidth: '100%'}}>
                            {app.name}
                        </Typography>
                        <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap>
                            {app.autoAssign && (
                                <Chip size="small" icon={<Public />} label="Global" />
                            )}
                            {app.allowMemberPost && (
                                <Chip size="small" icon={<Forum />} label="Chat" />
                            )}
                            {app.receiveNotifications === false && (
                                <Chip
                                    size="small"
                                    variant="outlined"
                                    icon={<NotificationsOff />}
                                    label="Muted"
                                />
                            )}
                        </Stack>
                    </Stack>

                    <Typography
                        className="channel-description"
                        variant="body2"
                        color="text.secondary">
                        {app.description || 'No description'}
                    </Typography>

                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={{xs: 0.5, sm: 2.5}}
                        color="text.secondary">
                        <Typography variant="caption">
                            Priority {app.defaultPriority}
                        </Typography>
                        <Typography variant="caption">
                            Last used: {app.lastUsed ? formatDate(app.lastUsed) : 'Never'}
                        </Typography>
                    </Stack>
                </Stack>

                <Stack direction="row" spacing={0.5} alignItems="center">
                    <Tooltip
                        title={
                            app.receiveNotifications === false
                                ? 'Enable notifications'
                                : 'Mute notifications'
                        }>
                        <IconButton
                            onClick={fToggleNotifications}
                            className="toggle-notifications"
                            aria-label={
                                app.receiveNotifications === false
                                    ? 'Enable notifications'
                                    : 'Mute notifications'
                            }>
                            {app.receiveNotifications === false ? (
                                <NotificationsOff />
                            ) : (
                                <NotificationsActive />
                            )}
                        </IconButton>
                    </Tooltip>

                    <Tooltip title="Open channel">
                        <IconButton
                            component={Link}
                            to={`/channels/${app.id}`}
                            aria-label="Open channel">
                            <OpenInNew />
                        </IconButton>
                    </Tooltip>

                    <IconButton
                        className="channel-actions"
                        aria-label="Channel actions"
                        onClick={(event) => setAnchorEl(event.currentTarget)}>
                        <MoreVert />
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
