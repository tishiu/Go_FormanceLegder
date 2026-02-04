import { cn } from '../../lib/cn';

interface AvatarProps {
    src?: string;
    alt?: string;
    fallback?: string;
    size?: 'sm' | 'md' | 'lg';
    className?: string;
}

const sizeStyles = {
    sm: 'w-8 h-8 text-xs',
    md: 'w-10 h-10 text-sm',
    lg: 'w-12 h-12 text-base',
};

export function Avatar({ src, alt, fallback, size = 'md', className }: AvatarProps) {
    const initials = fallback
        ? fallback.slice(0, 2).toUpperCase()
        : alt?.slice(0, 2).toUpperCase() || '?';

    return (
        <div
            className={cn(
                'rounded-full bg-primary/10 flex items-center justify-center font-medium text-primary overflow-hidden',
                sizeStyles[size],
                className
            )}
        >
            {src ? (
                <img src={src} alt={alt || 'Avatar'} className="w-full h-full object-cover" />
            ) : (
                <span>{initials}</span>
            )}
        </div>
    );
}
