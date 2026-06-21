import axios from 'axios';
import { useAuthStore } from '@/stores/auth';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
  withCredentials: true, // Crucial for sending/receiving HTTP-only cookies
});

let refreshPromise: Promise<void> | null = null;

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      if (originalRequest.url === '/auth/refresh' || originalRequest.url === '/auth/login') {
        useAuthStore.getState().clear();
        if (window.location.pathname !== '/login') {
          window.location.assign('/login');
        }
        return Promise.reject(error);
      }

      originalRequest._retry = true;

      try {
        if (!refreshPromise) {
          refreshPromise = api.post('/auth/refresh').then(() => {});
        }
        await refreshPromise;
        return api(originalRequest);
      } catch (refreshError) {
        useAuthStore.getState().clear();
        if (window.location.pathname !== '/login') {
          window.location.assign('/login');
        }
        return Promise.reject(refreshError);
      } finally {
        refreshPromise = null;
      }
    }

    if (error.response?.status === 403 && error.response?.data?.error?.code === 'password_change_required') {
      if (window.location.pathname !== '/change-password') {
        window.location.assign('/change-password');
      }
      return Promise.reject(error);
    }

    return Promise.reject(error);
  }
);
