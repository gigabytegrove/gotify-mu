import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    Alert,
    Button,
    Checkbox,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControlLabel,
    List,
    ListItem,
    ListItemText,
    Stack,
    Switch,
    TextField,
    MenuItem,
    Typography,
} from '@mui/material';
import ExpandMore from '@mui/icons-material/ExpandMore';
import Public from '@mui/icons-material/Public';
import Science from '@mui/icons-material/Science';
import {observer} from 'mobx-react-lite';
import {
    IApplication,
    IApplicationGroupGrant,
    IApplicationMember,
    IChannelRole,
    IUser,
} from '../types';
import {useStores} from '../stores';
import ElevationForm from '../common/ElevationForm';

interface IProps {
    app: IApplication;
    fClose: VoidFunction;
}

const memberStatus = (member: IApplicationMember | undefined, isOwner: boolean) => {
    if (isOwner) return <Chip size="small" label="Owner" />;
    if (!member) return <Chip size="small" variant="outlined" label="Not assigned" />;

    return (
        <Stack direction="row" spacing={0.5} sx={{flexWrap: 'wrap'}} useFlexGap>
            <Chip
                size="small"
                variant="outlined"
                label={
                    member.autoAssigned
                        ? 'Global'
                        : member.role === 'manager'
                          ? 'Manager'
                          : member.role === 'publisher'
                            ? 'Publisher'
                            : member.role === 'read-only'
                              ? 'Read Only'
                              : 'Member'
                }
            />
            {!member.receiveNotifications && (
                <Chip size="small" variant="outlined" label="Muted" />
            )}
        </Stack>
    );
};

