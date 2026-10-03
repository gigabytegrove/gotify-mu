import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import React from 'react';

interface IProps {
    label: string;
    value: React.ReactNode;
    icon?: React.ReactNode;
    helper?: string;
}

const StatCard = ({label, value, icon, helper}: IProps) => (
    <Paper
        variant="outlined"
        sx={{
            p: 2.25,
            height: '100%',
            borderRadius: 3,
            position: 'relative',
            overflow: 'hidden',
            transition: 'transform 160ms ease, box-shadow 160ms ease, border-color 160ms ease',
            '&:hover': {
                transform: 'translateY(-2px)',
                borderColor: 'primary.light',
                boxShadow: 2,
            },
        }}>
        <Stack
            direction="row"
            spacing={2}
            sx={{justifyContent: 'space-between', alignItems: 'flex-start'}}>
            <Box>
                <Typography variant="body2" color="text.secondary">
                    {label}
                </Typography>
                <Typography variant="h4" sx={{mt: 0.25, lineHeight: 1.1}}>
                    {value}
                </Typography>
                {helper && (
                    <Typography variant="caption" color="text.secondary">
                        {helper}
                    </Typography>
                )}
            </Box>
            {icon && (
                <Box
                    sx={{
                        color: 'primary.main',
                        bgcolor: 'action.hover',
                        width: 42,
                        height: 42,
                        borderRadius: 2.5,
                        display: 'grid',
                        placeItems: 'center',
                    }}>
                    {icon}
                </Box>
            )}
        </Stack>
    </Paper>
);

export default StatCard;
