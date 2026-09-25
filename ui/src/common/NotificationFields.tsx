import React from 'react';
import {Autocomplete, TextField, TextFieldProps} from '@mui/material';

export const priorityLabel = (value: number): string => {
    if (value >= 8) return 'Critical';
    if (value >= 4) return 'High';
    if (value <= -2) return 'Low';
    return 'Normal';
};

export const PriorityField = ({
    value,
    onChange,
    label = 'Priority',
    helperText,
    ...props
}: Omit<TextFieldProps, 'value' | 'onChange' | 'type'> & {
    value: number;
    onChange: (value: number) => void;
}) => (
    <TextField
        {...props}
        type="number"
        label={label}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
        helperText={helperText || priorityLabel(value) + ' · Gotify priorities may use any integer value.'}
    />
);

const fallbackTimezones = [
    'UTC',
    'America/New_York',
    'America/Chicago',
    'America/Denver',
    'America/Los_Angeles',
    'America/Phoenix',
    'America/Anchorage',
    'Pacific/Honolulu',
    'Europe/London',
    'Europe/Paris',
    'Europe/Berlin',
    'Asia/Tokyo',
    'Australia/Sydney',
];

const supportedTimezones = (): string[] => {
    try {
        const intl = Intl as typeof Intl & {supportedValuesOf?: (key: string) => string[]};
        const values = intl.supportedValuesOf?.('timeZone');
        return values && values.length > 0 ? values : fallbackTimezones;
    } catch {
        return fallbackTimezones;
    }
};

export const TimezoneField = ({
    value,
    onChange,
    label = 'Timezone',
}: {
    value: string;
    onChange: (value: string) => void;
    label?: string;
}) => (
    <Autocomplete
        freeSolo
        options={supportedTimezones()}
        value={value}
        onChange={(_event, next) => onChange(next || 'UTC')}
        onInputChange={(_event, next) => onChange(next)}
        renderInput={(params) => (
            <TextField
                {...params}
                label={label}
                helperText="Search by region/city. Custom IANA timezone names are also accepted."
            />
        )}
    />
);

export const TimeOfDayField = ({
    hour,
    minute,
    onChange,
    label = 'Time',
}: {
    hour: number;
    minute: number;
    onChange: (hour: number, minute: number) => void;
    label?: string;
}) => {
    const value =
        String(Math.max(0, Math.min(23, hour))).padStart(2, '0') +
        ':' +
        String(Math.max(0, Math.min(59, minute))).padStart(2, '0');
    return (
        <TextField
            type="time"
            label={label}
            value={value}
            onChange={(event) => {
                const [nextHour, nextMinute] = event.target.value.split(':').map(Number);
                onChange(nextHour || 0, nextMinute || 0);
            }}
            slotProps={{inputLabel: {shrink: true}}}
            fullWidth
        />
    );
};
