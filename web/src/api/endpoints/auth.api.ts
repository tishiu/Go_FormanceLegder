import { api } from '../client';
import type { LoginCredentials, RegisterCredentials, AuthResponse, User } from '../../types';

export const authApi = {
    login: async (credentials: LoginCredentials): Promise<AuthResponse> => {
        const response = await api.post<AuthResponse>('/auth/login', credentials);
        return response.data;
    },

    register: async (credentials: RegisterCredentials): Promise<AuthResponse> => {
        const response = await api.post<AuthResponse>('/auth/register', credentials);
        return response.data;
    },

    me: async (): Promise<User> => {
        const response = await api.get<User>('/auth/me');
        return response.data;
    },
};
