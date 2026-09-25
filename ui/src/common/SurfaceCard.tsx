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
    <Paper
        variant="outlined"
        sx={{
            p: {xs: 1.75, sm: 2},
            borderRadius: 2.5,
            overflowX: 'auto',
            boxShadow: '0 1px 2px rgba(0,0,0,0.02)',
        }}>
        {(title || subtitle || action) && (
            <Stack
                direction="row"
                spacing={2}
                sx={{mb: 1.5, justifyContent: 'space-between', alignItems: 'flex-start'}}> 
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
