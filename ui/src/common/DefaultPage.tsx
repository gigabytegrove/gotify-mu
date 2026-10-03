import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import React, {FC} from 'react';

interface IProps {
    title: string;
    description?: string;
    rightControl?: React.ReactNode;
    maxWidth?: number;
}

const DefaultPage: FC<React.PropsWithChildren<IProps>> = ({
    title,
    description,
    rightControl,
    maxWidth = 1320,
    children,
}) => (
    <Box component="main" sx={{width: '100%', maxWidth, mx: 'auto'}}>
        <Stack spacing={2.5}>
            <Box
                sx={{
                    pb: 2.1,
                    borderBottom: 1,
                    borderColor: 'divider',
                }}>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1.5}
                    sx={{
                        alignItems: {xs: 'stretch', sm: 'flex-end'},
                        justifyContent: 'space-between',
                    }}>
                    <Box sx={{minWidth: 0}}>
                        <Typography
                            variant="overline"
                            color="primary.main"
                            sx={{display: 'block', mb: 0.25}}>
                            Monita workspace
                        </Typography>
                        <Typography
                            variant="h4"
                            component="h1"
                            sx={{
                                fontSize: {xs: '1.75rem', sm: '2.15rem'},
                                lineHeight: 1.08,
                            }}>
                            {title}
                        </Typography>
                        {description && (
                            <Typography
                                color="text.secondary"
                                sx={{mt: 0.7, maxWidth: 760, fontSize: '0.94rem'}}>
                                {description}
                            </Typography>
                        )}
                    </Box>
                    {rightControl && <Box sx={{flexShrink: 0}}>{rightControl}</Box>}
                </Stack>
            </Box>
            {children}
        </Stack>
    </Box>
);

export default DefaultPage;
