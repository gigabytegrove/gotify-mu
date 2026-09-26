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
    MenuItem,
    Stack,
    Switch,
    TextField,
    Typography,
} from '@mui/material';
import ExpandMore from '@mui/icons-material/ExpandMore';
import Public from '@mui/icons-material/Public';
import Science from '@mui/icons-material/Science';
import {observer} from 'mobx-react-lite';
import {
    ChannelRole,
    IApplication,
    IApplicationGroupAssignment,
    IApplicationMember,
    IUser,
    IUserGroup,
} from '../types';
import {useStores} from '../stores';
import ElevationForm from '../common/ElevationForm';

interface IProps {
    app: IApplication;
    fClose: VoidFunction;
}

const roleLabel = (role?: ChannelRole) => {
    switch (role) {
        case 'owner':
            return 'Owner';
        case 'manager':
            return 'Manager';
        case 'publisher':
            return 'Publisher';
        case 'readonly':
            return 'Read Only';
        default:
            return 'Member';
    }
};

const editableRoles: Array<{value: Exclude<ChannelRole, 'owner'>; label: string}> = [
    {value: 'readonly', label: 'Read Only'},
    {value: 'member', label: 'Member'},
    {value: 'publisher', label: 'Publisher'},
    {value: 'manager', label: 'Manager'},
];

const memberStatus = (member: IApplicationMember | undefined, isOwner: boolean) => {
    if (isOwner) return <Chip size="small" label="Owner" />;
    if (!member) return <Chip size="small" variant="outlined" label="Not assigned" />;

    return (
        <Stack direction="row" spacing={0.5} sx={{flexWrap: 'wrap'}} useFlexGap>
            <Chip size="small" variant="outlined" label={roleLabel(member.role)} />
            {member.autoAssigned && <Chip size="small" variant="outlined" label="Global" />}
            {member.groupAssigned && <Chip size="small" variant="outlined" label="Via Group" />}
            {!member.receiveNotifications && (
                <Chip size="small" variant="outlined" label="Muted" />
            )}
        </Stack>
    );
};

