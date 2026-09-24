import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import React from 'react';

interface IProps {
    title?: string;
    subtitle?: string;
    action?: React.ReactNode;
    children: React.ReactNode;
}

const SurfaceCard = ({title, subtitle, action, children}: IProps) => (
    <Paper variant="outlined" sx={{p: 2.5, borderRadius: 3, overflowX: 'auto'}}>
        {(title || subtitle || action) && (
            <Stack
                direction="row"
                justifyContent="space-between"
                alignItems="flex-start"
                spacing={2}
                sx={{mb: 2}}>
                <Box>
                    {title && <Typography variant="h6">{title}</Typography>}
                    {subtitle && (
                        <Typography variant="body2" color="text.secondary">
                            {subtitle}
                        </Typography>
                    )}
                </Box>
                {action}
            </Stack>
        )}
        {children}
    </Paper>
);

export default SurfaceCard;
