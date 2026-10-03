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
            boxShadow: (theme) =>
                theme.palette.mode === 'dark'
                    ? '0 8px 24px rgba(0,0,0,.12)'
                    : '0 8px 24px rgba(25,39,62,.045)',
        }}>
        <Stack direction="row" spacing={2} sx={{alignItems: 'center'}}>
            {icon && (
                <Box
                    sx={{
                        width: 44,
                        height: 44,
                        display: 'grid',
                        placeItems: 'center',
                        flexShrink: 0,
                        borderRadius: 2.5,
                        bgcolor: 'action.selected',
                        color: 'primary.main',
                        '& svg': {fontSize: 22},
                    }}>
                    {icon}
                </Box>
            )}
            <Box sx={{minWidth: 0}}>
                <Typography
                    variant="overline"
                    color="text.secondary"
                    sx={{display: 'block', lineHeight: 1.15, fontSize: '0.66rem'}}>
                    {label}
                </Typography>
                <Typography variant="h4" sx={{mt: 0.3, lineHeight: 1, fontSize: '1.8rem'}}>
                    {value}
                </Typography>
                {helper && (
                    <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{display: 'block', mt: 0.45}}
                        noWrap>
                        {helper}
                    </Typography>
                )}
            </Box>
        </Stack>
    </Paper>
);

export default StatCard;
