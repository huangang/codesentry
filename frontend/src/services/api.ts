import axios from 'axios';
import { useAuthStore } from '../stores/authStore';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';
const REFRESH_PATH = '/auth/refresh';

let refreshTokenRequest: Promise<string> | null = null;
let redirectingToLogin = false;

type RetriableRequestConfig = {
  url?: string;
  headers?: Record<string, string>;
  _retry?: boolean;
};

function shouldSkipRefresh(config?: RetriableRequestConfig): boolean {
  if (!config?.url) {
    return true;
  }
  const path = config.url;
  return path.includes('/auth/login') || path.includes(REFRESH_PATH) || !!config._retry;
}

function isLoginRequest(config?: RetriableRequestConfig): boolean {
  return !!config?.url && config.url.includes('/auth/login');
}

function clearAuthAndRedirect(): void {
  const { logout } = useAuthStore.getState();
  logout();
  localStorage.removeItem('auth-storage');
  if (!redirectingToLogin) {
    redirectingToLogin = true;
    window.location.href = '/login';
  }
}

async function refreshAccessToken(): Promise<string> {
  if (!refreshTokenRequest) {
    refreshTokenRequest = axios
      .post(
        `${API_BASE_URL}${REFRESH_PATH}`,
        {},
        {
          withCredentials: true,
          headers: {
            'Content-Type': 'application/json',
          },
        }
      )
      .then((response) => {
        const body = response.data as unknown;
        const token =
          typeof body === 'object' && body !== null &&
          'data' in body && typeof (body as { data?: unknown }).data === 'object' && (body as { data?: { token?: unknown } }).data !== null
            ? (body as { data: { token?: string } }).data.token
            : undefined;
        if (!token) {
          throw new Error('refresh token response missing access token');
        }
        useAuthStore.getState().setToken(token);
        return token;
      })
      .finally(() => {
        refreshTokenRequest = null;
      });
  }
  return refreshTokenRequest;
}

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to unwrap envelope and handle errors
api.interceptors.response.use(
  (response) => {
    // Unwrap {code, data, message} envelope from response.Success()
    const body = response.data;
    if (body && typeof body === 'object' && 'code' in body && 'data' in body) {
      response.data = body.data;
    }
    return response;
  },
  async (error) => {
    const status = error.response?.status as number | undefined;
    const originalRequest = error.config as RetriableRequestConfig | undefined;

    if (status === 401) {
      if (shouldSkipRefresh(originalRequest)) {
        if (!isLoginRequest(originalRequest)) {
          clearAuthAndRedirect();
        }
        return Promise.reject(error);
      }

      try {
        const nextToken = await refreshAccessToken();
        if (!originalRequest) {
          return Promise.reject(error);
        }
        originalRequest._retry = true;
        originalRequest.headers = originalRequest.headers || {};
        originalRequest.headers.Authorization = `Bearer ${nextToken}`;
        return api(originalRequest);
      } catch {
        clearAuthAndRedirect();
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  }
);

export default api;
