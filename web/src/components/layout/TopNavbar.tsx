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
    Menu,
    X,
} from 'lucide-react';

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

interface TopNavbarProps {
    user?: { email: string };
    onLogout: () => void;
    isDark: boolean;
    onToggleTheme: () => void;
}

export function TopNavbar({ user, onLogout, isDark, onToggleTheme }: TopNavbarProps) {
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [userMenuOpen, setUserMenuOpen] = useState(false);
    const location = useLocation();

    const isActive = (path: string) => {
        if (path === '/dashboard') return location.pathname === path;
        return location.pathname.startsWith(path);
    };

    return (
        <nav className="sticky top-0 z-50 bg-white dark:bg-gray-900 border-b border-border-light dark:border-border-dark shadow-sm">
            <div className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8">
                <div className="flex items-center justify-between h-14">
                    {/* Left: Logo + Navigation */}
                    <div className="flex items-center gap-6">
                        {/* Logo */}
                        <Link to="/dashboard" className="flex items-center gap-2.5 shrink-0">
                            <div className="w-8 h-8 rounded-lg overflow-hidden bg-white shadow-sm">
                                <img src="/logo.jpg" alt="Formance" className="w-full h-full object-cover" />
                            </div>
                            <span className="font-bold text-base text-text-primary-light dark:text-text-primary-dark">
                                Formance
                            </span>
                        </Link>

                        {/* Desktop Navigation */}
                        <div className="hidden lg:flex items-center gap-1">
                            {navItems.map((item) => {
                                const Icon = item.icon;
                                const active = isActive(item.path);
                                return (
                                    <Link
                                        key={item.path}
                                        to={item.path}
                                        className={cn(
                                            'flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200',
                                            active
                                                ? 'bg-primary/10 text-primary'
                                                : 'text-text-secondary-light dark:text-text-secondary-dark hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-text-primary-light dark:hover:text-text-primary-dark'
                                        )}
                                    >
                                        <Icon className="w-4 h-4" />
                                        <span>{item.label}</span>
                                    </Link>
                                );
                            })}
                        </div>
                    </div>

                    {/* Right: Theme Toggle + User Menu */}
                    <div className="hidden lg:flex items-center gap-3">
                        {/* Theme Toggle */}
                        <button
                            onClick={onToggleTheme}
                            className="relative flex items-center justify-center w-9 h-9 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors text-text-secondary-light dark:text-text-secondary-dark"
                            aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
                        >
                            <Sun
                                className={cn(
                                    'absolute w-4 h-4 transition-all duration-300',
                                    isDark ? 'opacity-100 rotate-0 scale-100' : 'opacity-0 rotate-90 scale-0'
                                )}
                            />
                            <Moon
                                className={cn(
                                    'absolute w-4 h-4 transition-all duration-300',
                                    isDark ? 'opacity-0 -rotate-90 scale-0' : 'opacity-100 rotate-0 scale-100'
                                )}
                            />
                        </button>

                        {/* User Menu */}
                        <div className="relative group">
                            <button
                                onClick={() => setUserMenuOpen(!userMenuOpen)}
                                className="flex items-center justify-center w-8 h-8 rounded-full bg-primary/10 hover:bg-primary/20 transition-colors"
                            >
                                <span className="text-xs font-bold text-primary">
                                    {user?.email?.[0]?.toUpperCase() || 'U'}
                                </span>
                            </button>

                            {/* Hover Tooltip - User Email */}
                            <div className="absolute right-0 top-full mt-2 px-3 py-1.5 bg-gray-900 text-white text-xs rounded-md shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 whitespace-nowrap z-50 pointer-events-none">
                                {user?.email}
                            </div>

                            {/* User Dropdown */}
                            {userMenuOpen && (
                                <>
                                    <div
                                        className="fixed inset-0 z-40"
                                        onClick={() => setUserMenuOpen(false)}
                                    />
                                    <div className="absolute right-0 mt-2 w-52 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-border-light dark:border-border-dark py-1 z-50 animate-in fade-in zoom-in-95 duration-100">
                                        <div className="px-3 py-2 border-b border-border-light dark:border-border-dark">
                                            <p className="text-xs font-medium text-text-primary-light dark:text-text-primary-dark">
                                                Signed in as
                                            </p>
                                            <p className="text-xs text-text-muted-light dark:text-text-muted-dark truncate font-medium">
                                                {user?.email}
                                            </p>
                                        </div>
                                        <button
                                            onClick={onLogout}
                                            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-accent-coral hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                                        >
                                            <LogOut className="w-4 h-4" />
                                            <span>Logout</span>
                                        </button>
                                    </div>
                                </>
                            )}
                        </div>
                    </div>

                    {/* Mobile Menu Button */}
                    <button
                        onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                        className="lg:hidden p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                        aria-label="Toggle menu"
                    >
                        {mobileMenuOpen ? (
                            <X className="w-5 h-5" />
                        ) : (
                            <Menu className="w-5 h-5" />
                        )}
                    </button>
                </div>
            </div>

            {/* Mobile Menu */}
            {mobileMenuOpen && (
                <div className="lg:hidden border-t border-border-light dark:border-border-dark">
                    <div className="px-4 py-3 space-y-1">
                        {navItems.map((item) => {
                            const Icon = item.icon;
                            const active = isActive(item.path);
                            return (
                                <Link
                                    key={item.path}
                                    to={item.path}
                                    onClick={() => setMobileMenuOpen(false)}
                                    className={cn(
                                        'flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200',
                                        active
                                            ? 'bg-primary text-white'
                                            : 'text-text-secondary-light dark:text-text-secondary-dark hover:bg-gray-100 dark:hover:bg-gray-800'
                                    )}
                                >
                                    <Icon className="w-4 h-4" />
                                    <span>{item.label}</span>
                                </Link>
                            );
                        })}

                        {/* Mobile: Theme Toggle */}
                        <button
                            onClick={onToggleTheme}
                            className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium text-text-secondary-light dark:text-text-secondary-dark hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                        >
                            {isDark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
                            <span>{isDark ? 'Light Mode' : 'Dark Mode'}</span>
                        </button>

                        {/* Mobile: User Info + Logout */}
                        <div className="pt-3 border-t border-border-light dark:border-border-dark mt-3">
                            <div className="px-3 py-2 mb-2">
                                <p className="text-xs text-text-muted-light dark:text-text-muted-dark">Signed in as</p>
                                <p className="text-sm font-medium text-text-primary-light dark:text-text-primary-dark truncate">
                                    {user?.email}
                                </p>
                            </div>
                            <button
                                onClick={onLogout}
                                className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium text-accent-coral hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                            >
                                <LogOut className="w-4 h-4" />
                                <span>Logout</span>
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </nav>
    );
}
