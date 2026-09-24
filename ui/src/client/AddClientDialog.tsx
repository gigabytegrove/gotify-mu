import React, {useState} from 'react';
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
import {NumberField} from '../common/NumberField';

interface IProps {
    fClose: (token: string | null) => void;
    fOnSubmit: (name: string, expiresAfterInactivitySeconds: number) => Promise<string>;
}

const AddClientDialog = ({fClose, fOnSubmit}: IProps) => {
    const [name, setName] = useState('');
    const [expiresAfter, setExpiresAfter] = useState(0);
    const submitEnabled = name.trim().length !== 0;

    const submitAndNext = async () => {
        const token = await fOnSubmit(name.trim(), Math.max(0, expiresAfter));
        fClose(token);
    };

    return (
        <Dialog
            open
            onClose={() => fClose(null)}
            fullWidth
            maxWidth="sm"
            aria-labelledby="create-client-title"
            id="client-dialog">
            <DialogTitle id="create-client-title">Create Client</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Clients authenticate browsers, mobile apps, and API tools as your user account.
                </DialogContentText>
                <Stack spacing={2}>
                    <TextField
                        autoFocus
                        className="name"
                        label="Client name"
                        value={name}
                        onChange={(event) => setName(event.target.value)}
                        fullWidth
                        required
                    />
                    <NumberField
                        className="expires-after"
                        label="Expire after inactivity (seconds)"
                        value={expiresAfter}
                        onChange={setExpiresAfter}
                        fullWidth
                        helperText="Use 0 to never expire automatically."
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={() => fClose(null)}>Cancel</Button>
                <Tooltip title={submitEnabled ? '' : 'Client name is required'}>
                    <span>
                        <Button
                            className="create"
                            disabled={!submitEnabled}
                            onClick={() => void submitAndNext()}
                            variant="contained">
                            Create Client
                        </Button>
                    </span>
                </Tooltip>
            </DialogActions>
        </Dialog>
    );
};

export default AddClientDialog;
