import React, {useState} from 'react';
import {
    Alert,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    FormControl,
    InputLabel,
    MenuItem,
    Select,
    Stack,
} from '@mui/material';
import {observer} from 'mobx-react-lite';
import {useStores} from '../stores';
import ElevationForm from '../common/ElevationForm';

interface IProps {
    clientName: string;
    clientId: number;
    fClose: VoidFunction;
}

const durationOptions = [
    {label: 'Cancel elevation', seconds: -1},
    {label: '1 hour', seconds: 60 * 60},
    {label: '1 day', seconds: 24 * 60 * 60},
    {label: '30 days', seconds: 30 * 24 * 60 * 60},
    {label: '1 year', seconds: 365 * 24 * 60 * 60},
];

const ElevateClientDialog = observer(({clientName, clientId, fClose}: IProps) => {
    const {elevateStore, clientStore, currentUser} = useStores();
    const [durationSeconds, setDurationSeconds] = useState(durationOptions[1].seconds);
    const needsElevation = !elevateStore.elevated;

    const handleConfirm = async () => {
        await clientStore.elevate(clientId, durationSeconds);
        if (clientId === currentUser.user.clientId) {
            void currentUser.tryAuthenticate();
        }
        fClose();
    };

    const handleClose = () => {
        elevateStore.cleanupOidcElevate();
        fClose();
    };

    return (
        <Dialog
            open
            onClose={handleClose}
            fullWidth
            maxWidth="sm"
            className="elevate-client-dialog">
            <DialogTitle>Elevate Client · {clientName}</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Elevation temporarily allows this client to perform security-sensitive actions.
                </DialogContentText>

                {needsElevation ? (
                    <ElevationForm />
                ) : (
                    <Stack spacing={2}>
                        <Alert severity="warning">
                            Only elevate trusted clients for as long as necessary.
                        </Alert>
                        <FormControl fullWidth>
                            <InputLabel id="elevate-duration-label">Duration</InputLabel>
                            <Select
                                className="elevate-duration"
                                labelId="elevate-duration-label"
                                label="Duration"
                                value={durationSeconds}
                                onChange={(event) =>
                                    setDurationSeconds(event.target.value as number)
                                }>
                                {durationOptions.map((option) => (
                                    <MenuItem key={option.seconds} value={option.seconds}>
                                        {option.label}
                                    </MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                    </Stack>
                )}
            </DialogContent>
            <DialogActions>
                <Button className="elevate-cancel" onClick={handleClose}>
                    Cancel
                </Button>
                {!needsElevation && (
                    <Button
                        className="elevate-confirm"
                        onClick={() => void handleConfirm()}
                        variant="contained">
                        Apply
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
});

export default ElevateClientDialog;
