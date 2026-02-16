import { createContext, useState, useContext, useEffect } from 'react';
import type { ReactNode} from 'react';
import authService from '../services/auth';

interface User {
  id: string;
  email: string;
  name?: string;
  picture?: string;
}

interface AuthContextType {
  user: User | null;
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
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    const token = authService.getAccessToken();
    if (token && !authService.isTokenExpired()) {
      fetchUserProfile();
    } else {
      setLoading(false);
    }
  }, []);

  const fetchUserProfile = async (): Promise<void> => {
    try {
      const response = await authService.fetchWithAuth(
        `${authService.getApiUrl()}/profile`
      );

      console.log(response);
      
      if (!response.ok) {
        throw new Error('Failed to fetch user profile');
      }
      
      const userData: User = await response.json();
      setUser(userData);
    } catch (error) {
      console.error('Failed to fetch user profile:', error);
      authService.logout();
    } finally {
      setLoading(false);
    }
  };

  const login = (): void => {
    authService.googleLogin();
  };

  const logout = (): void => {
    authService.logout();
    setUser(null);
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
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};