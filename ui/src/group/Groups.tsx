import React from 'react';
import {
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    IconButton,
    InputAdornment,
    InputLabel,
    MenuItem,
    Select,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import Add from '@mui/icons-material/Add';
import Delete from '@mui/icons-material/Delete';
import Edit from '@mui/icons-material/Edit';
import Group from '@mui/icons-material/Group';
import PersonAdd from '@mui/icons-material/PersonAdd';
import Search from '@mui/icons-material/Search';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ConfirmDialog from '../common/ConfirmDialog';
import {observer} from 'mobx-react-lite';
import {IUserGroup, IUserGroupMember} from '../types';
import {useStores} from '../stores';
import {formatDate} from '../common/TimeAgoFormatter';

const Groups = observer(() => {
    const {groupStore} = useStores();
    const [query, setQuery] = React.useState('');
    const [editing, setEditing] = React.useState<IUserGroup>();
    const [createOpen, setCreateOpen] = React.useState(false);
    const [membersGroup, setMembersGroup] = React.useState<IUserGroup>();
    const [deleting, setDeleting] = React.useState<IUserGroup>();

    React.useEffect(() => void groupStore.refresh(), [groupStore]);

    const groups = groupStore.getItems();
    const normalizedQuery = query.trim().toLowerCase();
    const filtered = normalizedQuery
        ? groups.filter(
              (group) =>
                  group.name.toLowerCase().includes(normalizedQuery) ||
                  group.description.toLowerCase().includes(normalizedQuery)
          )
        : groups;

    return (
        <DefaultPage
            title="Groups"
            description="Organize users for shared administration, Channel assignment, and future policy rules."
            rightControl={
                <Button
                    variant="contained"
                    startIcon={<Add />}
                    onClick={() => setCreateOpen(true)}>
                    Create Group
                </Button>
            }>
            <SurfaceCard
                title="User Groups"
                subtitle={`${groups.length} group${groups.length === 1 ? '' : 's'}`}>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{mb: 1.5, alignItems: {sm: 'center'}, justifyContent: 'space-between'}}>
                    <TextField
                        value={query}
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder="Search groups"
                        aria-label="Search groups"
                        sx={{width: {xs: '100%', sm: 340}}}
                        slotProps={{
                            input: {
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <Search fontSize="small" />
                                    </InputAdornment>
                                ),
                            },
                        }}
                    />
                    <Typography variant="caption" color="text.secondary">
                        {filtered.length} shown
                    </Typography>
                </Stack>

                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Name</TableCell>
                            <TableCell>Description</TableCell>
                            <TableCell>Members</TableCell>
                            <TableCell>Created</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {filtered.map((group) => (
                            <TableRow key={group.id} hover>
                                <TableCell>
                                    <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                        <Group fontSize="small" color="action" />
                                        <Typography sx={{fontWeight: 700}}>{group.name}</Typography>
                                    </Stack>
                                </TableCell>
                                <TableCell>{group.description || '—'}</TableCell>
                                <TableCell>
                                    <Chip size="small" label={group.memberCount} />
                                </TableCell>
                                <TableCell>{formatDate(group.createdAt)}</TableCell>
                                <TableCell align="right">
                                    <Tooltip title="Manage members">
                                        <IconButton
                                            size="small"
                                            onClick={() => setMembersGroup(group)}>
                                            <PersonAdd fontSize="small" />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title="Edit group">
                                        <IconButton size="small" onClick={() => setEditing(group)}>
                                            <Edit fontSize="small" />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title="Delete group">
                                        <IconButton size="small" onClick={() => setDeleting(group)}>
                                            <Delete fontSize="small" />
                                        </IconButton>
                                    </Tooltip>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>

                {filtered.length === 0 && (
                    <Typography color="text.secondary" sx={{py: 3, textAlign: 'center'}}>
                        No groups match your search.
                    </Typography>
                )}
            </SurfaceCard>

            {createOpen && (
                <GroupDialog
                    title="Create Group"
                    fClose={() => setCreateOpen(false)}
                    fSubmit={groupStore.create}
                />
            )}

            {editing && (
                <GroupDialog
                    title={`Edit ${editing.name}`}
                    initialName={editing.name}
                    initialDescription={editing.description}
                    fClose={() => setEditing(undefined)}
                    fSubmit={(name, description) =>
                        groupStore.update(editing.id, name, description)
                    }
                />
            )}

            {membersGroup && (
                <MembersDialog group={membersGroup} fClose={() => setMembersGroup(undefined)} />
            )}

            {deleting && (
                <ConfirmDialog
                    title="Delete Group"
                    text={`Delete ${deleting.name}? User accounts will not be deleted.`}
                    fClose={() => setDeleting(undefined)}
                    fOnSubmit={() => groupStore.remove(deleting.id)}
                />
            )}
        </DefaultPage>
    );
});

const GroupDialog = ({
    title,
    initialName = '',
    initialDescription = '',
    fClose,
    fSubmit,
}: {
    title: string;
    initialName?: string;
    initialDescription?: string;
    fClose: VoidFunction;
    fSubmit: (name: string, description: string) => Promise<void>;
}) => {
    const [name, setName] = React.useState(initialName);
    const [description, setDescription] = React.useState(initialDescription);
    const [saving, setSaving] = React.useState(false);

    const submit = async () => {
        if (!name.trim() || saving) return;
        setSaving(true);
        try {
            await fSubmit(name.trim(), description.trim());
            fClose();
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={fClose} fullWidth maxWidth="sm">
            <DialogTitle>{title}</DialogTitle>
            <DialogContent>
                <Stack spacing={1.5} sx={{pt: 0.5}}>
                    <TextField
                        autoFocus
                        label="Name"
                        value={name}
                        onChange={(event) => setName(event.target.value)}
                        fullWidth
                    />
                    <TextField
                        label="Description"
                        value={description}
                        onChange={(event) => setDescription(event.target.value)}
                        multiline
                        minRows={2}
                        fullWidth
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={fClose}>Cancel</Button>
                <Button
                    variant="contained"
                    disabled={!name.trim() || saving}
                    loading={saving}
                    onClick={() => void submit()}>
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
};

const MembersDialog = observer(
    ({group, fClose}: {group: IUserGroup; fClose: VoidFunction}) => {
        const {groupStore, userStore} = useStores();
        const [members, setMembers] = React.useState<IUserGroupMember[]>([]);
        const [selectedUser, setSelectedUser] = React.useState<number | ''>('');

        const refresh = React.useCallback(async () => {
            const next = await groupStore.getMembers(group.id);
            setMembers(next);
        }, [group.id, groupStore]);

        React.useEffect(() => {
            void refresh();
            void userStore.refresh();
        }, [refresh, userStore]);

        const memberIds = new Set(members.map((member) => member.userId));
        const availableUsers = userStore.getItems().filter((user) => !memberIds.has(user.id));

        return (
            <Dialog open onClose={fClose} fullWidth maxWidth="sm">
                <DialogTitle>{group.name} Members</DialogTitle>
                <DialogContent>
                    <Stack spacing={2} sx={{pt: 0.5}}>
                        <Stack direction="row" spacing={1} sx={{alignItems: 'flex-end'}}>
                            <FormControl fullWidth>
                                <InputLabel id="group-user-select-label">Add user</InputLabel>
                                <Select
                                    labelId="group-user-select-label"
                                    value={selectedUser}
                                    label="Add user"
                                    onChange={(event) =>
                                        setSelectedUser(Number(event.target.value))
                                    }>
                                    {availableUsers.map((user) => (
                                        <MenuItem key={user.id} value={user.id}>
                                            {user.displayName
                                                ? `${user.displayName} (${user.name})`
                                                : user.name}
                                        </MenuItem>
                                    ))}
                                </Select>
                            </FormControl>
                            <Button
                                variant="contained"
                                disabled={selectedUser === ''}
                                onClick={() => {
                                    if (selectedUser === '') return;
                                    void groupStore
                                        .addMember(group.id, selectedUser)
                                        .then(() => refresh())
                                        .then(() => setSelectedUser(''));
                                }}>
                                Add
                            </Button>
                        </Stack>

                        <Stack spacing={0.5}>
                            {members.length === 0 && (
                                <Typography color="text.secondary">
                                    This group has no members yet.
                                </Typography>
                            )}
                            {members.map((member) => (
                                <Stack
                                    key={member.userId}
                                    direction="row"
                                    spacing={1}
                                    sx={{
                                        py: 0.75,
                                        alignItems: 'center',
                                        justifyContent: 'space-between',
                                    }}>
                                    <Stack spacing={0}>
                                        <Typography sx={{fontWeight: 650}}>
                                            {member.displayName || member.name}
                                        </Typography>
                                        {member.displayName && (
                                            <Typography variant="caption" color="text.secondary">
                                                {member.name}
                                            </Typography>
                                        )}
                                    </Stack>
                                    <Button
                                        size="small"
                                        color="error"
                                        onClick={() =>
                                            void groupStore
                                                .removeMember(group.id, member.userId)
                                                .then(() => refresh())
                                        }>
                                        Remove
                                    </Button>
                                </Stack>
                            ))}
                        </Stack>
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button onClick={fClose}>Close</Button>
                </DialogActions>
            </Dialog>
        );
    }
);

export default Groups;
