import {alpha, createTheme, PaletteMode} from '@mui/material/styles';

export type ThemeKey = 'dark' | 'light' | 'system';

export const isThemeKey = (value: string | null): value is ThemeKey =>
    value === 'light' || value === 'dark' || value === 'system';

export const createMonitaTheme = (mode: PaletteMode) => {
    const dark = mode === 'dark';
    const border = dark ? '#25324A' : '#D9E2EF';
    const softBorder = dark ? '#1E2A3E' : '#E8EDF5';
    const canvas = dark ? '#09111F' : '#F3F6FA';
    const paper = dark ? '#101B2D' : '#FFFFFF';
    const raised = dark ? '#142238' : '#F9FBFD';

    const base = createTheme({
        palette: {
            mode,
            primary: {
                main: '#2F6BFF',
                light: '#6B95FF',
                dark: '#1D4ED8',
                contrastText: '#FFFFFF',
            },
            secondary: {
                main: '#12A8A0',
                contrastText: '#FFFFFF',
            },
            info: {main: '#3C8DFF'},
            success: {main: '#12A36D'},
            warning: {main: '#D88A16'},
            error: {main: '#D94B5B'},
            background: {default: canvas, paper},
            text: dark
                ? {primary: '#F4F7FB', secondary: '#9BAAC0'}
                : {primary: '#152238', secondary: '#617087'},
            divider: border,
        },
        shape: {borderRadius: 14},
        typography: {
            fontFamily:
                'Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
            h4: {fontWeight: 780, letterSpacing: '-0.035em'},
            h5: {fontWeight: 760, letterSpacing: '-0.025em'},
            h6: {fontWeight: 730, letterSpacing: '-0.018em'},
            subtitle1: {fontWeight: 680},
            button: {textTransform: 'none', fontWeight: 700, letterSpacing: '-0.01em'},
            body2: {lineHeight: 1.55},
            overline: {fontWeight: 760, letterSpacing: '0.09em'},
        },
    });

    return createTheme(base, {
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    html: {backgroundColor: canvas},
                    body: {
                        backgroundColor: canvas,
                        backgroundImage: 'none',
                        scrollbarColor: dark ? '#3D4D68 transparent' : '#B7C2D2 transparent',
                    },
                    '::selection': {
                        backgroundColor: alpha(base.palette.primary.main, 0.24),
                    },
                },
            },
            MuiAppBar: {
                styleOverrides: {
                    root: {
                        backgroundImage: 'none',
                        boxShadow: 'none',
                    },
                },
            },
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundImage: 'none',
                    },
                    outlined: {
                        borderColor: border,
                        boxShadow: dark
                            ? '0 14px 34px rgba(0, 0, 0, 0.18)'
                            : '0 12px 30px rgba(35, 55, 80, 0.07)',
                    },
                },
            },
            MuiButton: {
                defaultProps: {disableElevation: true},
                styleOverrides: {
                    root: {
                        borderRadius: 11,
                        minHeight: 38,
                        paddingInline: 15,
                    },
                    contained: {
                        boxShadow: 'none',
                    },
                    outlined: {
                        borderColor: border,
                        backgroundColor: alpha(paper, 0.74),
                    },
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: {
                        borderRadius: 11,
                    },
                },
            },
            MuiChip: {
                styleOverrides: {
                    root: {
                        borderRadius: 9,
                        fontWeight: 700,
                    },
                },
            },
            MuiOutlinedInput: {
                styleOverrides: {
                    root: {
                        borderRadius: 11,
                        backgroundColor: raised,
                    },
                    notchedOutline: {
                        borderColor: border,
                    },
                },
            },
            MuiTableCell: {
                styleOverrides: {
                    root: {borderBottomColor: softBorder},
                    head: {
                        color: base.palette.text.secondary,
                        fontSize: '0.74rem',
                        fontWeight: 760,
                        textTransform: 'uppercase',
                        letterSpacing: '0.075em',
                        backgroundColor: raised,
                    },
                },
            },
            MuiDialog: {
                styleOverrides: {
                    paper: {
                        borderRadius: 20,
                        border: `1px solid ${border}`,
                        boxShadow: dark
                            ? '0 24px 80px rgba(0,0,0,.46)'
                            : '0 24px 80px rgba(31,48,74,.18)',
                    },
                },
            },
            MuiMenu: {
                defaultProps: {elevation: 0},
                styleOverrides: {
                    paper: {
                        border: `1px solid ${border}`,
                        borderRadius: 14,
                        minWidth: 220,
                        boxShadow: dark
                            ? '0 18px 50px rgba(0,0,0,.38)'
                            : '0 18px 50px rgba(31,48,74,.14)',
                    },
                },
            },
            MuiListItemButton: {
                styleOverrides: {
                    root: {
                        borderRadius: 11,
                        '&.Mui-selected': {
                            backgroundColor: alpha(base.palette.primary.main, dark ? 0.2 : 0.1),
                        },
                    },
                },
            },
            MuiTooltip: {
                defaultProps: {arrow: true, enterDelay: 400},
            },
        },
    });
};
