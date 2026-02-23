import axios from "axios";
import AuthService from "../services/auth";

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
  withCredentials: true,
  headers: {
    "Content-Type": "application/json",
  },
});

// REQUEST Interceptor: Add the token to every call
api.interceptors.request.use(
  async (config) => {
    // Optional: Auto-refresh if expired before even sending the request
    if (AuthService.isTokenExpired() && AuthService.getRefreshToken()) {
      try {
        await AuthService.refreshToken();
      } catch (err) {
        console.error("Could not refresh token before request", err);
      }
    }

    const token = AuthService.getAccessToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

// RESPONSE Interceptor: Handle 401s and retry
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    // If 401 and we haven't tried to refresh yet
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        const newToken = await AuthService.refreshToken();

        // Update the header and retry the original request
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      } catch (refreshError) {
        // AuthService.logout();
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  },
);
