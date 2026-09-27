import React, {useState} from 'react';
import {
    Avatar,
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
    Typography,
} from '@mui/material';
import {NumberField} from '../common/NumberField';
import * as config from '../config';

interface IProps {
    fClose: VoidFunction;
    fOnSubmit: (
        name: string,
        description: string,
        defaultPriority: number,
        retentionDays: number,
        channelType: 'notification' | 'chat'
    ) => Promise<void>;
    fUploadImage: (file: File) => Promise<string>;
    fDeleteImage: () => Promise<string>;
    initialName: string;
    initialDescription: string;
    initialDefaultPriority: number;
    initialRetentionDays: number;
    initialImage: string;
    initialChannelType?: 'notification' | 'chat';
}

export const UpdateApplicationDialog = ({
    initialName,
    initialDescription,
    initialDefaultPriority,
    initialRetentionDays,
    initialImage,
    initialChannelType = 'notification',
    fClose,
    fOnSubmit,
    fUploadImage,
    fDeleteImage,
}: IProps) => {
    const [name, setName] = useState(initialName);
    const [description, setDescription] = useState(initialDescription);
    const [defaultPriority, setDefaultPriority] = useState(initialDefaultPriority);
    const [retentionDays, setRetentionDays] = useState(initialRetentionDays);
    const [channelType, setChannelType] = useState<'notification' | 'chat'>(initialChannelType);
    const [image, setImage] = useState(initialImage);
    const [imageBusy, setImageBusy] = useState(false);

    const submitEnabled = name.trim().length !== 0;
    const defaultImage = image === 'static/defaultapp.png';

    const submitAndClose = async () => {
        await fOnSubmit(name.trim(), description, defaultPriority, retentionDays, channelType);
        fClose();
    };

    const uploadImage = async (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        event.target.value = '';
        if (!file) return;

        setImageBusy(true);
        try {
            setImage(await fUploadImage(file));
        } finally {
            setImageBusy(false);
        }
    };

    const deleteImage = async () => {
        setImageBusy(true);
        try {
            setImage(await fDeleteImage());
        } finally {
            setImageBusy(false);
        }
    };

    return (
        <Dialog id="app-dialog" open onClose={fClose} fullWidth maxWidth="sm">
            <DialogTitle>Edit Channel</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Update the Channel identity, image, and default delivery behavior.
                </DialogContentText>
                <Stack spacing={2}>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={2}
                        sx={{alignItems: {xs: 'flex-start', sm: 'center'}}}>
                        <Avatar
                            src={config.get('url') + image}
                            variant="rounded"
                            sx={{width: 72, height: 72, flexShrink: 0}}
                        />
                        <Stack spacing={0.75}>
                            <Typography variant="subtitle2">Channel image</Typography>
                            <Stack direction="row" spacing={1} useFlexGap sx={{flexWrap: 'wrap'}}>
                                <Button
                                    className="channel-image-upload"
                                    component="label"
                                    variant="outlined"
                                    size="small"
                                    disabled={imageBusy}>
                                    {defaultImage ? 'Upload image' : 'Change image'}
                                    <input
                                        hidden
                                        type="file"
                                        accept=".gif,.png,.jpg,.jpeg"
                                        onChange={(event) => void uploadImage(event)}
                                    />
                                </Button>
                                <Button
                                    className="channel-image-remove"
                                    size="small"
                                    disabled={imageBusy || defaultImage}
                                    onClick={() => void deleteImage()}>
                                    Remove image
                                </Button>
                            </Stack>
                            <Typography variant="caption" color="text.secondary">
                                PNG, JPG, JPEG, or GIF.
                            </Typography>
                        </Stack>
                    </Stack>

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
                    <FormControlLabel
                        control={
                            <Switch
                                checked={channelType === 'chat'}
                                onChange={(event) =>
                                    setChannelType(event.target.checked ? 'chat' : 'notification')
                                }
                            />
                        }
                        label="Present this Channel as a two-way Chat Channel"
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
