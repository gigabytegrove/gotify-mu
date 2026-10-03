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
            p: 2.1,
            height: '100%',
            borderRadius: 3.25,
            position: 'relative',
            overflow: 'hidden',
            transition: 'transform 150ms ease, box-shadow 150ms ease, border-color 150ms ease',
            '&:hover': {
                transform: 'translateY(-2px)',
                borderColor: 'primary.main',
                boxShadow: (theme) =>
                    theme.palette.mode === 'dark'
                        ? '0 14px 30px rgba(0,0,0,.18)'
                        : '0 14px 30px rgba(37,56,88,.08)',
            },
        }}>
        <Stack
            direction="row"
            spacing={1.75}
            sx={{justifyContent: 'space-between', alignItems: 'flex-start'}}>
            <Box sx={{minWidth: 0}}>
                <Typography
                    variant="overline"
                    color="text.secondary"
                    sx={{fontSize: '0.67rem', lineHeight: 1.4}}>
                    {label}
                </Typography>
                <Typography variant="h4" sx={{mt: 0.4, lineHeight: 1}}>
                    {value}
                </Typography>
                {helper && (
                    <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{display: 'block', mt: 0.65}}>
                        {helper}
                    </Typography>
                )}
            </Box>
            {icon && (
                <Box
                    sx={{
                        width: 42,
                        height: 42,
                        borderRadius: 2.5,
                        display: 'grid',
                        placeItems: 'center',
                        color: 'primary.main',
                        bgcolor: 'action.selected',
                        '& svg': {fontSize: 22},
                    }}>
                    {icon}
                </Box>
            )}
        </Stack>
    </Paper>
);
export default StatCard;
