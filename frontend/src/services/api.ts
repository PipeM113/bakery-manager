import axios from "axios";

const api = axios.create({
  // The localhost default only exists in dev; production builds fail without VITE_API_URL.
  baseURL: import.meta.env.VITE_API_URL || (import.meta.env.DEV ? "http://localhost:8080" : undefined),
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const isLoginEndpoint = err.config?.url?.includes("/auth/login");
    if (err.response?.status === 401 && !isLoginEndpoint) {
      localStorage.removeItem("token");
      window.location.href = "/login";
    }
    return Promise.reject(err);
  }
);

export default api;