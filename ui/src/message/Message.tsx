import React from 'react';
import {
    Avatar,
    Box,
    Button,
    IconButton,
    Paper,
    Stack,
    Tooltip,
    Typography,
} from '@mui/material';
import {ExpandLess, ExpandMore} from '@mui/icons-material';
import Delete from '@mui/icons-material/Delete';
import Archive from '@mui/icons-material/Archive';
import Unarchive from '@mui/icons-material/Unarchive';
import TimeAgo from 'react-timeago';
import {Markdown} from '../common/Markdown';
import * as config from '../config';
import {IMessageExtras} from '../types';
import {contentType, RenderMode} from './extras';
import {TimeAgoFormatter} from '../common/TimeAgoFormatter';

const PREVIEW_HEIGHT = 360;

interface IProps {
    title: string;
    image?: string;
    date: string;
    content: string;
    priority: number;
    appName: string;
    fDelete?: VoidFunction;
    fArchive?: VoidFunction;
    fRestore?: VoidFunction;
    senderName?: string;
    extras?: IMessageExtras;
    expanded: boolean;
    onExpand: (expand: boolean) => void;
}

const Message = ({
    fDelete,
    fArchive,
    fRestore,
    senderName,
    title,
    date,
    image,
    priority,
    content,
    extras,
    appName,
    onExpand,
    expanded: initialExpanded,
}: IProps) => {
    const contentRef = React.useRef<HTMLDivElement | null>(null);
    const [expanded, setExpanded] = React.useState(initialExpanded);
    const [isOverflowing, setOverflowing] = React.useState(false);

    const refreshOverflowing = React.useCallback(() => {
        const ref = contentRef.current;
        if (!ref) return;
        setOverflowing(ref.scrollHeight > PREVIEW_HEIGHT);
    }, []);

    React.useEffect(() => void onExpand(expanded), [expanded, onExpand]);
    React.useEffect(() => refreshOverflowing(), [content, refreshOverflowing]);

    const renderContent = () => {
        switch (contentType(extras)) {
            case RenderMode.Markdown:
                return <Markdown onImageLoaded={refreshOverflowing}>{content}</Markdown>;
            case RenderMode.Plain:
            default:
                return <Box sx={{whiteSpace: 'pre-wrap'}}>{content}</Box>;
        }
    };

    return (
        <Paper
            className="message"
            variant="outlined"
            sx={{
                p: {xs: 1.5, sm: 2},
                mb: 1.5,
                borderRadius: 3,
                borderLeftWidth: 4,
                borderLeftColor:
                    priority >= 8 ? 'error.main' : priority >= 4 ? 'warning.main' : 'divider',
            }}>
            <Stack spacing={1.5}>
                <Stack direction="row" spacing={1.5} alignItems="flex-start">
                    {image && (
                        <Avatar
                            src={config.get('url') + image}
                            alt={`${appName} logo`}
                            variant="rounded"
                            sx={{width: 42, height: 42}}
                        />
                    )}

                    <Box sx={{flex: 1, minWidth: 0}}>
                        <Typography className="title" variant="h6" sx={{lineHeight: 1.25}}>
                            {title}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                            {senderName ? `${senderName} · ${appName}` : appName}
                            {' · '}
                            <TimeAgo date={date} formatter={TimeAgoFormatter.long} />
                        </Typography>
                    </Box>

                    <Stack direction="row" spacing={0.25}>
                        {fRestore && (
                            <Tooltip title="Restore from Archive">
                                <IconButton onClick={fRestore} size="small">
                                    <Unarchive />
                                </IconButton>
                            </Tooltip>
                        )}
                        {fArchive && (
                            <Tooltip title="Archive for me">
                                <IconButton onClick={fArchive} size="small">
                                    <Archive />
                                </IconButton>
                            </Tooltip>
                        )}
                        {fDelete && (
                            <Tooltip title="Delete">
                                <IconButton onClick={fDelete} className="delete" size="small">
                                    <Delete />
                                </IconButton>
                            </Tooltip>
                        )}
                    </Stack>
                </Stack>

                <Box
                    ref={contentRef}
                    className="content"
                    sx={{
                        maxHeight: expanded ? 'none' : PREVIEW_HEIGHT,
                        overflow: 'hidden',
                        wordBreak: 'break-word',
                        '& p': {my: 0.75},
                        '& p:first-of-type': {mt: 0},
                        '& p:last-of-type': {mb: 0},
                        '& pre': {
                            overflow: 'auto',
                            borderRadius: 2,
                            bgcolor: 'action.hover',
                            p: 1.5,
                        },
                        '& img': {maxWidth: '100%'},
                    }}>
                    {renderContent()}
                </Box>

                {isOverflowing && (
                    <Button
                        onClick={() => setExpanded((current) => !current)}
                        size="small"
                        startIcon={expanded ? <ExpandLess /> : <ExpandMore />}>
                        {expanded ? 'Show less' : 'Show full message'}
                    </Button>
                )}
            </Stack>
        </Paper>
    );
};

export default Message;
