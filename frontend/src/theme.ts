import type { ThemeConfig } from 'antd';
import { theme as antdTheme } from 'antd';

// Base tokens shared between light and dark themes
const baseTokens = {
    colorPrimary: '#06b6d4', // Cyan
    colorInfo: '#06b6d4',
    colorSuccess: '#10b981', // Emerald
    colorWarning: '#f59e0b', // Amber
    colorError: '#ef4444',   // Red
    borderRadius: 8,
    wireframe: false,
    fontFamily: "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
};

// Light theme
export const lightTheme: ThemeConfig = {
    token: baseTokens,
    components: {
        Layout: {
            siderBg: '#0f172a', // Slate 900
            triggerBg: '#1e293b', // Slate 800
        },
        Menu: {
            darkItemBg: 'transparent',
            darkItemSelectedBg: 'rgba(6, 182, 212, 0.15)', // Cyan with opacity
            darkItemColor: '#94a3b8', // Slate 400
            darkItemSelectedColor: '#fff',
            darkSubMenuItemBg: 'transparent',
        }
    }
};

// Dark theme
export const darkTheme: ThemeConfig = {
    token: {
        ...baseTokens,
        colorBgContainer: '#1e293b', // Slate 800
        colorBgElevated: '#334155', // Slate 700
        colorBgLayout: '#0f172a', // Slate 900
        colorBgSpotlight: '#334155',
        colorText: '#f1f5f9', // Slate 100
        colorTextSecondary: '#94a3b8', // Slate 400
        colorTextTertiary: '#64748b', // Slate 500
        colorBorder: '#334155', // Slate 700
        colorBorderSecondary: '#1e293b', // Slate 800
    },
    algorithm: antdTheme.darkAlgorithm,
    components: {
        Layout: {
            siderBg: '#020617', // Slate 950
            triggerBg: '#0f172a', // Slate 900
            bodyBg: '#0f172a', // Slate 900
            headerBg: '#1e293b', // Slate 800
        },
        Menu: {
            darkItemBg: 'transparent',
            darkItemSelectedBg: 'rgba(6, 182, 212, 0.2)', // Cyan with opacity
            darkItemColor: '#94a3b8', // Slate 400
            darkItemSelectedColor: '#fff',
            darkSubMenuItemBg: 'transparent',
        },
        Card: {
            colorBgContainer: '#1e293b',
        },
        Table: {
            colorBgContainer: '#1e293b',
            headerBg: '#334155',
        },
        Modal: {
            contentBg: '#1e293b',
            headerBg: '#1e293b',
        },
        Input: {
            colorBgContainer: '#334155',
        },
        Select: {
            colorBgContainer: '#334155',
            colorBgElevated: '#1e293b',
        },
    }
};

// Helper function to get theme based on mode
export const getTheme = (isDark: boolean): ThemeConfig => {
    return isDark ? darkTheme : lightTheme;
};

// Keep default export for backwards compatibility
export const theme = lightTheme;
