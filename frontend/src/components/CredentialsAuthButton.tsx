import { useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { authApi } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';

type Mode = 'login' | 'register';

interface CredentialsAuthButtonProps {
  className?: string;
  onSuccess?: () => void;
}

export default function CredentialsAuthButton({ className, onSuccess }: CredentialsAuthButtonProps) {
  const { setAuthToken } = useAuth();
  const { language } = useLanguage();

  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<Mode>('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const labels = useMemo(() => {
    if (language === 'ru') {
      return {
        open: 'Логин и пароль',
        title: mode === 'login' ? 'Вход' : 'Регистрация',
        username: 'Логин',
        password: 'Пароль',
        submit: mode === 'login' ? 'Войти' : 'Зарегистрироваться',
        switchText: mode === 'login' ? 'Нет аккаунта?' : 'Уже есть аккаунт?',
        switchAction: mode === 'login' ? 'Создать' : 'Войти',
        close: 'Закрыть',
      };
    }

    return {
      open: 'Login & password',
      title: mode === 'login' ? 'Sign in' : 'Sign up',
      username: 'Username',
      password: 'Password',
      submit: mode === 'login' ? 'Sign in' : 'Sign up',
      switchText: mode === 'login' ? "Don't have an account?" : 'Already have an account?',
      switchAction: mode === 'login' ? 'Create one' : 'Sign in',
      close: 'Close',
    };
  }, [language, mode]);

  const resetForm = () => {
    setUsername('');
    setPassword('');
    setError('');
    setSubmitting(false);
  };

  const close = () => {
    setOpen(false);
    resetForm();
  };

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (submitting) return;

    setError('');
    setSubmitting(true);
    try {
      const cleanUsername = username.trim().toLowerCase();
      const cleanPassword = password.trim();
      const response = mode === 'login'
        ? await authApi.login(cleanUsername, cleanPassword)
        : await authApi.register(cleanUsername, cleanPassword);

      setAuthToken(response.data.token);
      close();
      onSuccess?.();
    } catch (err: any) {
      const apiError = err?.response?.data?.error;
      setError(typeof apiError === 'string' ? apiError : 'Request failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={className ?? 'bg-gray-900 hover:bg-gray-800 text-white text-sm font-medium rounded-md px-3 py-1.5 transition-colors'}
      >
        {labels.open}
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={close} />
          <div className="relative w-full max-w-sm rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 shadow-xl p-4">
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">{labels.title}</h3>
            <form onSubmit={onSubmit} className="mt-4 space-y-3">
              <label className="block">
                <span className="mb-1 block text-xs text-gray-500 dark:text-gray-400">{labels.username}</span>
                <input
                  type="text"
                  autoComplete="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-white outline-none focus:ring-2 focus:ring-indigo-500"
                  required
                />
              </label>

              <label className="block">
                <span className="mb-1 block text-xs text-gray-500 dark:text-gray-400">{labels.password}</span>
                <input
                  type="password"
                  autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-white outline-none focus:ring-2 focus:ring-indigo-500"
                  required
                />
              </label>

              {error && (
                <p className="text-xs text-red-500">{error}</p>
              )}

              <button
                type="submit"
                disabled={submitting}
                className="w-full rounded-lg bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60 text-white text-sm font-medium py-2 transition-colors"
              >
                {submitting ? '...' : labels.submit}
              </button>
            </form>

            <div className="mt-3 text-center text-xs text-gray-500 dark:text-gray-400">
              {labels.switchText}{' '}
              <button
                type="button"
                onClick={() => {
                  setMode(mode === 'login' ? 'register' : 'login');
                  setError('');
                }}
                className="text-indigo-500 hover:text-indigo-400 transition-colors"
              >
                {labels.switchAction}
              </button>
            </div>

            <button
              type="button"
              onClick={close}
              className="mt-3 w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-transparent text-gray-700 dark:text-gray-200 text-sm py-2 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              {labels.close}
            </button>
          </div>
        </div>
      )}
    </>
  );
}
