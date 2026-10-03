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
        <Stack spacing={{xs: 2.25, md: 3}}>
            <Stack
                direction={{xs: 'column', sm: 'row'}}
                spacing={2}
                sx={{
                    alignItems: {xs: 'stretch', sm: 'flex-end'},
                    justifyContent: 'space-between',
                    pb: 0.25,
                }}>
                <Box sx={{minWidth: 0}}>
                    <Typography
                        variant="overline"
                        sx={{display: 'block', color: 'primary.main', mb: 0.25}}>
                        Monita workspace
                    </Typography>
                    <Typography
                        variant="h4"
                        component="h1"
                        sx={{
                            fontSize: {xs: '1.8rem', sm: '2.15rem'},
                            lineHeight: 1.08,
                        }}>
                        {title}
                    </Typography>
                    {description && (
                        <Typography
                            color="text.secondary"
                            sx={{mt: 0.75, maxWidth: 760, fontSize: '0.94rem'}}>
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
