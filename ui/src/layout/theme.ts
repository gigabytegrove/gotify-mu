import {createTheme, PaletteMode} from '@mui/material/styles';

export type ThemeKey = 'dark' | 'light' | 'system';

export const isThemeKey = (value: string | null): value is ThemeKey =>
    value === 'light' || value === 'dark' || value === 'system';

export const createMonitaTheme = (mode: PaletteMode) => {
    const surfaceBorder = mode === 'dark' ? '#29323b' : '#dfe5eb';
    const tableBorder = mode === 'dark' ? '#232b33' : '#e9edf1';

    const theme = createTheme({
        palette: {
            mode,
            primary: {
                main: '#2563EB',
                light: '#3B82F6',
                dark: '#1D4ED8',
                contrastText: '#ffffff',
            },
            info: {
                main: '#06B6D4',
            },
            background:
                mode === 'dark'
                    ? {default: '#0F172A', paper: '#151f32'}
                    : {default: '#f6f8fb', paper: '#ffffff'},
            text:
                mode === 'dark'
                    ? {primary: '#f8fafc', secondary: '#94A3B8'}
                    : {primary: '#0F172A', secondary: '#64748b'},
        },
        shape: {
            borderRadius: 10,
        },
        typography: {
            fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
            h4: {fontWeight: 750, letterSpacing: '-0.02em'},
            h5: {fontWeight: 700, letterSpacing: '-0.015em'},
            h6: {fontWeight: 700, letterSpacing: '-0.01em'},
            button: {textTransform: 'none', fontWeight: 650},
            body2: {lineHeight: 1.5},
        },
    });

    return createTheme(theme, {
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    body: {
                        scrollbarColor:
                            mode === 'dark' ? '#49515a transparent' : '#b7bec6 transparent',
                    },
                },
            },
            MuiButton: {
                defaultProps: {disableElevation: true},
                styleOverrides: {
                    root: {
                        borderRadius: 8,
                        minHeight: 34,
                        paddingInline: 14,
                    },
                    sizeSmall: {
                        minHeight: 30,
                        paddingInline: 10,
                    },
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: {
                        borderRadius: 8,
                    },
                },
            },
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundImage: 'none',
                    },
                    outlined: {
                        borderColor: surfaceBorder,
                    },
                },
            },
            MuiDialog: {
                styleOverrides: {
                    paper: {
                        borderRadius: 14,
                        border: `1px solid ${surfaceBorder}`,
                    },
                },
            },
            MuiTableCell: {
                styleOverrides: {
                    root: {
                        borderBottomColor: tableBorder,
                    },
                    head: {
                        fontWeight: 700,
                        color: theme.palette.text.secondary,
                        fontSize: '0.78rem',
                        textTransform: 'uppercase',
                        letterSpacing: '0.04em',
                    },
                },
            },
            MuiChip: {
                styleOverrides: {
                    root: {
                        fontWeight: 650,
                        height: 24,
                        borderRadius: 7,
                    },
                    sizeSmall: {
                        height: 22,
                        fontSize: '0.72rem',
                    },
                },
            },
            MuiTextField: {
                defaultProps: {
                    size: 'small',
                },
            },
            MuiOutlinedInput: {
                styleOverrides: {
                    root: {
                        borderRadius: 9,
                    },
                },
            },
            MuiTooltip: {
                defaultProps: {
                    arrow: true,
                    enterDelay: 450,
                },
            },
            MuiMenu: {
                defaultProps: {
                    elevation: 8,
                },
                styleOverrides: {
                    paper: {
                        border: `1px solid ${surfaceBorder}`,
                        borderRadius: 10,
                        minWidth: 220,
                    },
                },
            },
        },
    });
};
