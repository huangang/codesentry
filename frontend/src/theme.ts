import type { ThemeConfig } from 'antd';

export const theme: ThemeConfig = {
    token: {
        colorPrimary: '#06b6d4', // Cyan
        colorInfo: '#06b6d4',
        colorSuccess: '#10b981', // Emerald
        colorWarning: '#f59e0b', // Amber
        colorError: '#ef4444',   // Red
        borderRadius: 8,
        wireframe: false,
        fontFamily: "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    },
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
