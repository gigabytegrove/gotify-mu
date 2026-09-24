import {createTheme, PaletteMode} from '@mui/material/styles';

export type ThemeKey = 'dark' | 'light' | 'system';

export const isThemeKey = (value: string | null): value is ThemeKey =>
    value === 'light' || value === 'dark' || value === 'system';

export const createGotifyMuTheme = (mode: PaletteMode) =>
    createTheme({
        palette: {
            mode,
            background:
                mode === 'dark'
                    ? {default: '#101418', paper: '#171c21'}
                    : {default: '#f5f7f9', paper: '#ffffff'},
        },
        shape: {
            borderRadius: 12,
        },
        typography: {
            fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
            h4: {fontWeight: 700},
            h5: {fontWeight: 700},
            h6: {fontWeight: 700},
            button: {textTransform: 'none', fontWeight: 600},
        },
        components: {
            MuiButton: {
                defaultProps: {disableElevation: true},
                styleOverrides: {root: {borderRadius: 9}},
            },
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundImage: 'none',
                    },
                },
            },
            MuiTableCell: {
                styleOverrides: {
                    head: {fontWeight: 700},
                },
            },
            MuiChip: {
                styleOverrides: {
                    root: {fontWeight: 600},
                },
            },
        },
    });
