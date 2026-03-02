import { createContext, useState, useContext, useEffect } from "react";
import type { ReactNode } from "react";
import authService from "../services/auth";
import { useQuery, useQueryClient } from "@tanstack/react-query";

export interface User {
  id: string;
  email: string;
  name: string;
  lastName: string;
  picture?: string;
  storageUsed: number;
  storageLimit: number;
}

interface AuthContextType {
  user: User | null | undefined;
  loading: boolean;
  login: () => void;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const queryClient = useQueryClient();

  const { data: user, isLoading: loading } = useQuery({
    queryKey: ["authUser"],
    queryFn: async () => {
      const token = authService.getAccessToken();
      if (!token || authService.isTokenExpired()) {
        return null;
      }

      const response = await authService.fetchWithAuth(
        `${authService.getApiUrl()}/users/profile`,
      );

      if (!response.ok) throw new Error("Failed to fetch user profile");
      return (await response.json()) as User;
    },
    staleTime: 60000,
    // retry: false,
  });

  const login = (): void => {
    authService.googleLogin();
  };

  const logout = (): void => {
    authService.logout();
    // Clear the query cache so the next user doesn't see old data
    queryClient.setQueryData(["authUser"], null);
    queryClient.removeQueries();
  };

  const value: AuthContextType = {
    user,
    loading,
    login,
    logout,
    isAuthenticated: !!user,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};
