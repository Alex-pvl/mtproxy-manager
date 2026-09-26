import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const wasLoggedIn = !!localStorage.getItem('token');
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      if (wasLoggedIn) {
        window.location.href = '/';
      }
    }
    return Promise.reject(error);
  }
);

export interface Subscription {
  active: boolean;
  plan_id?: string;
  plan_name?: string;
  expires_at?: string;
}

export interface User {
  id: number;
  username: string;
  role: 'user' | 'admin';
  max_proxies: number;
  created_at: string;
  subscription?: Subscription;
}

export interface UserWithCount extends User {
  proxy_count: number;
}

// A connection: one client in the 3x-ui panel.
export interface Proxy {
  id: number;
  user_id: number;
  name: string;
  link_sub?: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: {
    id: number;
    username: string;
    role: string;
  };
}

export const authApi = {
  me: () => api.get<User>('/auth/me'),
  login: (username: string, password: string) =>
    api.post<AuthResponse>('/auth/login', { username, password }),
  register: (username: string, password: string) =>
    api.post<AuthResponse>('/auth/register', { username, password }),
  webappLogin: (initData: string, ref?: string) =>
    api.post<AuthResponse>('/auth/webapp', { init_data: initData, ref: ref || undefined }),
};

export interface ReferralInfo {
  referral_link: string;
  invited_count: number;
  bonus_days_received: number;
}

export const referralApi = {
  get: (miniApp = false) =>
    api.get<ReferralInfo>('/referral', {
      params: miniApp ? { miniapp: '1' } : undefined,
    }),
};

export const proxyApi = {
  list: () => api.get<Proxy[]>('/proxies'),
  create: () => api.post<Proxy>('/proxies', {}),
  delete: (id: number) => api.delete(`/proxies/${id}`),
};

export const adminApi = {
  listUsers: () => api.get<UserWithCount[]>('/admin/users'),
  updateUser: (id: number, data: { role?: string; max_proxies?: number }) =>
    api.put<User>(`/admin/users/${id}`, data),
  deleteUser: (id: number) => api.delete(`/admin/users/${id}`),
  listProxies: () => api.get<Proxy[]>('/admin/proxies'),
  deleteProxy: (id: number) => api.delete(`/admin/proxies/${id}`),
};

export interface Plan {
  id: string;
  name: string;
  duration_days: number;
  price: number;
  price_label: string;
  price_usd_label?: string;
  original_price_label?: string;
  discount_percent?: number;
  per_month: string;
  max_proxies: number;
  stars_price?: number;
  ton_amount?: string;
}

export const paymentApi = {
  listPlans: () => api.get<Plan[]>('/plans'),
  createPayment: (planId: string, source?: 'web' | 'tg') =>
    api.post<{ payment_url: string }>('/payments/create', { plan_id: planId, source: source || undefined }),
  createSbpPayment: (planId: string, source?: 'web' | 'tg') =>
    api.post<{ payment_url: string }>('/payments/sbp/create', { plan_id: planId, source: source || undefined }),
  createStarsPayment: (planId: string) =>
    api.post<{ invoice_link: string }>('/payments/stars/create', { plan_id: planId }),
  createTonPayment: (planId: string) =>
    api.post<{ address: string; amount: string; comment: string }>('/payments/ton/create', { plan_id: planId }),
  checkPendingPayments: () =>
    api.post<{ updated: boolean }>('/payments/check-pending'),
  getSubscription: () => api.get<Subscription>('/subscription'),
};

/** Error text from the backend's {"error": "..."} body, or the fallback. */
export function apiError(err: unknown, fallback: string): string {
  return axios.isAxiosError<{ error?: string }>(err) ? err.response?.data?.error || fallback : fallback;
}

export default api;
