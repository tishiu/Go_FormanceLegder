/**
 * User and Authentication Types
 */

export interface User {
    id: string;
    email: string;
    created_at: string;
}

export interface Organization {
    id: string;
    name: string;
    created_at: string;
}

export interface Project {
    id: string;
    organization_id: string;
    name: string;
    code: string;
    created_at: string;
}

export interface AuthState {
    user: User | null;
    isAuthenticated: boolean;
    isLoading: boolean;
}

export interface LoginCredentials {
    email: string;
    password: string;
}

export interface RegisterCredentials {
    email: string;
    password: string;
}

export interface AuthResponse {
    token: string;
    user: User;
}
