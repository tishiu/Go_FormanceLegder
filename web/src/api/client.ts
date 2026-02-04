import axios from "axios";

export const api = axios.create({
    baseURL: "/api",
    withCredentials: true, // Include cookies in requests
});

// Response interceptor for handling auth errors
api.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            // Clear session cookie
            document.cookie = "session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
            // Only redirect if not already on auth pages
            if (!window.location.pathname.startsWith('/login') &&
                !window.location.pathname.startsWith('/register')) {
                window.location.href = "/login";
            }
        }
        return Promise.reject(error);
    }
);

