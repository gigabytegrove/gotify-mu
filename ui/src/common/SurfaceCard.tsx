import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Divider from '@mui/material/Divider';
import React from 'react';

interface IProps {
    title?: string;
    subtitle?: string;
    action?: React.ReactNode;
    children: React.ReactNode;
}

const SurfaceCard = ({title, subtitle, action, children}: IProps) => {
    const hasHeader = Boolean(title || subtitle || action);

    return (
        <Paper
            variant="outlined"
            sx={{
                borderRadius: 3,
                overflow: 'hidden',
                boxShadow: (theme) =>
                    theme.palette.mode === 'dark'
                        ? '0 10px 30px rgba(0,0,0,.14)'
                        : '0 8px 28px rgba(25,39,62,.055)',
            }}>
            {hasHeader && (
                <>
                    <Stack
                        direction={{xs: 'column', sm: 'row'}}
                        spacing={1.25}
                        sx={{
                            px: {xs: 2, sm: 2.5},
                            py: 1.8,
                            justifyContent: 'space-between',
                            alignItems: {xs: 'stretch', sm: 'center'},
                            bgcolor: (theme) =>
                                theme.palette.mode === 'dark'
                                    ? 'rgba(255,255,255,.012)'
                                    : '#FBFCFE',
                        }}>
                        <Box sx={{minWidth: 0}}>
                            {title && (
                                <Typography variant="h6" sx={{fontSize: '1rem'}}>
                                    {title}
                                </Typography>
                            )}
                            {subtitle && (
                                <Typography
                                    variant="body2"
                                    color="text.secondary"
                                    sx={{mt: title ? 0.2 : 0}}>
                                    {subtitle}
                                </Typography>
                            )}
                        </Box>
                        {action && <Box sx={{flexShrink: 0}}>{action}</Box>}
                    </Stack>
                    <Divider />
                </>
            )}
            <Box sx={{p: {xs: 2, sm: 2.5}, overflowX: 'auto'}}>{children}</Box>
        </Paper>
    );
};

export default SurfaceCard;
