import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import type { ReactNode } from 'react';
import { authApi } from '../api/client';
import type { User } from '../api/client';

interface AuthState {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string, ref?: string) => Promise<void>;
  telegramLogin: (idToken: string, ref?: string) => Promise<void>;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (token) {
      authApi.me()
        .then((res) => setUser(res.data))
        .catch(() => {
          localStorage.removeItem('token');
          setToken(null);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [token]);

  const login = async (username: string, password: string) => {
    const res = await authApi.login(username, password);
    localStorage.setItem('token', res.data.token);
    setToken(res.data.token);
    const me = await authApi.me();
    setUser(me.data);
  };

  const register = async (username: string, password: string, ref?: string) => {
    const res = await authApi.register(username, password, ref);
    localStorage.setItem('token', res.data.token);
    setToken(res.data.token);
    const me = await authApi.me();
    setUser(me.data);
  };

  const telegramLogin = async (idToken: string, ref?: string) => {
    const res = await authApi.telegramLogin(idToken, ref);
    localStorage.setItem('token', res.data.token);
    setToken(res.data.token);
    const me = await authApi.me();
    setUser(me.data);
  };

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
    <AuthContext.Provider value={{ user, token, isLoading, login, register, telegramLogin, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
