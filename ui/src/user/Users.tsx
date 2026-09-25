import React from 'react';
import {
    Button,
    Chip,
    IconButton,
    InputAdornment,
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
import Delete from '@mui/icons-material/Delete';
import Edit from '@mui/icons-material/Edit';
import Search from '@mui/icons-material/Search';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import AddEditDialog from './AddEditUserDialog';
import {IUser} from '../types';
import {useStores} from '../stores';
import {observer} from 'mobx-react-lite';
import {formatDate} from '../common/TimeAgoFormatter';

interface IRowProps {
    name: string;
    admin: boolean;
    createdAt: string;
    fDelete: VoidFunction;
    fEdit: VoidFunction;
}

const UserRow: React.FC<IRowProps> = ({name, admin, createdAt, fDelete, fEdit}) => (
    <TableRow hover>
        <TableCell>
            <strong>{name}</strong>
        </TableCell>
        <TableCell>
            <Chip
                size="small"
                variant={admin ? 'filled' : 'outlined'}
                label={admin ? 'Administrator' : 'User'}
            />
        </TableCell>
        <TableCell title={createdAt}>{formatDate(createdAt)}</TableCell>
        <TableCell align="right">
            <Tooltip title="Edit user">
                <IconButton onClick={fEdit} className="edit">
                    <Edit />
                </IconButton>
            </Tooltip>
            <Tooltip title="Delete user">
                <IconButton onClick={fDelete} className="delete">
                    <Delete />
                </IconButton>
            </Tooltip>
        </TableCell>
    </TableRow>
);

const Users = observer(() => {
    const [deleteUser, setDeleteUser] = React.useState<IUser>();
    const [editUser, setEditUser] = React.useState<IUser>();
    const [createDialog, setCreateDialog] = React.useState(false);
    const [query, setQuery] = React.useState('');
    const {userStore} = useStores();

    React.useEffect(() => void userStore.refresh(), []);

    const users = userStore.getItems();
    const normalizedQuery = query.trim().toLowerCase();
    const filteredUsers = normalizedQuery
        ? users.filter((user) => user.name.toLowerCase().includes(normalizedQuery))
        : users;
    const adminCount = users.filter((user) => user.admin).length;

    return (
        <DefaultPage
            title="Users"
            description="Manage local accounts and administrative access."
            rightControl={
                <Button
                    id="create-user"
                    variant="contained"
                    onClick={() => setCreateDialog(true)}>
                    Create User
                </Button>
            }>
            <SurfaceCard
                title="Accounts"
                subtitle={`${users.length} local account${users.length === 1 ? '' : 's'} · ${adminCount} administrator${adminCount === 1 ? '' : 's'}`}>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{mb: 1.5, alignItems: {sm: 'center'}, justifyContent: 'space-between'}}>
                    <TextField
                        value={query}
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder="Search users"
                        aria-label="Search users"
                        sx={{width: {xs: '100%', sm: 320}}}
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
                        {filteredUsers.length} shown
                    </Typography>
                </Stack>
                <Table id="user-table">
                    <TableHead>
                        <TableRow>
                            <TableCell>Username</TableCell>
                            <TableCell>Role</TableCell>
                            <TableCell>Created</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {filteredUsers.map((user: IUser) => (
                            <UserRow
                                key={user.id}
                                name={user.name}
                                admin={user.admin}
                                createdAt={user.createdAt}
                                fDelete={() => setDeleteUser(user)}
                                fEdit={() => setEditUser(user)}
                            />
                        ))}
                    </TableBody>
                </Table>
            </SurfaceCard>

            {createDialog && (
                <AddEditDialog fClose={() => setCreateDialog(false)} fOnSubmit={userStore.create} />
            )}
            {editUser && (
                <AddEditDialog
                    fClose={() => setEditUser(undefined)}
                    fOnSubmit={userStore.update.bind(this, editUser.id)}
                    name={editUser.name}
                    admin={editUser.admin}
                    isEdit={true}
                />
            )}
            {deleteUser && (
                <ConfirmDialog
                    title="Delete User"
                    text={`Delete ${deleteUser.name}? This cannot be undone.`}
                    fClose={() => setDeleteUser(undefined)}
                    fOnSubmit={() => userStore.remove(deleteUser.id)}
                />
            )}
        </DefaultPage>
    );
});

export default Users;
