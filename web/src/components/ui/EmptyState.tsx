import React from 'react';
import { cn } from '../../lib/cn';

interface EmptyStateProps {
    icon?: React.ReactNode;
    title?: string;
    message: string;
    action?: React.ReactNode;
    className?: string;
}

export function EmptyState({ icon, title, message, action, className }: EmptyStateProps) {
    return (
        <div className={cn('py-12 text-center', className)}>
            {icon && (
                <div className="w-14 h-14 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center mx-auto mb-4">
                    {icon}
                </div>
            )}
            {title && (
                <h3 className="text-lg font-semibold text-text-primary-light dark:text-text-primary-dark mb-1">
                    {title}
                </h3>
            )}
            <p className="text-sm text-text-muted-light dark:text-text-muted-dark max-w-sm mx-auto">
                {message}
            </p>
            {action && <div className="mt-4">{action}</div>}
        </div>
    );
}
