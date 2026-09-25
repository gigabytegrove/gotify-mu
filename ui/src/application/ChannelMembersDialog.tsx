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
    Typography,
} from '@mui/material';
import ExpandMore from '@mui/icons-material/ExpandMore';
import Public from '@mui/icons-material/Public';
import Science from '@mui/icons-material/Science';
import {observer} from 'mobx-react-lite';
import {IApplication, IApplicationMember, IUser} from '../types';
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
                label={member.autoAssigned ? 'Global' : 'Member'}
            />
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
    const [loading, setLoading] = useState(false);
    const [autoAssign, setAutoAssignState] = useState(Boolean(app.autoAssign));
    const [allowMemberPost, setAllowMemberPost] = useState(Boolean(app.allowMemberPost));

    const load = useCallback(async () => {
        if (!elevateStore.elevated) return;
        setLoading(true);
        try {
            const [loadedMembers, loadedUsers] = await Promise.all([
                appStore.getMembers(app.id),
                appStore.getAssignableUsers(app.id),
            ]);
            setMembers(loadedMembers);
            setUsers(loadedUsers);
        } finally {
            setLoading(false);
        }
    }, [app.id, appStore, elevateStore.elevated]);

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