const ChannelMembersDialog = observer(({app, fClose}: IProps) => {
    const {appStore, currentUser, elevateStore, groupStore} = useStores();
    const [members, setMembers] = useState<IApplicationMember[]>([]);
    const [users, setUsers] = useState<IUser[]>([]);
    const [loading, setLoading] = useState(false);
    const [autoAssign, setAutoAssignState] = useState(Boolean(app.autoAssign));
    const [allowMemberPost, setAllowMemberPost] = useState(Boolean(app.allowMemberPost));
    const [groupGrants, setGroupGrants] = useState<IApplicationGroupGrant[]>([]);

    const load = useCallback(async () => {
        if (!elevateStore.elevated) return;
        setLoading(true);
        try {
            await groupStore.refresh();
            const [loadedMembers, loadedUsers, loadedGroups] = await Promise.all([
                appStore.getMembers(app.id),
                appStore.getAssignableUsers(app.id),
                appStore.getGroupGrants(app.id),
            ]);
            setMembers(loadedMembers);
            setUsers(loadedUsers);
            setGroupGrants(loadedGroups);
        } finally {
            setLoading(false);
        }
    }, [app.id, appStore, elevateStore.elevated, groupStore]);

    useEffect(() => void load(), [load]);

    const memberIds = useMemo(() => new Set(members.map((member) => member.userId)), [members]);

    const toggleUser = async (user: IUser) => {
        if (user.id === app.ownerId || autoAssign) return;

        if (memberIds.has(user.id)) {
            await appStore.removeMember(app.id, user.id);
        } else {
            await appStore.setMember(app.id, user.id, true);
        }
        await load();
    };

    const setRole = async (
        user: IUser,
        role: Exclude<IChannelRole, 'owner'>
    ) => {
        const member = members.find((item) => item.userId === user.id);
        await appStore.setMember(
            app.id,
            user.id,
            member?.receiveNotifications ?? true,
            role
        );
        await load();
    };

    const setGroupGrant = async (
        groupId: number,
        role: Exclude<IChannelRole, 'owner'>
    ) => {
        await appStore.setGroupGrant(app.id, groupId, role, true);
        await load();
    };

    const toggleAutoAssign = async (enabled: boolean) => {
        await appStore.setAutoAssign(app.id, enabled);
        setAutoAssignState(enabled);
        await load();
    };

    const toggleMemberPosting = async (enabled: boolean) => {
        await appStore.setMemberPosting(app.id, enabled);
        setAllowMemberPost(enabled);
    };

    const transferOwnership = async (user: IUser) => {
        await appStore.transferOwnership(app.id, user.id);
        handleClose();
    };

    const handleClose = () => {
        elevateStore.cleanupOidcElevate();
        fClose();
    };

    return (
        <Dialog open onClose={handleClose} fullWidth maxWidth="md">
            <DialogTitle>Manage Channel · {app.name}</DialogTitle>
            <DialogContent>
                {!elevateStore.elevated ? (
                    <Stack spacing={2} sx={{pt: 1}}>
                        <Typography color="text.secondary">
                            Confirm your identity to manage Channel membership and ownership.
                        </Typography>
                        <ElevationForm />
                    </Stack>
                ) : (
                    <Stack spacing={2} sx={{pt: 1}}>
                        {currentUser.user.admin && (
                            <Stack spacing={1}>
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={autoAssign}
                                            onChange={(event) =>
                                                void toggleAutoAssign(event.target.checked)
                                            }
                                        />
                                    }
                                    label={
                                        <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                            <span>Global Channel</span>
                                            <Chip
                                                size="small"
                                                icon={<Public fontSize="small" />}
                                                label="All users"
                                                variant="outlined"
                                            />
                                        </Stack>
                                    }
                                />
                                {autoAssign && (
                                    <Alert severity="info">
                                        Every current and future user is assigned automatically.
                                        Individual membership cannot be removed while Global is
                                        enabled.
                                    </Alert>
                                )}

                                <Accordion
                                    elevation={0}
                                    disableGutters
                                    sx={{border: 1, borderColor: 'divider'}}>
                                    <AccordionSummary expandIcon={<ExpandMore />}>
                                        <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                            <Typography sx={{fontWeight: 600}}>Advanced</Typography>
                                            <Chip
                                                size="small"
                                                icon={<Science fontSize="small" />}
                                                label="Experimental"
                                                variant="outlined"
                                            />
                                        </Stack>
                                    </AccordionSummary>
                                    <AccordionDetails>
                                        <FormControlLabel
                                            control={
                                                <Switch
                                                    checked={allowMemberPost}
                                                    onChange={(event) =>
                                                        void toggleMemberPosting(
                                                            event.target.checked
                                                        )
                                                    }
                                                />
                                            }
                                            label="Allow assigned members to post"
                                        />
                                        <Typography variant="body2" color="text.secondary">
                                            Experimental. Official Gotify Android clients receive
                                            these messages but do not provide a compose interface.
                                        </Typography>
                                    </AccordionDetails>
                                </Accordion>
                            </Stack>
                        )}

                        <Stack spacing={0.25}>
                            <Typography variant="h6">Membership</Typography>
                            <Typography variant="body2" color="text.secondary">
                                Assignment controls access to this Channel. Notification mute is a
                                separate per-user preference.
                            </Typography>
                        </Stack>

                        <List disablePadding>
                            {users.map((user) => {
                                const member = members.find((item) => item.userId === user.id);
                                const isOwner = user.id === app.ownerId;
                                const assigned = memberIds.has(user.id);

                                return (
                                    <ListItem
                                        key={user.id}
                                        divider
                                        secondaryAction={
                                            <Stack
                                                direction="row"
                                                spacing={1}
                                                sx={{alignItems: 'center'}}> 
                                                {!isOwner && member && (
                                                    <Button
                                                        size="small"
                                                        disabled={loading}
                                                        onClick={() =>
                                                            void transferOwnership(user)
                                                        }>
                                                        Make Owner
                                                    </Button>
                                                )}
                                                {!isOwner && member && !member.autoAssigned && (
                                                    <TextField
                                                        select
                                                        size="small"
                                                        value={
                                                            member.role === 'owner'
                                                                ? 'manager'
                                                                : member.role
                                                        }
                                                        onChange={(event) =>
                                                            void setRole(
                                                                user,
                                                                event.target.value as Exclude<
                                                                    IChannelRole,
                                                                    'owner'
                                                                >
                                                            )
                                                        }
                                                        sx={{minWidth: 120}}>
                                                        <MenuItem value="manager">Manager</MenuItem>
                                                        <MenuItem value="publisher">Publisher</MenuItem>
                                                        <MenuItem value="member">Member</MenuItem>
                                                        <MenuItem value="read-only">Read Only</MenuItem>
                                                    </TextField>
                                                )}
                                                <Checkbox
                                                    edge="end"
                                                    checked={assigned}
                                                    disabled={isOwner || autoAssign || loading}
                                                    onChange={() => void toggleUser(user)}
                                                />
                                            </Stack>
                                        }>
                                        <ListItemText
                                            primary={
                                                <Stack
                                                    direction="row"
                                                    spacing={1}
                                                    useFlexGap
                                                    sx={{alignItems: 'center', flexWrap: 'wrap'}}> 
                                                    <Typography sx={{fontWeight: 600}}>
                                                        {user.name}
                                                    </Typography>
                                                    {user.admin && (
                                                        <Chip
                                                            size="small"
                                                            label="Admin"
                                                            variant="outlined"
                                                        />
                                                    )}
                                                </Stack>
                                            }
                                            secondary={memberStatus(member, isOwner)}
                                        />
                                    </ListItem>
                                );
                            })}
                        </List>

                        {currentUser.user.admin && (
                            <Stack spacing={1.25}>
                                <Stack spacing={0.25}>
                                    <Typography variant="h6">Group Access</Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        Give an entire Group access to this Channel without creating
                                        individual assignments.
                                    </Typography>
                                </Stack>
                                {groupStore.getItems().length === 0 ? (
                                    <Typography color="text.secondary">
                                        No Groups have been created.
                                    </Typography>
                                ) : (
                                    groupStore.getItems().map((group) => {
                                        const grant = groupGrants.find(
                                            (item) => item.groupId === group.id
                                        );
                                        return (
                                            <Stack
                                                key={group.id}
                                                direction={{xs: 'column', sm: 'row'}}
                                                spacing={1}
                                                sx={{
                                                    alignItems: {sm: 'center'},
                                                    justifyContent: 'space-between',
                                                    p: 1,
                                                    border: 1,
                                                    borderColor: 'divider',
                                                    borderRadius: 1,
                                                }}>
                                                <Stack>
                                                    <Typography sx={{fontWeight: 600}}>
                                                        {group.name}
                                                    </Typography>
                                                    <Typography variant="body2" color="text.secondary">
                                                        {group.memberCount} member
                                                        {group.memberCount === 1 ? '' : 's'}
                                                    </Typography>
                                                </Stack>
                                                <Stack direction="row" spacing={1}>
                                                    {grant && (
                                                        <TextField
                                                            select
                                                            size="small"
                                                            value={grant.role}
                                                            onChange={(event) =>
                                                                void setGroupGrant(
                                                                    group.id,
                                                                    event.target.value as Exclude<
                                                                        IChannelRole,
                                                                        'owner'
                                                                    >
                                                                )
                                                            }
                                                            sx={{minWidth: 120}}>
                                                            <MenuItem value="manager">Manager</MenuItem>
                                                            <MenuItem value="publisher">Publisher</MenuItem>
                                                            <MenuItem value="member">Member</MenuItem>
                                                            <MenuItem value="read-only">Read Only</MenuItem>
                                                        </TextField>
                                                    )}
                                                    <Button
                                                        size="small"
                                                        color={grant ? 'error' : 'primary'}
                                                        onClick={() =>
                                                            grant
                                                                ? void appStore
                                                                      .removeGroupGrant(
                                                                          app.id,
                                                                          group.id
                                                                      )
                                                                      .then(load)
                                                                : void setGroupGrant(
                                                                      group.id,
                                                                      'member'
                                                                  )
                                                        }>
                                                        {grant ? 'Remove' : 'Assign'}
                                                    </Button>
                                                </Stack>
                                            </Stack>
                                        );
                                    })
                                )}
                            </Stack>
                        )}
                    </Stack>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={handleClose}>Close</Button>
            </DialogActions>
        </Dialog>
    );
});

export default ChannelMembersDialog;
