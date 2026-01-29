import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type ThemeMode = 'light' | 'dark';

interface ThemeState {
    mode: ThemeMode;
    isDark: boolean;
    toggleTheme: () => void;
    setTheme: (mode: ThemeMode) => void;
}

// Default to light theme for new users
const getDefaultTheme = (): ThemeMode => {
    return 'light';
};

export const useThemeStore = create<ThemeState>()(
    persist(
        (set) => ({
            mode: getDefaultTheme(),
            isDark: false,
            toggleTheme: () =>
                set((state) => {
                    const newMode = state.mode === 'light' ? 'dark' : 'light';
                    return { mode: newMode, isDark: newMode === 'dark' };
                }),
            setTheme: (mode) => set({ mode, isDark: mode === 'dark' }),
        }),
        {
            name: 'theme-storage',
            partialize: (state) => ({ mode: state.mode }),
            onRehydrateStorage: () => (state) => {
                if (state) {
                    state.isDark = state.mode === 'dark';
                }
            },
        }
    )
);
