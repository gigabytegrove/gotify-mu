import {createTheme, PaletteMode} from '@mui/material/styles';

export const createMonitaTheme = (mode: PaletteMode) => {
    const dark = mode === 'dark';
    const border = dark ? '#263449' : '#dce4ee';
    const softBorder = dark ? '#1e2a3c' : '#e8edf3';

    const theme = createTheme({
        palette: {
            mode,
            primary: {
                main: '#2563EB',
                light: '#60A5FA',
                dark: '#1D4ED8',
                contrastText: '#ffffff',
            },
            info: {main: '#0891B2'},
            success: {main: '#16A34A'},
            warning: {main: '#D97706'},
            background: dark
                ? {default: '#0B1220', paper: '#111B2E'}
                : {default: '#F3F6FA', paper: '#FFFFFF'},
            text: dark
                ? {primary: '#F8FAFC', secondary: '#9FB0C5'}
                : {primary: '#172033', secondary: '#607086'},
            divider: border,
            action: {
                hover: dark ? 'rgba(148, 163, 184, 0.09)' : 'rgba(37, 99, 235, 0.055)',
                selected: dark ? 'rgba(37, 99, 235, 0.18)' : 'rgba(37, 99, 235, 0.09)',
            },
        },
        shape: {borderRadius: 12},
        typography: {
            fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
            h4: {fontWeight: 800, letterSpacing: '-0.035em'},
            h5: {fontWeight: 780, letterSpacing: '-0.025em'},
            h6: {fontWeight: 760, letterSpacing: '-0.018em'},
            button: {textTransform: 'none', fontWeight: 700, letterSpacing: '-0.01em'},
            body2: {lineHeight: 1.55},
            overline: {fontWeight: 800, letterSpacing: '0.11em'},
        },
    });

    return createTheme(theme, {
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    html: {backgroundColor: theme.palette.background.default},
                    body: {
                        backgroundColor: theme.palette.background.default,
                        scrollbarColor: dark ? '#41516a transparent' : '#b4c0cd transparent',
                    },
                    '::selection': {
                        backgroundColor: dark ? 'rgba(96,165,250,.35)' : 'rgba(37,99,235,.18)',
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
            MuiButton: {
                defaultProps: {disableElevation: true},
                styleOverrides: {
                    root: {
                        borderRadius: 10,
                        minHeight: 36,
                        paddingInline: 15,
                    },
                    sizeSmall: {
                        minHeight: 31,
                        paddingInline: 11,
                    },
                    outlined: {
                        borderColor: border,
                        backgroundColor: dark ? 'rgba(255,255,255,.015)' : '#fff',
                        '&:hover': {
                            borderColor: theme.palette.primary.main,
                            backgroundColor: theme.palette.action.hover,
                        },
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
            MuiPaper: {
                styleOverrides: {
                    root: {backgroundImage: 'none'},
                    outlined: {borderColor: border},
                },
            },
            MuiDialog: {
                styleOverrides: {
                    paper: {
                        borderRadius: 18,
                        border: `1px solid ${border}`,
                        boxShadow: dark
                            ? '0 24px 70px rgba(0,0,0,.44)'
                            : '0 24px 70px rgba(32,52,78,.18)',
                    },
                },
            },
            MuiTableCell: {
                styleOverrides: {
                    root: {borderBottomColor: softBorder},
                    head: {
                        fontWeight: 800,
                        color: theme.palette.text.secondary,
                        fontSize: '0.73rem',
                        textTransform: 'uppercase',
                        letterSpacing: '0.075em',
                        backgroundColor: dark ? '#10192a' : '#f7f9fc',
                    },
                },
            },
            MuiChip: {
                styleOverrides: {
                    root: {
                        fontWeight: 700,
                        height: 25,
                        borderRadius: 8,
                    },
                    sizeSmall: {
                        height: 23,
                        fontSize: '0.72rem',
                    },
                },
            },
            MuiTextField: {
                defaultProps: {size: 'small'},
            },
            MuiOutlinedInput: {
                styleOverrides: {
                    root: {
                        borderRadius: 11,
                        backgroundColor: dark ? 'rgba(3,8,17,.18)' : '#fff',
                    },
                },
            },
            MuiTooltip: {
                defaultProps: {arrow: true, enterDelay: 400},
            },
            MuiMenu: {
                defaultProps: {elevation: 8},
                styleOverrides: {
                    paper: {
                        border: `1px solid ${border}`,
                        borderRadius: 13,
                        minWidth: 220,
                    },
                },
            },
            MuiAlert: {
                styleOverrides: {
                    root: {borderRadius: 12},
                },
            },
        },
    });
};
