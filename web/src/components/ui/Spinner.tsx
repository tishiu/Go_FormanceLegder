import { cn } from '../../lib/cn';

interface SpinnerProps {
    size?: 'sm' | 'md' | 'lg';
    className?: string;
}

const sizeStyles = {
    sm: 'w-4 h-4 border-2',
    md: 'w-6 h-6 border-2',
    lg: 'w-10 h-10 border-3',
};

export function Spinner({ size = 'md', className }: SpinnerProps) {
    return (
        <div
            className={cn(
                'rounded-full border-primary border-t-transparent animate-spin',
                sizeStyles[size],
                className
            )}
            role="status"
            aria-label="Loading"
        />
    );
}

interface SkeletonProps {
    className?: string;
}

export function Skeleton({ className }: SkeletonProps) {
    return (
        <div
            className={cn(
                'bg-gradient-to-r from-gray-200 via-gray-100 to-gray-200',
                'dark:from-gray-700 dark:via-gray-600 dark:to-gray-700',
                'bg-[length:200%_100%] animate-[shimmer_1.5s_infinite] rounded-md',
                className
            )}
            aria-hidden="true"
        />
    );
}

export function SkeletonText({ lines = 3, className }: { lines?: number; className?: string }) {
    return (
        <div className={cn('space-y-2', className)}>
            {Array.from({ length: lines }).map((_, i) => (
                <Skeleton
                    key={i}
                    className={cn('h-4', i === lines - 1 && 'w-3/4')}
                />
            ))}
        </div>
    );
}
