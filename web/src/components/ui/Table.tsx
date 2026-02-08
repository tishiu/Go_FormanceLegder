import { ReactNode } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface Column<T> {
    key: string;
    header: string;
    render?: (item: T) => ReactNode;
    sortable?: boolean;
    className?: string;
}

interface TableProps<T> {
    data: T[];
    columns: Column<T>[];
    onRowClick?: (item: T) => void;
    emptyMessage?: string;
    className?: string;
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
    onSort?: (key: string) => void;
}

export function Table<T extends Record<string, any>>({
    data,
    columns,
    onRowClick,
    emptyMessage = 'No data available',
    className,
    sortBy,
    sortOrder,
    onSort,
}: TableProps<T>) {
    const handleSort = (key: string) => {
        if (onSort) {
            onSort(key);
        }
    };

    if (data.length === 0) {
        return (
            <div className="text-center py-12">
                <p className="text-text-muted-light dark:text-text-muted-dark">
                    {emptyMessage}
                </p>
            </div>
        );
    }

    return (
        <div className={cn('overflow-x-auto rounded-lg border border-border-light dark:border-border-dark', className)}>
            <table className="w-full">
                <thead className="bg-gray-50 dark:bg-gray-800">
                    <tr>
                        {columns.map((column) => (
                            <th
                                key={column.key}
                                className={cn(
                                    'px-6 py-3 text-left text-xs font-medium text-text-secondary-light dark:text-text-secondary-dark uppercase tracking-wider',
                                    column.className,
                                    column.sortable && 'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 select-none'
                                )}
                                onClick={() => column.sortable && handleSort(column.key)}
                            >
                                <div className="flex items-center gap-2">
                                    {column.header}
                                    {column.sortable && sortBy === column.key && (
                                        sortOrder === 'asc' ? (
                                            <ChevronUp className="w-4 h-4" />
                                        ) : (
                                            <ChevronDown className="w-4 h-4" />
                                        )
                                    )}
                                </div>
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody className="bg-white dark:bg-sidebar-dark divide-y divide-border-light dark:divide-border-dark">
                    {data.map((item, index) => (
                        <tr
                            key={index}
                            className={cn(
                                'transition-colors',
                                onRowClick && 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800'
                            )}
                            onClick={() => onRowClick?.(item)}
                        >
                            {columns.map((column) => (
                                <td
                                    key={column.key}
                                    className={cn(
                                        'px-6 py-4 whitespace-nowrap text-sm text-text-primary-light dark:text-text-primary-dark',
                                        column.className
                                    )}
                                >
                                    {column.render ? column.render(item) : item[column.key]}
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