const ChannelMembersDialog = observer(({app, fClose}: IProps) => {
    const {appStore, currentUser, elevateStore} = useStores();
    const [members, setMembers] = useState<IApplicationMember[]>([]);
    const [users, setUsers] = useState<IUser[]>([]);
    const [groups, setGroups] = useState<IUserGroup[]>([]);
    const [groupAssignments, setGroupAssignments] = useState<IApplicationGroupAssignment[]>([]);
    const [loading, setLoading] = useState(false);
    const [autoAssign, setAutoAssignState] = useState(Boolean(app.autoAssign));
    const [allowMemberPost, setAllowMemberPost] = useState(Boolean(app.allowMemberPost));

    const load = useCallback(async () => {
        if (!elevateStore.elevated) return;
        setLoading(true);
        try {
            const [loadedMembers, loadedUsers, loadedGroups, loadedGroupAssignments] =
                await Promise.all([
                    appStore.getMembers(app.id),
                    appStore.getAssignableUsers(app.id),
                    appStore.getAssignableGroups(app.id),
                    appStore.getGroupAssignments(app.id),
                ]);
            setMembers(loadedMembers);
            setUsers(loadedUsers);
            setGroups(loadedGroups);
            setGroupAssignments(loadedGroupAssignments);
        } finally {
            setLoading(false);
        }
    }, [app.id, appStore, elevateStore.elevated]);

    useEffect(() => void load(), [load]);

    const memberIds = useMemo(() => new Set(members.map((member) => member.userId)), [members]);

    const toggleUser = async (user: IUser) => {
        if (user.id === app.ownerId || autoAssign) return;

        if (memberIds.has(user.id)) {
            const member = members.find((item) => item.userId === user.id);
            if (member?.groupAssigned && !member.autoAssigned) {
                return;
            }
            await appStore.removeMember(app.id, user.id);
        } else {
            await appStore.setMember(app.id, user.id, true, 'member');
        }
        await load();
    };

    const setRole = async (
        user: IUser,
        role: Exclude<ChannelRole, 'owner'>
    ) => {
        const member = members.find((item) => item.userId === user.id);
        await appStore.setMember(
            app.id,
            user.id,
            member?.receiveNotifications !== false,
            role
        );
        await load();
    };

    const setGroup = async (
        group: IUserGroup,
        role: Exclude<ChannelRole, 'owner'>
    ) => {
        await appStore.setGroupAssignment(app.id, group.id, role, true);
        await load();
    };

    const removeGroup = async (groupId: number) => {
        await appStore.removeGroupAssignment(app.id, groupId);
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

                        <Stack spacing={1}>
                            <Typography variant="h6">Group Access</Typography>
                            <Typography variant="body2" color="text.secondary">
                                Assign a Group once and its members inherit the selected Channel role.
                                Direct user roles can still grant additional access.
                            </Typography>
                            {groups.length === 0 ? (
                                <Typography color="text.secondary">
                                    No Groups are available.
                                </Typography>
                            ) : (
                                <List disablePadding>
                                    {groups.map((group) => {
                                        const assignment = groupAssignments.find(
                                            (item) => item.groupId === group.id
                                        );
                                        return (
                                            <ListItem
                                                key={group.id}
                                                divider
                                                secondaryAction={
                                                    <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                                        <TextField
                                                            select
                                                            size="small"
                                                            label="Role"
                                                            value={assignment?.role || ''}
                                                            sx={{minWidth: 140}}
                                                            onChange={(event) => {
                                                                const value = event.target.value;
                                                                if (!value) {
                                                                    if (assignment) void removeGroup(group.id);
                                                                    return;
                                                                }
                                                                void setGroup(
                                                                    group,
                                                                    value as Exclude<ChannelRole, 'owner'>
                                                                );
                                                            }}>
                                                            <MenuItem value="">Not assigned</MenuItem>
                                                            {editableRoles.map((role) => (
                                                                <MenuItem key={role.value} value={role.value}>
                                                                    {role.label}
                                                                </MenuItem>
                                                            ))}
                                                        </TextField>
                                                    </Stack>
                                                }>
                                                <ListItemText
                                                    primary={group.name}
                                                    secondary={
                                                        (group.description || 'Group') +
                                                        ' · ' +
                                                        group.memberCount +
                                                        ' member' +
                                                        (group.memberCount === 1 ? '' : 's')
                                                    }
                                                />
                                            </ListItem>
                                        );
                                    })}
                                </List>
                            )}
                        </Stack>

                        <Stack spacing={0.25}>
                            <Typography variant="h6">Individual Access</Typography>
                            <Typography variant="body2" color="text.secondary">
                                Set direct access and role overrides for individual users.
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
                                                    <>
                                                        <TextField
                                                            select
                                                            size="small"
                                                            label="Role"
                                                            value={member.role || 'member'}
                                                            sx={{minWidth: 130}}
                                                            disabled={loading}
                                                            onChange={(event) =>
                                                                void setRole(
                                                                    user,
                                                                    event.target.value as Exclude<
                                                                        ChannelRole,
                                                                        'owner'
                                                                    >
                                                                )
                                                            }>
                                                            {editableRoles.map((role) => (
                                                                <MenuItem
                                                                    key={role.value}
                                                                    value={role.value}>
                                                                    {role.label}
                                                                </MenuItem>
                                                            ))}
                                                        </TextField>
                                                        <Button
                                                            size="small"
                                                            disabled={loading}
                                                            onClick={() =>
                                                                void transferOwnership(user)
                                                            }>
                                                            Make Owner
                                                        </Button>
                                                    </>
                                                )}
                                                <Checkbox
                                                    edge="end"
                                                    checked={assigned}
                                                    disabled={
                                                        isOwner ||
                                                        autoAssign ||
                                                        loading ||
                                                        Boolean(member?.groupAssigned && !member?.autoAssigned)
                                                    }
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
