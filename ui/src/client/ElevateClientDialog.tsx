import React, {useState} from 'react';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Typography from '@mui/material/Typography';
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
    {label: '4 hours', seconds: 4 * 60 * 60},
    {label: '8 hours', seconds: 8 * 60 * 60},
    {label: '24 hours', seconds: 24 * 60 * 60},
];

const ElevateClientDialog = observer(({clientName, clientId, fClose}: IProps) => {
    const {elevateStore, clientStore, currentUser} = useStores();
    const [durationSeconds, setDurationSeconds] = useState(durationOptions[2].seconds);

    const needsElevation = !elevateStore.elevated;

    const handleConfirm = async () => {
        await clientStore.elevate(clientId, durationSeconds);
        if (clientId === currentUser.user.clientId) {
            currentUser.tryAuthenticate();
        }
        fClose();
    };

    const handleClose = () => {
        elevateStore.cleanupOidcElevate();
        fClose();
    };

    return (
        <Dialog open={true} onClose={handleClose} className="elevate-client-dialog">
            <DialogTitle>Elevate Client: {clientName}</DialogTitle>
            <DialogContent>
                {needsElevation ? (
                    <ElevationForm />
                ) : (
                    <>
                        <Typography variant="body2" color="text.secondary" sx={{mb: 1}}>
                            The server security policy caps how long elevated access can remain active.
                        </Typography>
                        <FormControl fullWidth style={{marginTop: 8}}>
                            <InputLabel id="elevate-duration-label">Duration</InputLabel>
                            <Select
                                className="elevate-duration"
                                labelId="elevate-duration-label"
                                label="Duration"
                                value={durationSeconds}
                                onChange={(e) => setDurationSeconds(e.target.value as number)}>
                                {durationOptions.map((opt) => (
                                    <MenuItem key={opt.seconds} value={opt.seconds}>
                                        {opt.label}
                                    </MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                    </>
                )}
            </DialogContent>
            <DialogActions>
                <Button className="elevate-cancel" onClick={handleClose}>
                    Cancel
                </Button>
                {!needsElevation && (
                    <Button
                        className="elevate-confirm"
                        onClick={handleConfirm}
                        autoFocus
                        color="primary"
                        variant="contained">
                        Elevate
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
});

export default ElevateClientDialog;
