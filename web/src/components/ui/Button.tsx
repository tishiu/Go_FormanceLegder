import React, { forwardRef } from 'react';
import { cn } from '../../lib/cn';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
    size?: 'sm' | 'md' | 'lg';
    isLoading?: boolean;
    leftIcon?: React.ReactNode;
    rightIcon?: React.ReactNode;
}

const variantStyles = {
    primary: 'bg-primary text-white hover:bg-primary-dark shadow-md shadow-primary/20 hover:shadow-lg',
    secondary: 'border border-border-light dark:border-border-dark text-text-primary-light dark:text-text-primary-dark hover:bg-gray-50 dark:hover:bg-gray-800',
    ghost: 'text-text-secondary-light dark:text-text-secondary-dark hover:bg-gray-100 dark:hover:bg-gray-800',
    danger: 'bg-red-500 text-white hover:bg-red-600 shadow-md shadow-red-500/20',
};

const sizeStyles = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2.5 text-sm',
    lg: 'px-6 py-3 text-base',
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
    (
        {
            className,
            variant = 'primary',
            size = 'md',
            isLoading = false,
            disabled,
            leftIcon,
            rightIcon,
            children,
            ...props
        },
        ref
    ) => {
        const isDisabled = disabled || isLoading;

        return (
            <button
                ref={ref}
                className={cn(
                    'inline-flex items-center justify-center gap-2 rounded-xl font-medium',
                    'transition-all duration-200 ease-out',
                    'focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary/50',
                    'active:scale-[0.98]',
                    variantStyles[variant],
                    sizeStyles[size],
                    isDisabled && 'opacity-50 cursor-not-allowed',
                    className
                )}
                disabled={isDisabled}
                {...props}
            >
                {isLoading ? (
                    <span className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                ) : leftIcon ? (
                    <span className="w-4 h-4">{leftIcon}</span>
                ) : null}
                {children}
                {rightIcon && !isLoading && <span className="w-4 h-4">{rightIcon}</span>}
            </button>
        );
    }
);

Button.displayName = 'Button';
