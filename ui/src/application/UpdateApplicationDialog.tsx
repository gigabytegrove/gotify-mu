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
    fOnSubmit: (name: string, description: string, defaultPriority: number) => Promise<void>;
    initialName: string;
    initialDescription: string;
    initialDefaultPriority: number;
}

export const UpdateApplicationDialog = ({
    initialName,
    initialDescription,
    initialDefaultPriority,
    fClose,
    fOnSubmit,
}: IProps) => {
    const [name, setName] = useState(initialName);
    const [description, setDescription] = useState(initialDescription);
    const [defaultPriority, setDefaultPriority] = useState(initialDefaultPriority);

    const submitEnabled = name.trim().length !== 0;

    const submitAndClose = async () => {
        await fOnSubmit(name.trim(), description, defaultPriority);
        fClose();
    };

    return (
        <Dialog open onClose={fClose} fullWidth maxWidth="sm">
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
