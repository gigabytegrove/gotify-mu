import React from 'react';
import {
    Alert,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    FormControlLabel,
    Stack,
    Switch,
    TextField,
    Tooltip,
} from '@mui/material';

interface IProps {
    name?: string;
    admin?: boolean;
    fClose: VoidFunction;
    fOnSubmit: (name: string, pass: string, admin: boolean) => Promise<void>;
    isEdit?: boolean;
}

const AddEditUserDialog = ({
    fClose,
    fOnSubmit,
    isEdit,
    name: initialName = '',
    admin: initialAdmin = false,
}: IProps) => {
    const [name, setName] = React.useState(initialName);
    const [pass, setPass] = React.useState('');
    const [admin, setAdmin] = React.useState(initialAdmin);

    const namePresent = name.trim().length !== 0;
    const passPresent = pass.length !== 0 || Boolean(isEdit);

    const submitAndClose = async () => {
        await fOnSubmit(name.trim(), pass, admin);
        fClose();
    };

    const disabledReason = !namePresent
        ? 'Username is required'
        : !passPresent
          ? 'Password is required'
          : '';

    return (
        <Dialog
            open
            onClose={fClose}
            fullWidth
            maxWidth="sm"
            aria-labelledby="user-dialog-title"
            id="add-edit-user-dialog">
            <DialogTitle id="user-dialog-title">
                {isEdit ? `Edit User · ${initialName}` : 'Create User'}
            </DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    {isEdit
                        ? 'Update the local account. Leave the password blank to keep the current password.'
                        : 'Create a local Gotify MU account.'}
                </DialogContentText>

                <Stack spacing={2}>
                    <TextField
                        autoFocus
                        className="name"
                        label="Username"
                        value={name}
                        name="username"
                        id="username"
                        autoComplete="username"
                        onChange={(event) => setName(event.target.value)}
                        fullWidth
                        required
                    />
                    <TextField
                        className="password"
                        type="password"
                        value={pass}
                        fullWidth
                        label={isEdit ? 'New password (optional)' : 'Password'}
                        name="password"
                        id="password"
                        autoComplete={isEdit ? 'new-password' : 'new-password'}
                        onChange={(event) => setPass(event.target.value)}
                        required={!isEdit}
                    />

                    <FormControlLabel
                        control={
                            <Switch
                                checked={admin}
                                className="admin-rights"
                                onChange={(event) => setAdmin(event.target.checked)}
                                value="admin"
                            />
                        }
                        label="Administrator"
                    />

                    {admin && (
                        <Alert severity="warning">
                            Administrators can manage users, Global Channels, and other
                            security-sensitive server settings.
                        </Alert>
                    )}
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={fClose}>Cancel</Button>
                <Tooltip title={disabledReason}>
                    <span>
                        <Button
                            className="save-create"
                            disabled={!passPresent || !namePresent}
                            onClick={() => void submitAndClose()}
                            variant="contained">
                            {isEdit ? 'Save Changes' : 'Create User'}
                        </Button>
                    </span>
                </Tooltip>
            </DialogActions>
        </Dialog>
    );
};

export default AddEditUserDialog;
