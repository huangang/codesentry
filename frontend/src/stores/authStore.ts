import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '../types';

interface AuthState {
  token: string | null;
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isDeveloper: boolean;
  expireAt: string | null;
  setAuth: (token: string, user: User) => void;
  setToken: (token: string) => void;
  setExpireAt: (expireAt: string | null) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      isAuthenticated: false,
      isAdmin: false,
      isDeveloper: false,
      expireAt: null,
      setAuth: (token, user) => {
        localStorage.setItem('token', token);
        set({
          token,
          user,
          isAuthenticated: true,
          isAdmin: user.role === 'admin',
          isDeveloper: user.role === 'developer',
        });
      },
      setToken: (token) => {
        localStorage.setItem('token', token);
        set({ token });
      },
      setExpireAt: (expireAt) => {
        set({ expireAt });
      },
      logout: () => {
        localStorage.removeItem('token');
        set({ token: null, user: null, isAuthenticated: false, isAdmin: false, isDeveloper: false, expireAt: null });
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ token: state.token, user: state.user, isAuthenticated: state.isAuthenticated, isAdmin: state.isAdmin, isDeveloper: state.isDeveloper, expireAt: state.expireAt }),
    }
  )
);
