import React from 'react';
import { cn } from '../../lib/cn';

interface PageHeaderProps {
    title: string;
    description?: string;
    action?: React.ReactNode;
    className?: string;
}

export function PageHeader({ title, description, action, className }: PageHeaderProps) {
    return (
        <div className={cn('flex items-center justify-between mb-6', className)}>
            <div>
                <h1 className="text-2xl font-semibold text-text-primary-light dark:text-text-primary-dark">
                    {title}
                </h1>
                {description && (
                    <p className="text-text-muted-light dark:text-text-muted-dark mt-1">
                        {description}
                    </p>
                )}
            </div>
            {action && <div>{action}</div>}
        </div>
    );
}
