import React, {useCallback, useEffect, useMemo, useState} from 'react';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import FormControlLabel from '@mui/material/FormControlLabel';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Switch from '@mui/material/Switch';
import Typography from '@mui/material/Typography';
import {observer} from 'mobx-react-lite';
import {IApplication, IApplicationMember, IUser} from '../types';
import {useStores} from '../stores';
import ElevationForm from '../common/ElevationForm';

interface IProps {
    app: IApplication;
    fClose: VoidFunction;
}

const ChannelMembersDialog = observer(({app, fClose}: IProps) => {
    const {appStore, currentUser, elevateStore} = useStores();
    const [members, setMembers] = useState<IApplicationMember[]>([]);
    const [users, setUsers] = useState<IUser[]>([]);
    const [loading, setLoading] = useState(false);

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
        if (user.id === app.ownerId || app.autoAssign) return;
        if (memberIds.has(user.id)) {
            await appStore.removeMember(app.id, user.id);
        } else {
            await appStore.setMember(app.id, user.id, true);
        }
        await load();
    };

    const toggleAutoAssign = async (enabled: boolean) => {
        await appStore.setAutoAssign(app.id, enabled);
        await load();
    };

    const handleClose = () => {
        elevateStore.cleanupOidcElevate();
        fClose();
    };

    return (
        <Dialog open={true} onClose={handleClose} fullWidth maxWidth="sm">
            <DialogTitle>Channel members: {app.name}</DialogTitle>
            <DialogContent>
                {!elevateStore.elevated ? (
                    <ElevationForm />
                ) : (
                    <>
                        {currentUser.user.admin && (
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={app.autoAssign}
                                        onChange={(event) => void toggleAutoAssign(event.target.checked)}
                                    />
                                }
                                label="Automatically assign this channel to all users"
                            />
                        )}
                        {app.autoAssign && (
                            <Typography variant="body2" sx={{mb: 1}}>
                                This channel is assigned to every current and future user.
                            </Typography>
                        )}
                        <List dense>
                            {users.map((user) => {
                                const member = members.find((item) => item.userId === user.id);
                                const isOwner = user.id === app.ownerId;
                                return (
                                    <ListItem
                                        key={user.id}
                                        secondaryAction={
                                            <Checkbox
                                                edge="end"
                                                checked={memberIds.has(user.id)}
                                                disabled={isOwner || app.autoAssign || loading}
                                                onChange={() => void toggleUser(user)}
                                            />
                                        }>
                                        <ListItemText
                                            primary={user.name}
                                            secondary={
                                                isOwner
                                                    ? 'Owner'
                                                    : member?.autoAssigned
                                                      ? 'Automatically assigned'
                                                      : member
                                                        ? 'Assigned'
                                                        : 'Not assigned'
                                            }
                                        />
                                    </ListItem>
                                );
                            })}
                        </List>
                    </>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={handleClose}>Close</Button>
            </DialogActions>
        </Dialog>
    );
});

export default ChannelMembersDialog;
