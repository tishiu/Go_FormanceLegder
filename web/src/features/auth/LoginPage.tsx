import { useState, useEffect } from "react";
import { api } from "../../api/client";
import { useNavigate, Link } from "react-router-dom";
import { Mail, Lock, ArrowRight, Sun, Moon, Check } from "lucide-react";

export function LoginPage() {
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [isDark, setIsDark] = useState(() => {
        if (typeof window !== 'undefined') {
            return localStorage.getItem('theme') === 'dark';
        }
        return false;
    });
    const navigate = useNavigate();

    useEffect(() => {
        if (isDark) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }, [isDark]);

    async function onSubmit(e: React.FormEvent) {
        e.preventDefault();
        setError(null);
        setLoading(true);
        try {
            await api.post("/auth/login", { email, password });
            // Backend sets session cookie, redirect to dashboard
            navigate("/dashboard");
        } catch (err: unknown) {
            const axiosError = err as { response?: { status?: number; data?: string } };
            if (axiosError.response?.status === 401) {
                setError("Invalid email or password. Please try again.");
            } else if (axiosError.response?.data) {
                setError(axiosError.response.data);
            } else {
                setError("Login failed. Please try again.");
            }
        } finally {
            setLoading(false);
        }
    }

    const features = [
        "Double-entry accounting",
        "Real-time balance updates",
        "Webhook notifications",
        "RESTful API access"
    ];

    return (
        <div className="flex min-h-screen bg-surface-light dark:bg-surface-dark transition-colors duration-300">
            {/* Theme toggle */}
            <button
                onClick={() => setIsDark(!isDark)}
                className="fixed top-4 right-4 p-2.5 rounded-xl bg-white dark:bg-gray-800 border border-border-light dark:border-border-dark text-text-secondary-light dark:text-text-secondary-dark hover:bg-gray-50 dark:hover:bg-gray-700 transition-all duration-200 shadow-sm z-50"
                aria-label="Toggle theme"
            >
                <div className="relative w-5 h-5">
                    <Sun className={`absolute inset-0 transition-all duration-300 ${isDark ? 'opacity-100 rotate-0' : 'opacity-0 rotate-90'}`} />
                    <Moon className={`absolute inset-0 transition-all duration-300 ${isDark ? 'opacity-0 -rotate-90' : 'opacity-100 rotate-0'}`} />
                </div>
            </button>

            {/* Left panel - branding */}
            <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-primary via-primary to-primary-dark items-center justify-center p-12 relative overflow-hidden">
                {/* Background Pattern */}
                <div className="absolute inset-0 opacity-10">
                    <div className="absolute top-20 left-20 w-72 h-72 bg-white rounded-full blur-3xl" />
                    <div className="absolute bottom-20 right-20 w-96 h-96 bg-white rounded-full blur-3xl" />
                </div>

                <div className="relative z-10 max-w-md animate-fade-in">
                    <div className="w-20 h-20 rounded-2xl bg-white/10 backdrop-blur flex items-center justify-center mb-8 transition-transform duration-300 hover:scale-105 overflow-hidden">
                        <img src="/logo.jpg" alt="Formance" className="w-full h-full object-cover" />
                    </div>
                    <h1 className="text-4xl font-bold text-white mb-4">Formance Ledger</h1>
                    <p className="text-white/70 text-lg mb-8">
                        Modern ledger infrastructure for developers. Build financial products with confidence.
                    </p>

                    <div className="space-y-3">
                        {features.map((feature, i) => (
                            <div
                                key={feature}
                                className="flex items-center gap-3 text-white/90 animate-slide-in"
                                style={{ animationDelay: `${i * 100}ms` }}
                            >
                                <div className="w-5 h-5 rounded-full bg-white/20 flex items-center justify-center">
                                    <Check className="w-3 h-3" />
                                </div>
                                {feature}
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            {/* Right panel - form */}
            <div className="flex-1 flex items-center justify-center p-8">
                <form onSubmit={onSubmit} className="w-full max-w-sm space-y-6 animate-slide-up">
                    {/* Mobile Logo */}
                    <div className="lg:hidden flex items-center gap-3 mb-8">
                        <div className="w-11 h-11 rounded-xl overflow-hidden shadow-lg shadow-primary/20">
                            <img src="/logo.jpg" alt="Formance" className="w-full h-full object-cover" />
                        </div>
                        <span className="text-xl font-bold text-text-primary-light dark:text-text-primary-dark">Formance</span>
                    </div>

                    <div>
                        <h2 className="text-2xl font-bold text-text-primary-light dark:text-text-primary-dark">
                            Welcome back
                        </h2>
                        <p className="text-text-muted-light dark:text-text-muted-dark mt-1">
                            Sign in to continue to your dashboard
                        </p>
                    </div>

                    {error && (
                        <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800/50 p-4 text-red-600 dark:text-red-400 text-sm animate-scale-in flex items-start gap-3">
                            <div className="w-5 h-5 rounded-full bg-red-100 dark:bg-red-900/50 flex items-center justify-center shrink-0 mt-0.5">
                                <span className="text-xs">!</span>
                            </div>
                            {error}
                        </div>
                    )}

                    <div className="space-y-4">
                        <div className="space-y-1.5">
                            <label className="block text-sm font-medium text-text-primary-light dark:text-text-primary-dark">
                                Email address
                            </label>
                            <div className="relative group">
                                <Mail className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-text-muted-light dark:text-text-muted-dark transition-colors group-focus-within:text-primary" />
                                <input
                                    className="input pl-11 pr-4 py-3"
                                    type="email"
                                    placeholder="you@example.com"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    required
                                />
                            </div>
                        </div>
                        <div className="space-y-1.5">
                            <label className="block text-sm font-medium text-text-primary-light dark:text-text-primary-dark">
                                Password
                            </label>
                            <div className="relative group">
                                <Lock className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-text-muted-light dark:text-text-muted-dark transition-colors group-focus-within:text-primary" />
                                <input
                                    className="input pl-11 pr-4 py-3"
                                    type="password"
                                    placeholder="••••••••"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    required
                                />
                            </div>
                        </div>
                    </div>

                    <button
                        type="submit"
                        disabled={loading}
                        className="w-full btn btn-primary py-3 text-base shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 active:scale-[0.98]"
                    >
                        {loading ? (
                            <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                        ) : (
                            <>
                                Sign in
                                <ArrowRight className="w-4 h-4 transition-transform group-hover:translate-x-1" />
                            </>
                        )}
                    </button>

                    <p className="text-center text-sm text-text-muted-light dark:text-text-muted-dark">
                        Don't have an account?{" "}
                        <Link to="/register" className="text-primary hover:text-primary-dark font-medium transition-colors">
                            Create account
                        </Link>
                    </p>
                </form>
            </div>
        </div>
    );
}
