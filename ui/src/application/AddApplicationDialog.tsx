import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import FormControlLabel from '@mui/material/FormControlLabel';
import Switch from '@mui/material/Switch';
import {useStores} from '../stores';
import {NumberField} from '../common/NumberField';
import React, {useState} from 'react';

interface IProps {
    fClose: (token: string | null) => void;
    fOnSubmit: (
        name: string,
        description: string,
        defaultPriority: number,
        autoAssign?: boolean
    ) => Promise<string>;
}

export const AddApplicationDialog = ({fClose, fOnSubmit}: IProps) => {
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const [defaultPriority, setDefaultPriority] = useState(0);
    const [autoAssign, setAutoAssign] = useState(false);
    const {currentUser} = useStores();

    const submitEnabled = name.length !== 0;
    const submitAndNext = async () => {
        const token = await fOnSubmit(name, description, defaultPriority, autoAssign);
        fClose(token);
    };

    return (
        <Dialog
            open={true}
            onClose={() => fClose(null)}
            aria-labelledby="form-dialog-title"
            id="app-dialog">
            <DialogTitle id="form-dialog-title">Create a channel</DialogTitle>
            <DialogContent>
                <DialogContentText>
                    A channel receives messages and can be shared with multiple users.
                </DialogContentText>
                <TextField
                    autoFocus
                    margin="dense"
                    className="name"
                    label="Name *"
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    fullWidth
                />
                <TextField
                    margin="dense"
                    className="description"
                    label="Short Description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    fullWidth
                    multiline
                />
                <NumberField
                    margin="dense"
                    className="priority"
                    label="Default Priority"
                    value={defaultPriority}
                    onChange={(value) => setDefaultPriority(value)}
                    fullWidth
                />
                {currentUser.user.admin && (
                    <FormControlLabel
                        control={
                            <Switch
                                checked={autoAssign}
                                onChange={(event) => setAutoAssign(event.target.checked)}
                            />
                        }
                        label="Automatically assign this channel to all users"
                    />
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={() => fClose(null)}>Cancel</Button>
                <Tooltip title={submitEnabled ? '' : 'name is required'}>
                    <div>
                        <Button
                            className="create"
                            disabled={!submitEnabled}
                            onClick={submitAndNext}
                            color="primary"
                            variant="contained">
                            Create
                        </Button>
                    </div>
                </Tooltip>
            </DialogActions>
        </Dialog>
    );
};
