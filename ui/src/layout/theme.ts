import {alpha, createTheme, PaletteMode} from '@mui/material/styles';

export type ThemeKey = 'dark' | 'light' | 'system';

export const isThemeKey = (value: string | null): value is ThemeKey =>
    value === 'light' || value === 'dark' || value === 'system';

export const createMonitaTheme = (mode: PaletteMode) => {
    const dark = mode === 'dark';
    const colors = {
        canvas: dark ? '#080D17' : '#F3F6FA',
        paper: dark ? '#0E1626' : '#FFFFFF',
        raised: dark ? '#121D30' : '#F9FBFD',
        border: dark ? '#22304A' : '#DCE4EE',
        borderSoft: dark ? '#18243A' : '#E9EEF4',
        text: dark ? '#F4F7FB' : '#122033',
        muted: dark ? '#92A3BA' : '#617086',
        primary: dark ? '#6E8CFF' : '#3459E6',
        primaryDark: dark ? '#5675EA' : '#2747C7',
        accent: dark ? '#3BC9DB' : '#0A9CB0',
        nav: '#0A1220',
        navRaised: '#111C2F',
    };

    const base = createTheme({
        palette: {
            mode,
            primary: {
                main: colors.primary,
                dark: colors.primaryDark,
                contrastText: '#FFFFFF',
            },
            secondary: {
                main: colors.accent,
            },
            info: {
                main: colors.accent,
            },
            background: {
                default: colors.canvas,
                paper: colors.paper,
            },
            text: {
                primary: colors.text,
                secondary: colors.muted,
            },
            divider: colors.border,
        },
        shape: {
            borderRadius: 12,
        },
        typography: {
            fontFamily:
                '"Segoe UI Variable Text", "Segoe UI", "Inter", "Roboto", "Helvetica", "Arial", sans-serif',
            h1: {fontWeight: 760, letterSpacing: '-0.035em'},
            h2: {fontWeight: 750, letterSpacing: '-0.03em'},
            h3: {fontWeight: 740, letterSpacing: '-0.028em'},
            h4: {fontWeight: 730, letterSpacing: '-0.025em'},
            h5: {fontWeight: 720, letterSpacing: '-0.02em'},
            h6: {fontWeight: 700, letterSpacing: '-0.012em'},
            button: {textTransform: 'none', fontWeight: 680, letterSpacing: '-0.005em'},
            body1: {lineHeight: 1.55},
            body2: {lineHeight: 1.5},
            caption: {lineHeight: 1.45},
            overline: {fontWeight: 750, letterSpacing: '0.09em'},
        },
    });

    return createTheme(base, {
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    'html, body, #root': {
                        minHeight: '100%',
                    },
                    body: {
                        backgroundColor: colors.canvas,
                        scrollbarColor: dark ? '#3A4961 transparent' : '#B8C3D1 transparent',
                    },
                    '::selection': {
                        backgroundColor: alpha(colors.primary, 0.24),
                    },
                    '*': {
                        scrollbarWidth: 'thin',
                    },
                },
            },
            MuiAppBar: {
                styleOverrides: {
                    root: {
                        backgroundImage: 'none',
                    },
                },
            },
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundImage: 'none',
                    },
                    outlined: {
                        borderColor: colors.border,
                    },
                },
            },
            MuiButton: {
                defaultProps: {
                    disableElevation: true,
                },
                styleOverrides: {
                    root: {
                        minHeight: 38,
                        borderRadius: 10,
                        paddingInline: 16,
                    },
                    sizeSmall: {
                        minHeight: 32,
                        paddingInline: 11,
                    },
                    contained: {
                        boxShadow: 'none',
                    },
                    outlined: {
                        borderColor: colors.border,
                    },
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: {
                        borderRadius: 10,
                    },
                },
            },
            MuiChip: {
                styleOverrides: {
                    root: {
                        height: 26,
                        borderRadius: 8,
                        fontWeight: 680,
                    },
                    sizeSmall: {
                        height: 23,
                        fontSize: '0.72rem',
                    },
                },
            },
            MuiDialog: {
                styleOverrides: {
                    paper: {
                        borderRadius: 18,
                        border: `1px solid ${colors.border}`,
                        boxShadow: dark
                            ? '0 26px 80px rgba(0,0,0,.55)'
                            : '0 26px 80px rgba(22,34,54,.18)',
                    },
                },
            },
            MuiDialogTitle: {
                styleOverrides: {
                    root: {
                        fontWeight: 720,
                        padding: '20px 22px 10px',
                    },
                },
            },
            MuiDialogContent: {
                styleOverrides: {
                    root: {
                        paddingInline: 22,
                    },
                },
            },
            MuiDialogActions: {
                styleOverrides: {
                    root: {
                        padding: '12px 22px 20px',
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
                        borderRadius: 10,
                        backgroundColor: dark ? alpha('#FFFFFF', 0.015) : '#FFFFFF',
                    },
                    notchedOutline: {
                        borderColor: colors.border,
                    },
                },
            },
            MuiSelect: {
                styleOverrides: {
                    select: {
                        minHeight: '1.45em',
                    },
                },
            },
            MuiMenu: {
                defaultProps: {
                    elevation: 0,
                },
                styleOverrides: {
                    paper: {
                        minWidth: 220,
                        marginTop: 6,
                        border: `1px solid ${colors.border}`,
                        borderRadius: 12,
                        boxShadow: dark
                            ? '0 16px 44px rgba(0,0,0,.42)'
                            : '0 16px 44px rgba(25,39,62,.14)',
                    },
                },
            },
            MuiMenuItem: {
                styleOverrides: {
                    root: {
                        minHeight: 40,
                        borderRadius: 8,
                        marginInline: 6,
                        marginBlock: 2,
                    },
                },
            },
            MuiTableCell: {
                styleOverrides: {
                    root: {
                        borderBottomColor: colors.borderSoft,
                    },
                    head: {
                        color: colors.muted,
                        backgroundColor: colors.raised,
                        fontWeight: 750,
                        fontSize: '0.73rem',
                        textTransform: 'uppercase',
                        letterSpacing: '0.055em',
                    },
                },
            },
            MuiToggleButton: {
                styleOverrides: {
                    root: {
                        textTransform: 'none',
                        fontWeight: 650,
                        borderColor: colors.border,
                    },
                },
            },
            MuiAlert: {
                styleOverrides: {
                    root: {
                        borderRadius: 12,
                    },
                },
            },
            MuiTooltip: {
                defaultProps: {
                    arrow: true,
                    enterDelay: 420,
                },
            },
        },
    });
};
