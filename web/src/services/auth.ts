interface Tokens {
  access_token: string;
  refresh_token: string;
  expires_in: string | number;
}

interface AuthHeaders {
  Authorization: string;
  'Content-Type': string;
}

interface FetchOptions extends RequestInit {
  headers?: Record<string, string>;
}

class AuthService {
  private readonly API_URL: string;
  private refreshPromise: Promise<string> | null = null;

  constructor() {
    this.API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000/api';
  }

  getApiUrl() {
    return this.API_URL
  }

  // Initiate Google login
  public googleLogin(): void {
    window.location.href = `${this.API_URL}/auth/google/login`;
  }

  // Handle OAuth callback
  public handleCallback(): boolean {

    const params = new URLSearchParams(window.location.search);
    const accessToken = params.get('access_token');
    const refreshToken = params.get('refresh_token');
    const expiresIn = params.get('expires_in');

    if (accessToken && refreshToken && expiresIn) {
      this.setTokens({ 
        access_token: accessToken, 
        refresh_token: refreshToken, 
        expires_in: expiresIn 
      });
      return true;
    }
    
    return false;
  }

  // Store tokens (consider using httpOnly cookies for better security)
  private setTokens(tokens: Tokens): void {
    console.log("setting tokens", tokens);
    try {
      localStorage.setItem('access_token', tokens.access_token);
      localStorage.setItem('refresh_token', tokens.refresh_token);
      localStorage.setItem('token_expiry', tokens.expires_in.toString());
    } catch (error) {
      console.error('Failed to store tokens:', error);
    }
  }

  public getAccessToken(): string | null {
    return localStorage.getItem('access_token');
  }

  public getRefreshToken(): string | null {
    return localStorage.getItem('refresh_token');
  }

  // Check if token is expired
  public isTokenExpired(): boolean {
    const expiry = localStorage.getItem('token_expiry');
    if (!expiry) return true;
    
    const expiryTime = parseInt(expiry, 10) * 1000;
    // Add a small buffer (5 seconds) to prevent edge cases
    return Date.now() + 5000 > expiryTime;
  }

  // Refresh token with deduplication
  public async refreshToken(): Promise<string> {
    // If a refresh is already in progress, return that promise
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      this.logout();
      throw new Error('No refresh token available');
    }

    this.refreshPromise = (async () => {
      try {
        const response = await fetch(`${this.API_URL}/auth/refresh`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            refresh_token: refreshToken,
          }),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.message || 'Failed to refresh token');
        }

        const tokens: Tokens = await response.json();
        this.setTokens(tokens);
        return tokens.access_token;
      } catch (error) {
        this.logout();
        throw error;
      } finally {
        this.refreshPromise = null;
      }
    })();

    return this.refreshPromise;
  }

  // Logout
  public logout(): void {
    // Clear local storage
    try {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('token_expiry');
    } catch (error) {
      console.error('Failed to clear storage:', error);
    }

    // Call logout endpoint (optional) - don't await
    fetch(`${this.API_URL}/auth/logout`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    }).catch(() => {
      // Silently fail - logout already happened client-side
    });

    // Redirect to login
    window.location.href = '/login';
  }

  // Get auth headers for API calls
  public getAuthHeaders(): AuthHeaders {
    const token = this.getAccessToken();
    return {
      'Authorization': `Bearer ${token || ''}`,
      'Content-Type': 'application/json',
    };
  }

  // Check if user is authenticated
  public isAuthenticated(): boolean {
    return !!this.getAccessToken() && !this.isTokenExpired();
  }

  // Make authenticated API call with automatic token refresh
  public async fetchWithAuth<T = Response>(
    url: string, 
    options: FetchOptions = {}
  ): Promise<T> {
    // Check if token is expired and refresh if needed
    if (this.isTokenExpired()) {
      try {
        await this.refreshToken();
      } catch (error) {
        throw new Error('Session expired. Please login again.');
      }
    }

    let response = await fetch(url, {
      ...options,
      headers: {
        ...this.getAuthHeaders(),
        ...(options.headers || {}),
      },
    });

    // If unauthorized, try to refresh token once
    if (response.status === 401) {
      try {
        await this.refreshToken();
        
        // Retry the request with new token
        response = await fetch(url, {
          ...options,
          headers: {
            ...this.getAuthHeaders(),
            ...(options.headers || {}),
          },
        });
      } catch (error) {
        this.logout();
        throw new Error('Authentication failed. Please login again.');
      }
    }

    return response as unknown as T;
  }

}

// Export singleton instance
export default new AuthService();