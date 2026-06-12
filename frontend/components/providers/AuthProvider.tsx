"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import axios, { AxiosInstance } from "axios";
import { useRouter, usePathname } from "next/navigation";
import toast from "react-hot-toast";

interface User {
  id: string;
  name: string;
  email: string;
  avatar: string;
  role: string;
  theme: string;
  timezone: string;
  notify_email: boolean;
  notify_in_app: boolean;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  isAuthenticated: boolean;
  login: () => void;
  logout: () => Promise<void>;
  updateUser: (updates: Partial<User>) => void;
  api: AxiosInstance;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Axios instance configured with backend URL
export const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api",
  headers: {
    "Content-Type": "application/json",
  },
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  // Set auth headers on interceptor
  useEffect(() => {
    const requestInterceptor = api.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem("accessToken");
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    const responseInterceptor = api.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;
        if (
          error.response?.status === 401 &&
          !originalRequest._retry &&
          originalRequest.url !== "/auth/refresh"
        ) {
          originalRequest._retry = true;
          const refreshToken = localStorage.getItem("refreshToken");
          if (refreshToken) {
            try {
              const res = await axios.post(
                `${api.defaults.baseURL}/auth/refresh`,
                { refresh_token: refreshToken }
              );
              const newAccessToken = res.data.access_token;
              localStorage.setItem("accessToken", newAccessToken);
              originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
              return api(originalRequest);
            } catch (refreshErr) {
              // Refresh token failed, clean up and redirect
              localStorage.removeItem("accessToken");
              localStorage.removeItem("refreshToken");
              setUser(null);
              toast.error("Session expired. Please log in again.");
              router.push("/login");
            }
          }
        }
        return Promise.reject(error);
      }
    );

    return () => {
      api.interceptors.request.eject(requestInterceptor);
      api.interceptors.response.eject(responseInterceptor);
    };
  }, [router]);

  // Load user on mount
  useEffect(() => {
    async function loadUser() {
      const token = localStorage.getItem("accessToken");
      if (!token) {
        setLoading(false);
        // If not logged in and not on landing/login/auth/invite paths, redirect to login
        const isPublicPath =
          pathname === "/" ||
          pathname === "/login" ||
          pathname.startsWith("/auth/callback") ||
          pathname.startsWith("/invite");
        if (!isPublicPath) {
          router.push("/login");
        }
        return;
      }

      try {
        const res = await api.get("/auth/me");
        setUser(res.data);
      } catch (err) {
        console.error("Failed to load user", err);
        localStorage.removeItem("accessToken");
        localStorage.removeItem("refreshToken");
      } finally {
        setLoading(false);
      }
    }
    loadUser();
  }, [pathname, router]);

  const login = () => {
    // Redirect to backend Google Login endpoint
    window.location.href = `${
      process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api"
    }/auth/google`;
  };

  const logout = async () => {
    try {
      await api.post("/auth/logout");
    } catch (err) {
      console.error("Logout request failed", err);
    } finally {
      localStorage.removeItem("accessToken");
      localStorage.removeItem("refreshToken");
      setUser(null);
      toast.success("Logged out successfully");
      router.push("/login");
    }
  };

  const updateUser = (updates: Partial<User>) => {
    setUser((prev) => (prev ? { ...prev, ...updates } : null));
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAuthenticated: !!user,
        login,
        logout,
        updateUser,
        api,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
