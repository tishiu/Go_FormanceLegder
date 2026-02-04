/**
 * Design Tokens: Colors
 * Consistent with tailwind.config.js theme extension
 */

export const colors = {
    // Brand / Primary (Teal)
    primary: {
        DEFAULT: '#1a7f7f',
        light: '#20a0a0',
        dark: '#156666',
        50: '#f0f9f9',
        100: '#d1eded',
        200: '#a3dbdb',
        300: '#75c9c9',
        400: '#47b7b7',
        500: '#1a7f7f',
        600: '#156666',
        700: '#104d4d',
        800: '#0b3434',
        900: '#061a1a',
    },

    // Semantic Colors
    success: {
        DEFAULT: '#10b981',
        light: '#d1fae5',
        dark: '#065f46',
    },
    warning: {
        DEFAULT: '#f59e0b',
        light: '#fef3c7',
        dark: '#92400e',
    },
    error: {
        DEFAULT: '#ef4444',
        light: '#fee2e2',
        dark: '#991b1b',
    },
    info: {
        DEFAULT: '#3b82f6',
        light: '#dbeafe',
        dark: '#1e40af',
    },

    // Surface Colors
    surface: {
        light: '#ffffff',
        dark: '#191a1a',
    },
    sidebar: {
        light: '#f9fafb',
        dark: '#1e1f20',
    },
    border: {
        light: '#e5e7eb',
        dark: '#2d2e2f',
    },

    // Text Colors
    text: {
        primary: {
            light: '#1f2937',
            dark: '#f3f4f6',
        },
        secondary: {
            light: '#6b7280',
            dark: '#9ca3af',
        },
        muted: {
            light: '#9ca3af',
            dark: '#6b7280',
        },
    },

    // Accent Colors
    accent: {
        teal: '#1a7f7f',
        cyan: '#06b6d4',
        coral: '#f97066',
        purple: '#8b5cf6',
        indigo: '#6366f1',
    },
} as const;

export type ColorToken = keyof typeof colors;
