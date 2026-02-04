import { useState, useEffect } from 'react';

/**
 * Hook to detect dark mode preference and manage theme state
 */
export function useTheme() {
    const [isDark, setIsDark] = useState(() => {
        if (typeof window !== 'undefined') {
            const saved = localStorage.getItem('theme');
            if (saved) return saved === 'dark';
            return window.matchMedia('(prefers-color-scheme: dark)').matches;
        }
        return false;
    });

    useEffect(() => {
        const root = document.documentElement;
        if (isDark) {
            root.classList.add('dark');
            localStorage.setItem('theme', 'dark');
        } else {
            root.classList.remove('dark');
            localStorage.setItem('theme', 'light');
        }
    }, [isDark]);

    const toggle = () => setIsDark(!isDark);
    const setTheme = (dark: boolean) => setIsDark(dark);

    return { isDark, toggle, setTheme };
}
