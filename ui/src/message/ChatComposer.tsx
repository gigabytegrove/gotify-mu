import AttachFile from '@mui/icons-material/AttachFile';
import Close from '@mui/icons-material/Close';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import React, {useEffect, useRef, useState} from 'react';

const MaxImages = 8;
const MaxImageBytes = 25 * 1024 * 1024;
const MaxTotalImageBytes = 50 * 1024 * 1024;
const AllowedImageTypes = new Set(['image/jpeg', 'image/png', 'image/gif', 'image/webp']);
const imageType = (file: File): string => {
    if (file.type) return file.type.toLowerCase();
    const extension = file.name.split('.').pop()?.toLowerCase();
    if (extension === 'png') return 'image/png';
    if (extension === 'gif') return 'image/gif';
    if (extension === 'webp') return 'image/webp';
    if (extension === 'jpg' || extension === 'jpeg') return 'image/jpeg';
    return '';
};

interface IProps {
    channelName: string;
    fOnSubmit: (message: string, images: File[]) => Promise<void>;
    fOnTyping?: (typing: boolean) => Promise<void> | void;
}

interface SelectedImage {
    file: File;
    preview: string;
}

const ChatComposer = ({channelName, fOnSubmit, fOnTyping}: IProps) => {
    const [message, setMessage] = useState('');
    const [images, setImages] = useState<SelectedImage[]>([]);
    const [imageError, setImageError] = useState('');
    const [sending, setSending] = useState(false);
    const stopTimer = useRef<number | null>(null);
    const lastTypingSentAt = useRef(0);
    const imageInput = useRef<HTMLInputElement | null>(null);
    const imagesRef = useRef<SelectedImage[]>([]);

    useEffect(() => {
        imagesRef.current = images;
    }, [images]);

    const clearStopTimer = () => {
        if (stopTimer.current != null) {
            window.clearTimeout(stopTimer.current);
            stopTimer.current = null;
        }
    };

    const setTyping = (typing: boolean) => {
        if (!fOnTyping) return;
        void Promise.resolve(fOnTyping(typing)).catch(() => {});
        if (!typing) lastTypingSentAt.current = 0;
    };

    const noteInput = (value: string) => {
        setMessage(value);
        clearStopTimer();

        if (value.trim().length === 0) {
            setTyping(false);
            return;
        }

        const now = Date.now();
        if (now - lastTypingSentAt.current > 2200) {
            lastTypingSentAt.current = now;
            setTyping(true);
        }

        stopTimer.current = window.setTimeout(() => setTyping(false), 2600);
    };

    const addImages = (files: FileList | File[] | null) => {
        if (!files) return;

        const incoming = Array.from(files);
        const next: SelectedImage[] = [];
        let error = '';
        let totalBytes = images.reduce((sum, item) => sum + item.file.size, 0);

        for (const file of incoming) {
            if (images.length + next.length >= MaxImages) {
                error = `A Chat message can include at most ${MaxImages} images.`;
                break;
            }
            if (!AllowedImageTypes.has(imageType(file))) {
                error = `${file.name}: only JPEG, PNG, GIF, and WebP images are supported.`;
                continue;
            }
            if (file.size <= 0 || file.size > MaxImageBytes) {
                error = `${file.name}: each image must be 25 MiB or smaller.`;
                continue;
            }
            if (totalBytes + file.size > MaxTotalImageBytes) {
                error = 'Images in one Chat message may total at most 50 MiB.';
                break;
            }
            totalBytes += file.size;
            next.push({file, preview: URL.createObjectURL(file)});
        }

        if (next.length > 0) setImages((current) => [...current, ...next]);
        setImageError(error);
        if (imageInput.current) imageInput.current.value = '';
    };

    const removeImage = (index: number) => {
        setImages((current) => {
            const target = current[index];
            if (target) URL.revokeObjectURL(target.preview);
            return current.filter((_item, itemIndex) => itemIndex !== index);
        });
        setImageError('');
    };

    const clearImages = () => {
        imagesRef.current.forEach((item) => URL.revokeObjectURL(item.preview));
        imagesRef.current = [];
        setImages([]);
    };

    const send = async () => {
        const trimmed = message.trim();
        if ((!trimmed && images.length === 0) || sending) return;

        clearStopTimer();
        setTyping(false);
        setSending(true);
        try {
            await fOnSubmit(
                trimmed,
                images.map((item) => item.file)
            );
            setMessage('');
            setImageError('');
            clearImages();
        } finally {
            setSending(false);
        }
    };

    useEffect(
        () => () => {
            clearStopTimer();
            setTyping(false);
            imagesRef.current.forEach((item) => URL.revokeObjectURL(item.preview));
        },
        []
    );

    return (
        <Paper
            elevation={0}
            variant="outlined"
            onDragOver={(event) => {
                if (event.dataTransfer.types.includes('Files')) event.preventDefault();
            }}
            onDrop={(event) => {
                if (!event.dataTransfer.files.length) return;
                event.preventDefault();
                addImages(event.dataTransfer.files);
            }}
            sx={{padding: 1, marginBottom: 1}}>
            {images.length > 0 && (
                <Stack direction="row" spacing={1} useFlexGap sx={{mb: 1, flexWrap: 'wrap'}}>
                    {images.map((item, index) => (
                        <Box
                            key={item.preview}
                            sx={{
                                position: 'relative',
                                width: 92,
                                height: 92,
                                borderRadius: 1.5,
                                overflow: 'hidden',
                                border: 1,
                                borderColor: 'divider',
                                bgcolor: 'action.hover',
                            }}>
                            <Box
                                component="img"
                                src={item.preview}
                                alt={item.file.name}
                                sx={{width: '100%', height: '100%', objectFit: 'cover'}}
                            />
                            <IconButton
                                size="small"
                                aria-label={`Remove ${item.file.name}`}
                                disabled={sending}
                                onClick={() => removeImage(index)}
                                sx={{
                                    position: 'absolute',
                                    top: 3,
                                    right: 3,
                                    bgcolor: 'background.paper',
                                    '&:hover': {bgcolor: 'background.paper'},
                                }}>
                                <Close fontSize="small" />
                            </IconButton>
                        </Box>
                    ))}
                </Stack>
            )}

            {imageError && (
                <Typography variant="caption" color="error" sx={{display: 'block', mb: 0.75}}>
                    {imageError}
                </Typography>
            )}

            <Stack direction="row" spacing={1} sx={{alignItems: 'flex-end'}}>
                <input
                    ref={imageInput}
                    hidden
                    type="file"
                    accept="image/jpeg,image/png,image/gif,image/webp"
                    multiple
                    onChange={(event) => addImages(event.target.files)}
                />
                <Tooltip title="Add image or GIF">
                    <span>
                        <IconButton
                            aria-label="Add image or GIF"
                            disabled={sending || images.length >= MaxImages}
                            onClick={() => imageInput.current?.click()}>
                            <AttachFile />
                        </IconButton>
                    </span>
                </Tooltip>
                <TextField
                    autoFocus
                    fullWidth
                    multiline
                    maxRows={5}
                    label={`Message #${channelName}`}
                    value={message}
                    onChange={(event) => noteInput(event.target.value)}
                    onPaste={(event) => {
                        const pasted = Array.from(event.clipboardData.files);
                        if (pasted.length > 0) addImages(pasted);
                    }}
                    onBlur={() => {
                        clearStopTimer();
                        setTyping(false);
                    }}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter' && !event.shiftKey) {
                            event.preventDefault();
                            void send();
                        }
                    }}
                />
                <Button
                    variant="contained"
                    disabled={sending || (message.trim().length === 0 && images.length === 0)}
                    onClick={() => void send()}>
                    {sending ? 'Sending…' : 'Send'}
                </Button>
            </Stack>
        </Paper>
    );
};

export default ChatComposer;
