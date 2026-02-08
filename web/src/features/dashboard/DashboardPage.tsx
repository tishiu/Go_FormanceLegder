import { useEffect, useState } from "react";
import { Routes, Route, useNavigate } from "react-router-dom";
import { api } from "../../api/client";
import { TopNavbar } from "../../components/layout/TopNavbar";
import {
    Plus,
    TrendingUp,
    Clock,
    ChevronRight,
    Activity,
    BarChart3,
    Zap,
    Layers,
    Wallet,
    ArrowLeftRight,
    KeyRound,
} from "lucide-react";

// Theme hook with system preference detection
function useTheme() {
    const [isDark, setIsDark] = useState(() => {
        if (typeof window !== 'undefined') {
            const saved = localStorage.getItem('theme');
            if (saved) return saved === 'dark';
            return window.matchMedia('(prefers-color-scheme: dark)').matches;
        }
        return false;
    });

    useEffect(() => {
        const root = document.documentElement;
        if (isDark) {
            root.classList.add('dark');
            localStorage.setItem('theme', 'dark');
        } else {
            root.classList.remove('dark');
            localStorage.setItem('theme', 'light');
        }
    }, [isDark]);

    return { isDark, toggle: () => setIsDark(!isDark) };
}

export function DashboardPage() {
    const [user, setUser] = useState<{ email: string } | null>(null);
    const navigate = useNavigate();
    const { isDark, toggle } = useTheme();

    useEffect(() => {
        // Cookie-based auth: just try to get user info
        api.get("/auth/me")
            .then((res) => setUser(res.data))
            .catch(() => navigate("/login"));
    }, [navigate]);

    const handleLogout = () => {
        // Clear cookie by calling logout endpoint or just redirect
        document.cookie = "session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
        navigate("/login");
    };

    if (!user) {
        return (
            <div className="min-h-screen bg-surface-light dark:bg-surface-dark flex items-center justify-center">
                <div className="flex flex-col items-center gap-4">
                    <div className="w-10 h-10 rounded-full border-2 border-primary border-t-transparent animate-spin" />
                    <p className="text-text-muted-light dark:text-text-muted-dark text-sm animate-pulse">Loading...</p>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-surface-light dark:bg-surface-dark transition-colors duration-300">
            {/* Top Navbar */}
            <TopNavbar
                user={user}
                onLogout={handleLogout}
                isDark={isDark}
                onToggleTheme={toggle}
            />

            {/* Main Content */}
            <main className="min-h-[calc(100vh-3.5rem)]">
                <div className="max-w-7xl mx-auto p-6 sm:p-8 animate-fade-in">
                    <Routes>
                        <Route index element={<OverviewSection />} />
                        <Route path="ledgers" element={<LedgersSection />} />
                        <Route path="accounts" element={<AccountsSection />} />
                        <Route path="transactions" element={<TransactionsSection />} />
                        <Route path="webhooks" element={<WebhooksSection />} />
                        <Route path="api-keys" element={<ApiKeysSection />} />
                    </Routes>
                </div>
            </main>
        </div>
    );
}

// Overview Section with enhanced visuals
function OverviewSection() {
    return (
        <div className="max-w-6xl space-y-8">
            {/* Header */}
            <div className="animate-slide-up">
                <h1 className="text-2xl font-semibold text-text-primary-light dark:text-text-primary-dark">
                    Dashboard
                </h1>
                <p className="text-text-muted-light dark:text-text-muted-dark mt-1">
                    Welcome back! Here's an overview of your ledger activity.
                </p>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard
                    title="Total Ledgers"
                    value="0"
                    icon={Layers}
                    trend="+0%"
                    color="primary"
                    delay={0}
                />
                <StatCard
                    title="Accounts"
                    value="0"
                    icon={Wallet}
                    trend="+0%"
                    color="cyan"
                    delay={1}
                />
                <StatCard
                    title="Transactions"
                    value="0"
                    icon={ArrowLeftRight}
                    trend="+0%"
                    color="green"
                    delay={2}
                />
                <StatCard
                    title="API Calls"
                    value="0"
                    icon={Activity}
                    trend="+0%"
                    color="purple"
                    delay={3}
                />
            </div>

            {/* Two Column Layout */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Recent Activity */}
                <div className="card p-6 animate-slide-up stagger-4">
                    <div className="flex items-center justify-between mb-5">
                        <div className="flex items-center gap-3">
                            <div className="w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center">
                                <Clock className="w-4 h-4 text-primary" />
                            </div>
                            <h3 className="font-semibold text-text-primary-light dark:text-text-primary-dark">
                                Recent Activity
                            </h3>
                        </div>
                        <button className="text-sm text-primary hover:underline flex items-center gap-1 transition-all hover:gap-2">
                            View all <ChevronRight className="w-4 h-4" />
                        </button>
                    </div>
                    <div className="space-y-3">
                        <EmptyState message="No recent activity to show" />
                    </div>
                </div>

                {/* Quick Actions */}
                <div className="card p-6 animate-slide-up stagger-5">
                    <div className="flex items-center gap-3 mb-5">
                        <div className="w-9 h-9 rounded-lg bg-accent-cyan/10 flex items-center justify-center">
                            <Zap className="w-4 h-4 text-accent-cyan" />
                        </div>
                        <h3 className="font-semibold text-text-primary-light dark:text-text-primary-dark">
                            Quick Actions
                        </h3>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                        <ActionButton icon={Plus} label="Create Ledger" primary />
                        <ActionButton icon={ArrowLeftRight} label="New Transaction" />
                        <ActionButton icon={Wallet} label="Add Account" />
                        <ActionButton icon={KeyRound} label="Generate Key" />
                    </div>
                </div>
            </div>

            {/* Chart Placeholder */}
            <div className="card p-6 animate-slide-up stagger-6">
                <div className="flex items-center gap-3 mb-5">
                    <div className="w-9 h-9 rounded-lg bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
                        <BarChart3 className="w-4 h-4 text-green-600 dark:text-green-400" />
                    </div>
                    <h3 className="font-semibold text-text-primary-light dark:text-text-primary-dark">
                        Transaction Volume
                    </h3>
                </div>
                <div className="h-48 flex items-center justify-center border-2 border-dashed border-border-light dark:border-border-dark rounded-lg">
                    <p className="text-text-muted-light dark:text-text-muted-dark text-sm">
                        Chart will appear when you have transactions
                    </p>
                </div>
            </div>
        </div>
    );
}

// Enhanced Stat Card
function StatCard({
    title,
    value,
    icon: Icon,
    trend,
    color,
    delay
}: {
    title: string;
    value: string;
    icon: React.ComponentType<{ className?: string }>;
    trend: string;
    color: 'primary' | 'cyan' | 'green' | 'purple';
    delay: number;
}) {
    const colors = {
        primary: 'bg-primary/10 text-primary',
        cyan: 'bg-cyan-100 dark:bg-cyan-900/30 text-cyan-600 dark:text-cyan-400',
        green: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400',
        purple: 'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400',
    };

    return (
        <div
            className={`card card-hover p-5 animate-slide-up stagger-${delay + 1}`}
        >
            <div className="flex items-start justify-between mb-3">
                <div className={`w-10 h-10 rounded-xl ${colors[color]} flex items-center justify-center transition-transform duration-300 hover:scale-110`}>
                    <Icon className="w-5 h-5" />
                </div>
                <span className="inline-flex items-center gap-1 text-xs font-medium text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30 px-2 py-0.5 rounded-full">
                    <TrendingUp className="w-3 h-3" />
                    {trend}
                </span>
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

// Action Button Component
function ActionButton({
    icon: Icon,
    label,
    primary
}: {
    icon: React.ComponentType<{ className?: string }>;
    label: string;
    primary?: boolean;
}) {
    return (
        <button className={`flex items-center gap-2 px-4 py-3 rounded-xl font-medium text-sm transition-all duration-200 ${primary
            ? 'bg-primary text-white hover:bg-primary-dark shadow-md shadow-primary/20 hover:shadow-lg hover:shadow-primary/30'
            : 'border border-border-light dark:border-border-dark text-text-primary-light dark:text-text-primary-dark hover:bg-gray-50 dark:hover:bg-gray-800 hover:border-primary/30'
            } active:scale-95`}>
            <Icon className="w-4 h-4" />
            {label}
        </button>
    );
}

// Empty State Component
function EmptyState({ message }: { message: string }) {
    return (
        <div className="py-8 text-center">
            <div className="w-12 h-12 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center mx-auto mb-3">
                <Clock className="w-5 h-5 text-text-muted-light dark:text-text-muted-dark" />
            </div>
            <p className="text-sm text-text-muted-light dark:text-text-muted-dark">{message}</p>
        </div>
    );
}

// Section Components
function LedgersSection() {
    return (
        <div className="max-w-6xl animate-fade-in">
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h1 className="text-2xl font-semibold text-text-primary-light dark:text-text-primary-dark">Ledgers</h1>
                    <p className="text-text-muted-light dark:text-text-muted-dark mt-1">Manage your ledger books</p>
                </div>
                <button className="btn btn-primary px-4 py-2.5 text-sm">
                    <Plus className="w-4 h-4" />
                    Create Ledger
                </button>
            </div>
            <div className="card p-8">
                <EmptyState message="No ledgers yet. Create your first ledger to get started." />
            </div>
        </div>
    );
}

function AccountsSection() {
    return (
        <div className="max-w-6xl animate-fade-in">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-text-primary-light dark:text-text-primary-dark">Accounts</h1>
                <p className="text-text-muted-light dark:text-text-muted-dark mt-1">View and manage accounts</p>
            </div>
            <div className="card p-8">
                <EmptyState message="Select a ledger to view its accounts." />
            </div>
        </div>
    );
}

function TransactionsSection() {
    return (
        <div className="max-w-6xl animate-fade-in">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-text-primary-light dark:text-text-primary-dark">Transactions</h1>
                <p className="text-text-muted-light dark:text-text-muted-dark mt-1">Transaction history and details</p>
            </div>
            <div className="card p-8">
                <EmptyState message="No transactions yet." />
            </div>
        </div>
    );
}

function WebhooksSection() {
    return (
        <div className="max-w-6xl animate-fade-in">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold text-text-primary-light dark:text-text-primary-dark">Webhooks</h1>
                <p className="text-text-muted-light dark:text-text-muted-dark mt-1">Configure webhook endpoints</p>
            </div>
            <div className="card p-8">
                <EmptyState message="No webhooks configured." />
            </div>
        </div>
    );
}

function ApiKeysSection() {
    return (
        <div className="max-w-6xl animate-fade-in">
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h1 className="text-2xl font-semibold text-text-primary-light dark:text-text-primary-dark">API Keys</h1>
                    <p className="text-text-muted-light dark:text-text-muted-dark mt-1">Manage API access credentials</p>
                </div>
                <button className="btn btn-primary px-4 py-2.5 text-sm">
                    <Plus className="w-4 h-4" />
                    Generate Key
                </button>
            </div>
            <div className="card p-8">
                <EmptyState message="No API keys yet. Generate a key to access the API." />
            </div>
        </div>
    );
}
