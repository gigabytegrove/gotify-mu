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
            p: 2,
            height: '100%',
            borderRadius: 2.5,
            transition: 'transform 140ms ease, box-shadow 140ms ease',
            '&:hover': {
                transform: 'translateY(-1px)',
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
            {icon && <Box sx={{color: 'text.secondary'}}>{icon}</Box>}
        </Stack>
    </Paper>
);

export default StatCard;
