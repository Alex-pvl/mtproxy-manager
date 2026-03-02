import { useState, useCallback } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { useTheme } from '../context/ThemeContext';
import TelegramLoginButton from '../components/TelegramLoginButton';
import type { TelegramUser } from '../components/TelegramLoginButton';

function SunIcon() {
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <circle cx="12" cy="12" r="5" />
      <line x1="12" y1="1" x2="12" y2="3" />
      <line x1="12" y1="21" x2="12" y2="23" />
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
      <line x1="1" y1="12" x2="3" y2="12" />
      <line x1="21" y1="12" x2="23" y2="12" />
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  );
}

export default function Login() {
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { telegramLogin } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const refFromUrl = searchParams.get('ref') ?? undefined;
  const { t, language, setLanguage } = useLanguage();
  const { theme, toggleTheme } = useTheme();

  const handleTelegramAuth = useCallback(async (user: TelegramUser) => {
    setError('');
    setLoading(true);
    try {
      await telegramLogin(user, refFromUrl);
      navigate('/');
    } catch {
      setError(t.login.telegramError);
    } finally {
      setLoading(false);
    }
  }, [telegramLogin, navigate, refFromUrl, t]);

  const handleTelegramError = useCallback((err: string) => {
    setError(err || t.login.telegramError);
  }, [t]);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex flex-col items-center justify-center px-4">
      <div className="absolute top-4 right-4 flex items-center gap-2">
        <button
          type="button"
          onClick={toggleTheme}
          className="p-1.5 rounded-md text-gray-400 dark:text-gray-400 hover:text-gray-700 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          aria-label="Toggle theme"
        >
          {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
        </button>
        <button
          type="button"
          onClick={() => setLanguage(language === 'ru' ? 'en' : 'ru')}
          className="text-xs font-semibold px-2 py-1 rounded-md border border-gray-200 dark:border-gray-700 text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
        >
          {language === 'ru' ? 'EN' : 'RU'}
        </button>
      </div>

      <div className="w-full max-w-sm flex-1 flex flex-col justify-center">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white text-center mb-8">{t.common.appName}</h1>
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-6 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white text-center">{t.login.title}</h2>

          {error && (
            <div className="bg-red-500/10 border border-red-500/20 text-red-500 dark:text-red-400 text-sm rounded px-3 py-2">
              {error}
            </div>
          )}

          <TelegramLoginButton
            label={t.login.telegramLogin}
            onAuth={handleTelegramAuth}
            onError={handleTelegramError}
            disabled={loading}
          />

          {loading && (
            <p className="text-center text-sm text-gray-400 dark:text-gray-500">{t.login.loading}</p>
          )}
        </div>
      </div>
      <footer className="py-4 text-center text-xs text-gray-400 dark:text-gray-500" />
    </div>
  );
}
