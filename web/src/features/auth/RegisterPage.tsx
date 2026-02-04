import { useState, useEffect } from "react";
import { api } from "../../api/client";
import { useNavigate, Link } from "react-router-dom";
import { Mail, Lock, ArrowRight, Sun, Moon, Sparkles } from "lucide-react";

export function RegisterPage() {
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
            await api.post("/auth/register", { email, password });
            // Backend sets session cookie, redirect to dashboard
            navigate("/dashboard");
        } catch (err: unknown) {
            const axiosError = err as { response?: { status?: number; data?: string } };
            if (axiosError.response?.status === 409) {
                setError("Email already exists. Please use a different email or sign in.");
            } else if (axiosError.response?.data) {
                setError(axiosError.response.data);
            } else {
                setError("Registration failed. Please try again.");
            }
        } finally {
            setLoading(false);
        }
    }

    // Password strength indicator
    const getPasswordStrength = () => {
        if (password.length === 0) return { width: '0%', color: 'bg-gray-200', text: '' };
        if (password.length < 6) return { width: '25%', color: 'bg-red-500', text: 'Weak' };
        if (password.length < 8) return { width: '50%', color: 'bg-yellow-500', text: 'Fair' };
        if (password.length < 12) return { width: '75%', color: 'bg-green-400', text: 'Good' };
        return { width: '100%', color: 'bg-green-500', text: 'Strong' };
    };

    const strength = getPasswordStrength();

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
            <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-accent-cyan via-primary to-primary-dark items-center justify-center p-12 relative overflow-hidden">
                {/* Background Pattern */}
                <div className="absolute inset-0 opacity-10">
                    <div className="absolute top-10 right-10 w-64 h-64 bg-white rounded-full blur-3xl" />
                    <div className="absolute bottom-10 left-10 w-80 h-80 bg-white rounded-full blur-3xl" />
                </div>

                <div className="relative z-10 max-w-md animate-fade-in">
                    <div className="w-16 h-16 rounded-2xl bg-white/10 backdrop-blur flex items-center justify-center mb-8 transition-transform duration-300 hover:scale-105 hover:rotate-3">
                        <Sparkles className="w-8 h-8 text-white" />
                    </div>
                    <h1 className="text-4xl font-bold text-white mb-4">Start building today</h1>
                    <p className="text-white/70 text-lg mb-8">
                        Create your free account and start building with our ledger API in minutes.
                    </p>

                    <div className="grid grid-cols-2 gap-4">
                        {[
                            { value: '99.9%', label: 'Uptime SLA' },
                            { value: '<50ms', label: 'API Latency' },
                            { value: '10K+', label: 'Daily API Calls' },
                            { value: '24/7', label: 'Support' },
                        ].map((stat, i) => (
                            <div
                                key={stat.label}
                                className="bg-white/10 backdrop-blur rounded-xl p-4 animate-slide-up"
                                style={{ animationDelay: `${i * 100}ms` }}
                            >
                                <div className="text-2xl font-bold text-white">{stat.value}</div>
                                <div className="text-white/60 text-sm">{stat.label}</div>
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
                            Create your account
                        </h2>
                        <p className="text-text-muted-light dark:text-text-muted-dark mt-1">
                            Start your free trial, no credit card required
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
                                    placeholder="Minimum 6 characters"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    required
                                    minLength={6}
                                />
                            </div>
                            {/* Password Strength Indicator */}
                            {password.length > 0 && (
                                <div className="mt-2 animate-fade-in">
                                    <div className="h-1.5 w-full bg-gray-100 dark:bg-gray-800 rounded-full overflow-hidden">
                                        <div
                                            className={`h-full ${strength.color} transition-all duration-300`}
                                            style={{ width: strength.width }}
                                        />
                                    </div>
                                    <p className="text-xs text-text-muted-light dark:text-text-muted-dark mt-1">
                                        Password strength: <span className="font-medium">{strength.text}</span>
                                    </p>
                                </div>
                            )}
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
                                Create account
                                <ArrowRight className="w-4 h-4 transition-transform group-hover:translate-x-1" />
                            </>
                        )}
                    </button>

                    <p className="text-center text-xs text-text-muted-light dark:text-text-muted-dark">
                        By creating an account, you agree to our{" "}
                        <a href="#" className="text-primary hover:underline">Terms</a> and{" "}
                        <a href="#" className="text-primary hover:underline">Privacy Policy</a>
                    </p>

                    <div className="pt-4 border-t border-border-light dark:border-border-dark">
                        <p className="text-center text-sm text-text-muted-light dark:text-text-muted-dark">
                            Already have an account?{" "}
                            <Link to="/login" className="text-primary hover:text-primary-dark font-medium transition-colors">
                                Sign in
                            </Link>
                        </p>
                    </div>
                </form>
            </div>
        </div>
    );
}
