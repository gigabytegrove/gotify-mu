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
    maxWidth = 1280,
    children,
}) => (
    <Box component="main" sx={{width: '100%', maxWidth, mx: 'auto'}}>
        <Stack spacing={3}>
            <Stack
                direction={{xs: 'column', sm: 'row'}}
                spacing={2}
                sx={{
                    alignItems: {xs: 'stretch', sm: 'center'},
                    justifyContent: 'space-between',
                }}>
                <Box sx={{minWidth: 0}}>
                    <Typography variant="h4" component="h1">
                        {title}
                    </Typography>
                    {description && (
                        <Typography color="text.secondary" sx={{mt: 0.5}}>
                            {description}
                        </Typography>
                    )}
                </Box>
                {rightControl && <Box sx={{flexShrink: 0}}>{rightControl}</Box>}
            </Stack>
            {children}
        </Stack>
    </Box>
);

export default DefaultPage;
