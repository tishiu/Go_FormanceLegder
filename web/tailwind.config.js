/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    darkMode: 'class',
    theme: {
        extend: {
            colors: {
                // Perplexity-inspired palette
                primary: {
                    DEFAULT: '#1a7f7f', // Teal
                    light: '#20a0a0',
                    dark: '#156666',
                },
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
                    }
                },
                accent: {
                    teal: '#1a7f7f',
                    cyan: '#06b6d4',
                    coral: '#f97066',
                }
            },
            fontFamily: {
                sans: ['Inter', 'system-ui', 'sans-serif'],
            },
        },
    },
    plugins: [],
}
