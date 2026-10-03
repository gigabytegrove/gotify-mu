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
            p: {xs: 2, sm: 2.5},
            borderRadius: 3,
            overflowX: 'auto',
            position: 'relative',
            boxShadow: 'none',
            transition: 'border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease',
            '&:hover': {
                borderColor: 'primary.light',
                boxShadow: 1,
            },
        }}>
        {(title || subtitle || action) && (
            <Stack
                direction="row"
                spacing={2}
                sx={{mb: 2, justifyContent: 'space-between', alignItems: 'flex-start'}}>
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
