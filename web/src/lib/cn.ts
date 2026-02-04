import { type ClassValue, clsx } from 'clsx';

/**
 * Utility function to merge class names conditionally
 * Uses clsx for conditional classes
 */
export function cn(...inputs: ClassValue[]): string {
    return clsx(inputs);
}
