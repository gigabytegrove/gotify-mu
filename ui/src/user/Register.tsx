import React from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    Stack,
    TextField,
    Tooltip,
} from '@mui/material';

interface IProps {
    name?: string;
    fClose: VoidFunction;
    fOnSubmit: (name: string, pass: string) => Promise<boolean>;
}

const RegistrationDialog = ({fClose, fOnSubmit, name: initialName = ''}: IProps) => {
    const [name, setName] = React.useState(initialName);
    const [pass, setPass] = React.useState('');

    const namePresent = name.trim().length !== 0;
    const passPresent = pass.length !== 0;

    const submitAndClose = async () => {
        const success = await fOnSubmit(name.trim(), pass);
        if (success) fClose();
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
            aria-labelledby="registration-title"
            id="add-edit-user-dialog">
            <DialogTitle id="registration-title">Create Account</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Register a local account on this Gotify MU server.
                </DialogContentText>
                <Stack spacing={2}>
                    <TextField
                        autoFocus
                        id="register-username"
                        className="name"
                        label="Username"
                        name="username"
                        value={name}
                        autoComplete="username"
                        onChange={(event) => setName(event.target.value)}
                        fullWidth
                        required
                    />
                    <TextField
                        id="register-password"
                        className="password"
                        type="password"
                        value={pass}
                        fullWidth
                        label="Password"
                        name="password"
                        autoComplete="new-password"
                        onChange={(event) => setPass(event.target.value)}
                        required
                    />
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
                            Create Account
                        </Button>
                    </span>
                </Tooltip>
            </DialogActions>
        </Dialog>
    );
};

export default RegistrationDialog;
