import React, {useState} from 'react';
import {
    Alert,
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
import VpnKey from '@mui/icons-material/VpnKey';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import ElevationForm from '../common/ElevationForm';
import {ThemeKey} from '../layout/theme';
import {useStores} from '../stores';
import * as config from '../config';

interface IProps {
    themeMode: ThemeKey;
    setTheme: (theme: ThemeKey) => void;
}

const Settings = ({themeMode, setTheme}: IProps) => (
    <DefaultPage
        title="Settings"
        description="Account preferences, authentication, and server security information."
        maxWidth={900}>
        <SurfaceCard
            title="Appearance"
            subtitle="Choose how the Gotify MU Web UI is displayed."
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
                    <Typography>Local password authentication</Typography>
                    <Chip
                        size="small"
                        label={config.get('localAuth') ? 'Enabled' : 'Disabled'}
                    />
                </Stack>
                <Stack
                    direction={{xs: 'column', sm: 'row'}}
                    spacing={1}
                    sx={{justifyContent: 'space-between'}}> 
                    <Typography>OIDC authentication</Typography>
                    <Chip size="small" label={config.get('oidc') ? 'Enabled' : 'Disabled'} />
                </Stack>
                <Alert severity="info">
                    MFA/2FA and LDAP/Active Directory authentication are planned security
                    features. They are not enabled by this UI rewrite.
                </Alert>
            </Stack>
        </SurfaceCard>

        <SurfaceCard
            title="Planned Security"
            subtitle="Authentication capabilities tracked for future Gotify MU releases."
            action={<VpnKey color="action" />}>
            <Stack direction="row" spacing={1} sx={{flexWrap: 'wrap'}} useFlexGap>
                <Chip label="TOTP MFA" variant="outlined" />
                <Chip label="Recovery Codes" variant="outlined" />
                <Chip label="WebAuthn / Passkeys" variant="outlined" />
                <Chip label="LDAP / Active Directory" variant="outlined" />
                <Chip label="MFA Enforcement" variant="outlined" />
                <Chip label="Security Audit Log" variant="outlined" />
            </Stack>
            <Typography variant="body2" color="text.secondary" sx={{mt: 2}}>
                These controls are roadmap items only. Existing local and OIDC authentication
                behavior remains unchanged until the corresponding backend support is implemented.
            </Typography>
        </SurfaceCard>

        <SurfaceCard
            title="Change Password"
            subtitle="Update the password used for local authentication."
            action={<Key color="action" />}>
            <ChangePasswordForm />
        </SurfaceCard>
    </DefaultPage>
);

const ChangePasswordForm = () => {
    const [pass, setPass] = useState('');
    const {currentUser, elevateStore} = useStores();
    const localAuthEnabled = config.get('localAuth');

    const submit = () => {
        currentUser.changePassword(pass);
        setPass('');
    };

    if (!localAuthEnabled) {
        return <Typography color="text.secondary">Password login is disabled on this server.</Typography>;
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
