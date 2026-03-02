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
        window.location.href = '/login';
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

export interface Proxy {
  id: number;
  user_id: number;
  port: number;
  domain: string;
  secret: string;
  container_id: string;
  container_name: string;
  status: 'running' | 'stopped' | 'error';
  created_at: string;
  link?: string;
  link_socks5?: string;
  link_vless?: string;
  socks5_port?: number;
  socks5_user?: string;
  socks5_pass?: string;
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
  telegramLogin: (idToken: string, ref?: string) =>
    api.post<AuthResponse>('/auth/telegram', { id_token: idToken, ref: ref || undefined }),
  me: () => api.get<User>('/auth/me'),
};

export interface ReferralInfo {
  referral_link: string;
  invited_count: number;
  bonus_days_received: number;
}

export const referralApi = {
  get: () => api.get<ReferralInfo>('/referral'),
};

export const proxyApi = {
  list: () => api.get<Proxy[]>('/proxies'),
  create: (domain: string, port?: number) =>
    api.post<Proxy>('/proxies', { domain, port: port || undefined }),
  start: (id: number) => api.post<Proxy>(`/proxies/${id}/start`),
  stop: (id: number) => api.post<Proxy>(`/proxies/${id}/stop`),
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
}

export const paymentApi = {
  listPlans: () => api.get<Plan[]>('/plans'),
  createPayment: (planId: string) =>
    api.post<{ payment_url: string }>('/payments/create', { plan_id: planId }),
  checkPendingPayments: () =>
    api.post<{ updated: boolean }>('/payments/check-pending'),
  getSubscription: () => api.get<Subscription>('/subscription'),
};

export default api;
