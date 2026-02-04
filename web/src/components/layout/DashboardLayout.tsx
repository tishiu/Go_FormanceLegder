import React from 'react';
import { Sidebar } from './Sidebar';

interface DashboardLayoutProps {
    children: React.ReactNode;
    user?: { email: string };
    onLogout: () => void;
}

export function DashboardLayout({ children, user, onLogout }: DashboardLayoutProps) {
    return (
        <div className="min-h-screen bg-surface-light dark:bg-surface-dark transition-colors duration-300">
            <Sidebar user={user} onLogout={onLogout} />
            <main className="ml-16 min-h-screen transition-all duration-300">
                <div className="p-8 animate-fade-in">{children}</div>
            </main>
        </div>
    );
}
