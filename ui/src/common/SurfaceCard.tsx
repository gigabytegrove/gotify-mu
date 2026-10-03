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
            p: {xs: 1.75, sm: 2.25},
            borderRadius: 3.25,
            overflowX: 'auto',
            boxShadow: (theme) =>
                theme.palette.mode === 'dark'
                    ? '0 14px 34px rgba(0,0,0,.12)'
                    : '0 12px 30px rgba(37,56,88,.055)',
        }}>
        {(title || subtitle || action) && (
            <Stack
                direction={{xs: 'column', sm: 'row'}}
                spacing={1.5}
                sx={{
                    mb: 1.8,
                    justifyContent: 'space-between',
                    alignItems: {xs: 'stretch', sm: 'flex-start'},
                }}>
                <Box sx={{minWidth: 0}}>
                    {title && (
                        <Typography variant="h6" sx={{lineHeight: 1.2}}>
                            {title}
                        </Typography>
                    )}
                    {subtitle && (
                        <Typography variant="body2" color="text.secondary" sx={{mt: 0.35}}>
                            {subtitle}
                        </Typography>
                    )}
                </Box>
                {action && <Box sx={{flexShrink: 0}}>{action}</Box>}
            </Stack>
        )}
        {children}
    </Paper>
);

export default SurfaceCard;
