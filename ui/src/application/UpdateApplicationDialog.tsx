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
    fOnSubmit: (name: string, description: string, defaultPriority: number, retentionDays: number) => Promise<void>;
    initialName: string;
    initialDescription: string;
    initialDefaultPriority: number;
    initialRetentionDays: number;
}

export const UpdateApplicationDialog = ({
    initialName,
    initialDescription,
    initialDefaultPriority,
    initialRetentionDays,
    fClose,
    fOnSubmit,
}: IProps) => {
    const [name, setName] = useState(initialName);
    const [description, setDescription] = useState(initialDescription);
    const [defaultPriority, setDefaultPriority] = useState(initialDefaultPriority);
    const [retentionDays, setRetentionDays] = useState(initialRetentionDays);

    const submitEnabled = name.trim().length !== 0;

    const submitAndClose = async () => {
        await fOnSubmit(name.trim(), description, defaultPriority, retentionDays);
        fClose();
    };

    return (
        <Dialog id="app-dialog" open onClose={fClose} fullWidth maxWidth="sm">
            <DialogTitle>Edit Channel</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Update the Channel identity and default delivery priority.
                </DialogContentText>
                <Stack spacing={2}>
                    <TextField
                        autoFocus
                        className="name"
                        label="Channel name"
                        value={name}
                        onChange={(event) => setName(event.target.value)}
                        fullWidth
                        required
                    />
                    <TextField
                        className="description"
                        label="Description"
                        value={description}
                        onChange={(event) => setDescription(event.target.value)}
                        fullWidth
                        multiline
                        minRows={2}
                    />
                    <NumberField
                        className="priority"
                        label="Default priority"
                        value={defaultPriority}
                        onChange={setDefaultPriority}
                        fullWidth
                    />
                    <TextField
                        type="number"
                        label="Message retention"
                        value={retentionDays}
                        onChange={(event) =>
                            setRetentionDays(Math.max(0, Number(event.target.value)))
                        }
                        helperText="Days to keep Channel message history. Use 0 to keep messages indefinitely."
                        slotProps={{htmlInput: {min: 0, max: 36500}}}
                        fullWidth
                    />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={fClose}>Cancel</Button>
                <Tooltip title={submitEnabled ? '' : 'Channel name is required'}>
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
