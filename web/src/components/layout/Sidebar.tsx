import React, { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../lib/cn';
import {
    LayoutGrid,
    Layers,
    Wallet,
    ArrowLeftRight,
    Webhook,
    KeyRound,
    LogOut,
    Sun,
    Moon,
} from 'lucide-react';
import { useTheme } from '../../hooks/useTheme';

interface NavItem {
    path: string;
    label: string;
    icon: React.ComponentType<{ className?: string }>;
}

const navItems: NavItem[] = [
    { path: '/dashboard', label: 'Overview', icon: LayoutGrid },
    { path: '/dashboard/ledgers', label: 'Ledgers', icon: Layers },
    { path: '/dashboard/accounts', label: 'Accounts', icon: Wallet },
    { path: '/dashboard/transactions', label: 'Transactions', icon: ArrowLeftRight },
    { path: '/dashboard/webhooks', label: 'Webhooks', icon: Webhook },
    { path: '/dashboard/api-keys', label: 'API Keys', icon: KeyRound },
];

interface SidebarProps {
    user?: { email: string };
    onLogout: () => void;
}

export function Sidebar({ user, onLogout }: SidebarProps) {
    const [expanded, setExpanded] = useState(false);
    const location = useLocation();
    const { isDark, toggle } = useTheme();

    const isActive = (path: string) => {
        if (path === '/dashboard') return location.pathname === path;
        return location.pathname.startsWith(path);
    };

    return (
        <aside
            className={cn(
                'fixed left-0 top-0 h-full bg-sidebar-light dark:bg-sidebar-dark',
                'border-r border-border-light dark:border-border-dark',
                'flex flex-col transition-all duration-300 ease-out z-50',
                expanded ? 'w-56' : 'w-16'
            )}
            onMouseEnter={() => setExpanded(true)}
            onMouseLeave={() => setExpanded(false)}
        >
            {/* Logo */}
            <div className="h-16 flex items-center px-3 border-b border-border-light dark:border-border-dark">
                <div className="w-10 h-10 rounded-xl overflow-hidden bg-white flex items-center justify-center shrink-0 transition-transform duration-300 hover:scale-105 shadow-sm">
                    <img src="/logo.jpg" alt="Formance" className="w-full h-full object-cover" />
                </div>
                <div
                    className={cn(
                        'ml-3 overflow-hidden transition-all duration-300',
                        expanded ? 'opacity-100 w-auto' : 'opacity-0 w-0'
                    )}
                >
                    <h1 className="font-semibold text-text-primary-light dark:text-text-primary-dark whitespace-nowrap">
                        Formance
                    </h1>
                    {user && (
                        <p className="text-xs text-text-muted-light dark:text-text-muted-dark truncate max-w-[120px]">
                            {user.email}
                        </p>
                    )}
                </div>
            </div>

            {/* Navigation */}
            <nav className="flex-1 py-4 px-2 space-y-1 overflow-y-auto overflow-x-hidden">
                {navItems.map((item) => {
                    const Icon = item.icon;
                    const active = isActive(item.path);
                    return (
                        <Link
                            key={item.path}
                            to={item.path}
                            className={cn(
                                'group relative flex items-center rounded-xl px-3 py-2.5 transition-all duration-200',
                                active
                                    ? 'bg-primary text-white shadow-md shadow-primary/20'
                                    : 'text-text-secondary-light dark:text-text-secondary-dark hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-text-primary-light dark:hover:text-text-primary-dark'
                            )}
                        >
                            <Icon
                                className={cn(
                                    'w-5 h-5 shrink-0 transition-transform duration-200',
                                    !active && 'group-hover:scale-110'
                                )}
                            />
                            <span
                                className={cn(
                                    'ml-3 font-medium whitespace-nowrap transition-all duration-300',
                                    expanded ? 'opacity-100' : 'opacity-0 w-0'
                                )}
                            >
                                {item.label}
                            </span>
                            {!expanded && (
                                <span className="tooltip invisible opacity-0 absolute left-full ml-2 px-2 py-1 text-xs rounded-md bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900 transition-all duration-200 whitespace-nowrap z-50 group-hover:visible group-hover:opacity-100">
                                    {item.label}
                                </span>
                            )}
                        </Link>
                    );
                })}
            </nav>

            {/* Bottom Section */}
            <div className="p-2 border-t border-border-light dark:border-border-dark space-y-1">
                {/* Theme Toggle */}
                <button
                    onClick={toggle}
                    className="group relative w-full flex items-center rounded-xl px-3 py-2.5 text-text-secondary-light dark:text-text-secondary-dark hover:bg-gray-100 dark:hover:bg-gray-800 transition-all duration-200"
                >
                    <div className="relative w-5 h-5 shrink-0">
                        <Sun
                            className={cn(
                                'absolute inset-0 transition-all duration-300',
                                isDark ? 'opacity-100 rotate-0' : 'opacity-0 rotate-90'
                            )}
                        />
                        <Moon
                            className={cn(
                                'absolute inset-0 transition-all duration-300',
                                isDark ? 'opacity-0 -rotate-90' : 'opacity-100 rotate-0'
                            )}
                        />
                    </div>
                    <span
                        className={cn(
                            'ml-3 font-medium whitespace-nowrap transition-all duration-300',
                            expanded ? 'opacity-100' : 'opacity-0 w-0'
                        )}
                    >
                        {isDark ? 'Light Mode' : 'Dark Mode'}
                    </span>
                </button>

                {/* Logout */}
                <button
                    onClick={onLogout}
                    className="group relative w-full flex items-center rounded-xl px-3 py-2.5 text-accent-coral hover:bg-red-50 dark:hover:bg-red-900/20 transition-all duration-200"
                >
                    <LogOut className="w-5 h-5 shrink-0 transition-transform duration-200 group-hover:scale-110" />
                    <span
                        className={cn(
                            'ml-3 font-medium whitespace-nowrap transition-all duration-300',
                            expanded ? 'opacity-100' : 'opacity-0 w-0'
                        )}
                    >
                        Logout
                    </span>
                </button>
            </div>
        </aside>
    );
}
