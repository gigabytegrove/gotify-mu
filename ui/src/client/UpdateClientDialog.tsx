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
    fClose: VoidFunction;
    fOnSubmit: (name: string, expiresAfterInactivitySeconds: number) => Promise<void>;
    initialName: string;
    initialExpiresAfterInactivitySeconds: number;
}

const UpdateClientDialog = ({
    fClose,
    fOnSubmit,
    initialName,
    initialExpiresAfterInactivitySeconds,
}: IProps) => {
    const [name, setName] = useState(initialName);
    const [expiresAfter, setExpiresAfter] = useState(initialExpiresAfterInactivitySeconds);

    const submitEnabled = name.trim().length !== 0;

    const submitAndClose = async () => {
        await fOnSubmit(name.trim(), Math.max(0, expiresAfter));
        fClose();
    };

    return (
        <Dialog open onClose={fClose} fullWidth maxWidth="sm" id="client-dialog">
            <DialogTitle>Edit Client</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Rename this client or change when it expires after inactivity.
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
                <Button onClick={fClose}>Cancel</Button>
                <Tooltip title={submitEnabled ? '' : 'Client name is required'}>
                    <span>
                        <Button
                            className="update"
                            disabled={!submitEnabled}
                            onClick={() => void submitAndClose()}
                            variant="contained">
                            Save Changes
                        </Button>
                    </span>
                </Tooltip>
            </DialogActions>
        </Dialog>
    );
};

export default UpdateClientDialog;
