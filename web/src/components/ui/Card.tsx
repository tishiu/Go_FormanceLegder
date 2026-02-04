import React from 'react';
import { cn } from '../../lib/cn';

export interface CardProps {
    children: React.ReactNode;
    className?: string;
    hover?: boolean;
    padding?: 'none' | 'sm' | 'md' | 'lg';
}

const paddingStyles = {
    none: '',
    sm: 'p-4',
    md: 'p-6',
    lg: 'p-8',
};

export function Card({ children, className, hover = false, padding = 'md' }: CardProps) {
    return (
        <div
            className={cn(
                'rounded-xl border border-border-light dark:border-border-dark',
                'bg-white dark:bg-sidebar-dark',
                'transition-all duration-200 ease-out',
                hover && 'hover:shadow-lg hover:border-primary/20 dark:hover:border-primary/30 cursor-pointer',
                hover && 'hover:-translate-y-0.5',
                paddingStyles[padding],
                className
            )}
        >
            {children}
        </div>
    );
}

export function CardHeader({ children, className }: { children: React.ReactNode; className?: string }) {
    return (
        <div className={cn('flex items-center justify-between mb-4', className)}>
            {children}
        </div>
    );
}

export function CardTitle({ children, className }: { children: React.ReactNode; className?: string }) {
    return (
        <h3 className={cn('font-semibold text-text-primary-light dark:text-text-primary-dark', className)}>
            {children}
        </h3>
    );
}

export function CardContent({ children, className }: { children: React.ReactNode; className?: string }) {
    return <div className={cn(className)}>{children}</div>;
}

export function CardFooter({ children, className }: { children: React.ReactNode; className?: string }) {
    return (
        <div className={cn('mt-4 pt-4 border-t border-border-light dark:border-border-dark', className)}>
            {children}
        </div>
    );
}
