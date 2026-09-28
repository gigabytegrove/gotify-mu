import React, {useState} from 'react';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    Avatar,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    Divider,
    FormControlLabel,
    Stack,
    Switch,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import ExpandMore from '@mui/icons-material/ExpandMore';
import Public from '@mui/icons-material/Public';
import Forum from '@mui/icons-material/Forum';
import {useStores} from '../stores';
import {NumberField} from '../common/NumberField';

interface IProps {
    fClose: (token: string | null) => void;
    fOnSubmit: (
        name: string,
        description: string,
        defaultPriority: number,
        autoAssign?: boolean,
        allowMemberPost?: boolean,
        channelType?: 'notification' | 'chat',
        image?: File
    ) => Promise<string>;
}

export const AddApplicationDialog = ({fClose, fOnSubmit}: IProps) => {
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const [defaultPriority, setDefaultPriority] = useState(0);
    const [autoAssign, setAutoAssign] = useState(false);
    const [allowMemberPost, setAllowMemberPost] = useState(false);
    const [chatChannel, setChatChannel] = useState(false);
    const [imageFile, setImageFile] = useState<File>();
    const [imagePreview, setImagePreview] = useState('');
    const {currentUser} = useStores();

    const submitEnabled = name.trim().length !== 0;

    React.useEffect(
        () => () => {
            if (imagePreview) URL.revokeObjectURL(imagePreview);
        },
        [imagePreview]
    );

    const selectImage = (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        event.target.value = '';
        if (!file) return;

        if (imagePreview) URL.revokeObjectURL(imagePreview);
        setImageFile(file);
        setImagePreview(URL.createObjectURL(file));
    };

    const clearImage = () => {
        if (imagePreview) URL.revokeObjectURL(imagePreview);
        setImageFile(undefined);
        setImagePreview('');
    };

    const submitAndNext = async () => {
        const token = await fOnSubmit(
            name.trim(),
            description,
            defaultPriority,
            autoAssign,
            allowMemberPost,
            chatChannel ? 'chat' : 'notification',
            imageFile
        );
        fClose(token);
    };

    return (
        <Dialog
            id="app-dialog"
            open
            onClose={() => fClose(null)}
            fullWidth
            maxWidth="sm"
            aria-labelledby="create-channel-title">
            <DialogTitle id="create-channel-title">Create Channel</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{mb: 2}}>
                    Channels are notification destinations. They can stay private, be shared with
                    selected users, or be made Global by an administrator.
                </DialogContentText>

                <Stack spacing={2}>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={2}
                        sx={{alignItems: {xs: 'flex-start', sm: 'center'}}}>
                        <Avatar
                            src={imagePreview || undefined}
                            variant="rounded"
                            sx={{width: 72, height: 72, flexShrink: 0}}>
                            {name.trim().slice(0, 2).toUpperCase()}
                        </Avatar>
                        <Stack spacing={0.75}>
                            <Typography variant="subtitle2">Channel image</Typography>
                            <Stack direction="row" spacing={1} useFlexGap sx={{flexWrap: 'wrap'}}>
                                <Button
                                    className="channel-image-upload"
                                    component="label"
                                    variant="outlined"
                                    size="small">
                                    {imageFile ? 'Change image' : 'Choose image'}
                                    <input
                                        hidden
                                        type="file"
                                        accept=".gif,.png,.jpg,.jpeg"
                                        onChange={selectImage}
                                    />
                                </Button>
                                <Button
                                    className="channel-image-remove"
                                    size="small"
                                    disabled={!imageFile}
                                    onClick={clearImage}>
                                    Remove image
                                </Button>
                            </Stack>
                            <Typography variant="caption" color="text.secondary">
                                Optional. PNG, JPG, JPEG, or GIF.
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

                    {currentUser.user.admin && (
                        <>
                            <Divider />
                            <Stack spacing={0.5}>
                                <Typography variant="subtitle1" sx={{fontWeight: 700}}>
                                    Membership
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Global Channels are automatically assigned to every current and
                                    future user.
                                </Typography>
                            </Stack>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={autoAssign}
                                        onChange={(event) => setAutoAssign(event.target.checked)}
                                    />
                                }
                                label={
                                    <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                        <span>Global Channel</span>
                                        <Chip
                                            size="small"
                                            icon={<Public fontSize="small" />}
                                            label="All users"
                                            variant="outlined"
                                        />
                                    </Stack>
                                }
                            />

                            <Accordion
                                elevation={0}
                                disableGutters
                                sx={{border: 1, borderColor: 'divider'}}>
                                <AccordionSummary expandIcon={<ExpandMore />}>
                                    <Stack direction="row" spacing={1} sx={{alignItems: 'center'}}>
                                        <Typography sx={{fontWeight: 600}}>
                                            Channel behavior
                                        </Typography>
                                        {chatChannel && (
                                            <Chip
                                                size="small"
                                                icon={<Forum fontSize="small" />}
                                                label="Chat"
                                                color="primary"
                                                variant="outlined"
                                            />
                                        )}
                                    </Stack>
                                </AccordionSummary>
                                <AccordionDetails>
                                    <Stack spacing={1.5}>
                                        <FormControlLabel
                                            control={
                                                <Switch
                                                    checked={chatChannel}
                                                    onChange={(event) => {
                                                        const enabled = event.target.checked;
                                                        setChatChannel(enabled);
                                                        setAllowMemberPost(enabled);
                                                    }}
                                                />
                                            }
                                            label="Two-way Chat Channel"
                                        />
                                        <Typography variant="body2" color="text.secondary">
                                            Chat Channels use the conversation interface in Gotify
                                            MU desktop and mobile clients instead of the
                                            notification feed.
                                        </Typography>
                                        {chatChannel && (
                                            <FormControlLabel
                                                control={
                                                    <Switch
                                                        checked={allowMemberPost}
                                                        onChange={(event) =>
                                                            setAllowMemberPost(event.target.checked)
                                                        }
                                                    />
                                                }
                                                label="Allow assigned members to post"
                                            />
                                        )}
                                    </Stack>
                                </AccordionDetails>
                            </Accordion>
                        </>
                    )}
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={() => fClose(null)}>Cancel</Button>
                <Tooltip title={submitEnabled ? '' : 'Channel name is required'}>
                    <span>
                        <Button
                            className="create"
                            disabled={!submitEnabled}
                            onClick={() => void submitAndNext()}
                            variant="contained">
                            Create Channel
                        </Button>
                    </span>
                </Tooltip>
            </DialogActions>
        </Dialog>
    );
};
