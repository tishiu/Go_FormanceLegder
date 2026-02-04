import React from 'react';
import { cn } from '../../lib/cn';
import { TrendingUp, TrendingDown } from 'lucide-react';

interface StatCardProps {
    title: string;
    value: string | number;
    icon: React.ReactNode;
    trend?: {
        value: string;
        direction: 'up' | 'down' | 'neutral';
    };
    color?: 'primary' | 'cyan' | 'green' | 'purple' | 'amber';
    className?: string;
}

const colorStyles = {
    primary: 'bg-primary/10 text-primary',
    cyan: 'bg-cyan-100 dark:bg-cyan-900/30 text-cyan-600 dark:text-cyan-400',
    green: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400',
    purple: 'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400',
    amber: 'bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400',
};

const trendColors = {
    up: 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30',
    down: 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/30',
    neutral: 'text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800',
};

export function StatCard({
    title,
    value,
    icon,
    trend,
    color = 'primary',
    className,
}: StatCardProps) {
    return (
        <div
            className={cn(
                'rounded-xl border border-border-light dark:border-border-dark',
                'bg-white dark:bg-sidebar-dark p-5',
                'transition-all duration-200 ease-out',
                'hover:shadow-lg hover:border-primary/20 dark:hover:border-primary/30',
                'hover:-translate-y-0.5',
                className
            )}
        >
            <div className="flex items-start justify-between mb-3">
                <div
                    className={cn(
                        'w-10 h-10 rounded-xl flex items-center justify-center',
                        'transition-transform duration-300 hover:scale-110',
                        colorStyles[color]
                    )}
                >
                    {icon}
                </div>
                {trend && (
                    <span
                        className={cn(
                            'inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full',
                            trendColors[trend.direction]
                        )}
                    >
                        {trend.direction === 'up' && <TrendingUp className="w-3 h-3" />}
                        {trend.direction === 'down' && <TrendingDown className="w-3 h-3" />}
                        {trend.value}
                    </span>
                )}
            </div>
            <div className="text-2xl font-bold text-text-primary-light dark:text-text-primary-dark mb-1">
                {value}
            </div>
            <div className="text-sm text-text-muted-light dark:text-text-muted-dark">
                {title}
            </div>
        </div>
    );
}
