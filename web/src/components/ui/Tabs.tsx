import { ReactNode } from 'react';
import { cn } from '../../lib/utils';

export interface Tab {
    id: string;
    label: string;
    icon?: React.ComponentType<{ className?: string }>;
    badge?: string | number;
}

interface TabsProps {
    tabs: Tab[];
    activeTab: string;
    onChange: (tabId: string) => void;
    className?: string;
}

export function Tabs({ tabs, activeTab, onChange, className }: TabsProps) {
    return (
        <div className={cn('border-b border-border-light dark:border-border-dark', className)}>
            <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                {tabs.map((tab) => {
                    const isActive = activeTab === tab.id;
                    const Icon = tab.icon;

                    return (
                        <button
                            key={tab.id}
                            onClick={() => onChange(tab.id)}
                            className={cn(
                                'group inline-flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm transition-colors',
                                isActive
                                    ? 'border-primary text-primary'
                                    : 'border-transparent text-text-muted-light dark:text-text-muted-dark hover:text-text-secondary-light dark:hover:text-text-secondary-dark hover:border-gray-300 dark:hover:border-gray-600'
                            )}
                            aria-current={isActive ? 'page' : undefined}
                        >
                            {Icon && (
                                <Icon
                                    className={cn(
                                        'w-5 h-5 transition-colors',
                                        isActive
                                            ? 'text-primary'
                                            : 'text-text-muted-light dark:text-text-muted-dark group-hover:text-text-secondary-light dark:group-hover:text-text-secondary-dark'
                                    )}
                                />
                            )}
                            {tab.label}
                            {tab.badge !== undefined && (
                                <span
                                    className={cn(
                                        'ml-1 py-0.5 px-2 rounded-full text-xs font-medium',
                                        isActive
                                            ? 'bg-primary/10 text-primary'
                                            : 'bg-gray-100 dark:bg-gray-800 text-text-muted-light dark:text-text-muted-dark'
                                    )}
                                >
                                    {tab.badge}
                                </span>
                            )}
                        </button>
                    );
                })}
            </nav>
        </div>
    );
}

interface TabPanelProps {
    children: ReactNode;
    className?: string;
}

export function TabPanel({ children, className }: TabPanelProps) {
    return (
        <div className={cn('py-6', className)}>
            {children}
        </div>
    );
}
