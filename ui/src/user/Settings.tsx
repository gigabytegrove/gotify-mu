import React, {useState} from 'react';
import {
    Button,
    Chip,
    FormControl,
    InputLabel,
    MenuItem,
    Select,
    Stack,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import DarkMode from '@mui/icons-material/DarkMode';
import Security from '@mui/icons-material/Security';
import Key from '@mui/icons-material/Key';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ElevationForm from '../common/ElevationForm';
import {ThemeKey} from '../layout/theme';
import {useStores} from '../stores';
import * as config from '../config';
import {UpdateStatusCard} from '../update/UpdateStatus';

interface IProps {
    themeMode: ThemeKey;
    setTheme: (theme: ThemeKey) => void;
}

const Settings = ({themeMode, setTheme}: IProps) => {
    const {currentUser} = useStores();

    return (
    <DefaultPage
        title="Settings"
        description="Account preferences and sign-in settings."
        maxWidth={900}>
        {currentUser.user.admin && <UpdateStatusCard />}

        <SurfaceCard
            title="Appearance"
            subtitle="Choose how Gotify MU looks on this device."
            action={<DarkMode color="action" />}>
            <FormControl fullWidth>
                <InputLabel id="theme-select-label">Theme</InputLabel>
                <Select
                    labelId="theme-select-label"
                    className="theme-select"
                    label="Theme"
                    value={themeMode}
                    onChange={(e) => setTheme(e.target.value as ThemeKey)}>
                    <MenuItem value="light">Light</MenuItem>
                    <MenuItem value="dark">Dark</MenuItem>
                    <MenuItem value="system">System</MenuItem>
                </Select>
            </FormControl>
        </SurfaceCard>

        <SurfaceCard
            title="Account Security"
            subtitle="Security controls for your local Gotify MU account."
            action={<Security color="action" />}>
            <Stack spacing={2}>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{justifyContent: 'space-between'}}> 
                    <Typography>Password sign-in</Typography>
                    <Chip
                        size="small"
                        label={config.get('localAuth') ? 'Enabled' : 'Disabled'}
                    />
                </Stack>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{justifyContent: 'space-between'}}> 
                    <Typography>Single sign-on</Typography>
                    <Chip size="small" label={config.get('oidc') ? 'Enabled' : 'Disabled'} />
                </Stack>
            </Stack>
        </SurfaceCard>

        <SurfaceCard
            title="Change Password"
            subtitle="Choose a new password for your account."
            action={<Key color="action" />}>
            <ChangePasswordForm />
        </SurfaceCard>
    </DefaultPage>
    );
};

const ChangePasswordForm = () => {
    const [pass, setPass] = useState('');
    const {currentUser, elevateStore} = useStores();
    const localAuthEnabled = config.get('localAuth');

    const submit = () => {
        currentUser.changePassword(pass);
        setPass('');
    };

    if (!localAuthEnabled) {
        return <Typography color="text.secondary">Password sign-in is disabled on this server.</Typography>;
    }

    if (!elevateStore.elevated) {
        return <ElevationForm />;
    }

    return (
        <form
            id="changepw-form"
            onSubmit={(e) => {
                e.preventDefault();
                submit();
            }}>
            <Stack spacing={2}>
                <TextField
                    className="newpass"
                    type="password"
                    label="New Password"
                    value={pass}
                    disabled={!localAuthEnabled}
                    onChange={(e) => setPass(e.target.value)}
                    fullWidth
                />
                <Tooltip title={pass.length !== 0 ? '' : 'Password is required'}>
                    <span>
                        <Button
                            className="change"
                            type="submit"
                            disabled={!localAuthEnabled || pass.length === 0}
                            variant="contained">
                            Change Password
                        </Button>
                    </span>
                </Tooltip>
            </Stack>
        </form>
    );
};

export default Settings;
