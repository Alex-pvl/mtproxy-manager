import { createContext, useContext, useState, useEffect, useCallback, useRef } from 'react';
import type { ReactNode } from 'react';
import { authApi } from '../api/client';
import type { User } from '../api/client';

declare global {
  interface Window {
    Telegram?: {
      WebApp?: {
        initData: string;
        initDataUnsafe: {
          user?: {
            id: number;
            first_name: string;
            last_name?: string;
            username?: string;
            photo_url?: string;
          };
          start_param?: string;
        };
        openInvoice: (url: string, callback: (status: string) => void) => void;
        ready: () => void;
        expand: () => void;
        close: () => void;
      };
    };
  }
}

interface AuthState {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  isMiniApp: boolean;
  telegramPhotoUrl: string | null;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

function getTelegramWebApp() {
  return window.Telegram?.WebApp;
}

function getInitialToken(): string | null {
  const params = new URLSearchParams(window.location.search);
  const urlToken = params.get('token');
  if (urlToken) {
    localStorage.setItem('token', urlToken);
    return urlToken;
  }
  return localStorage.getItem('token');
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(getInitialToken);
  const [isLoading, setIsLoading] = useState(true);
  const webappLoginAttempted = useRef(false);

  const webApp = getTelegramWebApp();
  const isMiniApp = !!(webApp && webApp.initData);
  const telegramPhotoUrl = isMiniApp
    ? (webApp!.initDataUnsafe.user?.photo_url ?? null)
    : null;

  // Signal to Telegram that the Mini App is ready
  useEffect(() => {
    if (isMiniApp) {
      webApp!.ready();
      webApp!.expand();
    }
  }, [isMiniApp, webApp]);

  // Clean up token / auth_error from URL after OIDC redirect
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.has('token') || params.has('auth_error')) {
      const authError = params.get('auth_error');
      if (authError) {
        console.error('Telegram auth error:', authError);
      }
      params.delete('token');
      params.delete('auth_error');
      const qs = params.toString();
      window.history.replaceState(null, '', window.location.pathname + (qs ? '?' + qs : ''));
    }
  }, []);

  // Auto-login via Mini App initData
  useEffect(() => {
    if (!isMiniApp || token || webappLoginAttempted.current) return;
    webappLoginAttempted.current = true;

    const ref = webApp!.initDataUnsafe.start_param || undefined;

    authApi.webappLogin(webApp!.initData, ref)
      .then((res) => {
        localStorage.setItem('token', res.data.token);
        setToken(res.data.token);
      })
      .catch((err) => {
        console.error('Mini App auto-login failed:', err);
        setIsLoading(false);
      });
  }, [isMiniApp, token, webApp]);

  // Validate token and fetch user
  useEffect(() => {
    if (token) {
      authApi.me()
        .then((res) => setUser(res.data))
        .catch(() => {
          localStorage.removeItem('token');
          setToken(null);
        })
        .finally(() => setIsLoading(false));
    } else if (!isMiniApp || webappLoginAttempted.current) {
      setIsLoading(false);
    }
  }, [token, isMiniApp]);

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setUser(null);
  };

  const refreshUser = useCallback(async () => {
    if (!token) return;
    try {
      const res = await authApi.me();
      setUser(res.data);
    } catch { /* ignore */ }
  }, [token]);

  return (
    <AuthContext.Provider value={{ user, token, isLoading, isMiniApp, telegramPhotoUrl, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
