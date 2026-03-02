import { useState, useCallback } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useTheme } from '../context/ThemeContext';
import { useLanguage } from '../context/LanguageContext';
import ReferralModal from './ReferralModal';
import TelegramLoginButton from './TelegramLoginButton';
import type { TelegramUser } from './TelegramLoginButton';

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

function NavLink({ to, children, className = '', onClick }: { to: string; children: React.ReactNode; className?: string; onClick?: () => void }) {
  const location = useLocation();
  const isActive = location.pathname === to;
  return (
    <Link
      to={to}
      onClick={onClick}
      className={`text-sm transition-colors whitespace-nowrap ${isActive ? 'text-gray-900 dark:text-white font-medium' : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'} ${className}`}
    >
      {children}
    </Link>
  );
}

export default function Layout() {
  const { user, telegramLogin, logout } = useAuth();
  const navigate = useNavigate();
  const { theme, toggleTheme } = useTheme();
  const { language, setLanguage, t } = useLanguage();
  const [referralOpen, setReferralOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [loginLoading, setLoginLoading] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/');
    setMenuOpen(false);
  };

  const handleTelegramAuth = useCallback(async (tgUser: TelegramUser) => {
    setLoginLoading(true);
    try {
      await telegramLogin(tgUser);
    } catch { /* ignore */ }
    setLoginLoading(false);
  }, [telegramLogin]);

  const handleTelegramError = useCallback(() => {}, []);

  const navLinks = (
    <>
      <NavLink to="/proxies" onClick={() => setMenuOpen(false)}>
        {t.nav.myServices}
      </NavLink>
      <NavLink to="/pricing" onClick={() => setMenuOpen(false)}>
        {t.nav.pricing}
      </NavLink>
      {user && (
        <button
          type="button"
          onClick={() => { setReferralOpen(true); setMenuOpen(false); }}
          className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors whitespace-nowrap text-left"
        >
          {t.nav.referrals}
        </button>
      )}
      {user?.role === 'admin' && (
        <NavLink to="/admin" onClick={() => setMenuOpen(false)}>
          {t.nav.admin}
        </NavLink>
      )}
    </>
  );

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col">
      <ReferralModal isOpen={referralOpen} onClose={() => setReferralOpen(false)} />
      <nav className="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 sticky top-0 z-40">
        <div className="mx-auto max-w-6xl px-3 sm:px-4 py-3 flex items-center justify-between gap-2">
          <div className="flex items-center gap-3 sm:gap-6 min-w-0">
            <Link to="/" className="text-base sm:text-lg font-bold text-gray-900 dark:text-white tracking-tight whitespace-nowrap">
              {t.common.appName}
            </Link>
            <div className="hidden md:flex items-center gap-3 sm:gap-6">
              {navLinks}
            </div>
          </div>
          <div className="flex items-center gap-2 sm:gap-3 shrink-0">
            <button
              type="button"
              onClick={toggleTheme}
              aria-label="Toggle theme"
              className="relative inline-flex items-center w-14 h-7 rounded-full transition-colors focus:outline-none bg-gray-200 dark:bg-gray-700 touch-manipulation shrink-0"
            >
              <span className={`absolute inset-0 flex items-center transition-all duration-200 ${theme === 'dark' ? 'justify-end pr-1' : 'justify-start pl-1'}`}>
                <span className="w-5 h-5 rounded-full bg-white shadow flex items-center justify-center text-gray-600 dark:text-gray-700">
                  {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
                </span>
              </span>
            </button>

            <button
              type="button"
              onClick={() => setLanguage(language === 'ru' ? 'en' : 'ru')}
              aria-label="Toggle language"
              className="relative inline-flex items-center w-14 h-7 rounded-full transition-colors focus:outline-none bg-gray-200 dark:bg-gray-700 touch-manipulation shrink-0"
            >
              <span className={`absolute inset-0 flex items-center transition-all duration-200 ${language === 'en' ? 'justify-end pr-1' : 'justify-start pl-1'}`}>
                <span className="w-5 h-5 rounded-full bg-white shadow flex items-center justify-center text-[10px] font-bold text-gray-700">
                  {language === 'ru' ? 'RU' : 'EN'}
                </span>
              </span>
            </button>

            {user ? (
              <>
                <span className="text-sm text-gray-500 dark:text-gray-400 hidden sm:inline">
                  {user.username}
                  {user.role === 'admin' && (
                    <span className="ml-1.5 text-xs bg-indigo-600/30 text-indigo-400 dark:text-indigo-300 px-1.5 py-0.5 rounded">
                      admin
                    </span>
                  )}
                </span>
                <button
                  onClick={handleLogout}
                  className="hidden md:block text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors whitespace-nowrap"
                >
                  {t.nav.logout}
                </button>
              </>
            ) : (
              <TelegramLoginButton
                label={t.nav.login}
                onAuth={handleTelegramAuth}
                onError={handleTelegramError}
                disabled={loginLoading}
                className="hidden md:flex items-center gap-1.5 bg-[#54a9eb] hover:bg-[#4a96d2] disabled:opacity-50 text-white text-sm font-medium rounded-md px-3 py-1.5 transition-colors touch-manipulation whitespace-nowrap"
              />
            )}
            <button
              type="button"
              onClick={() => setMenuOpen((o) => !o)}
              className="md:hidden p-2 -mr-2 text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors touch-manipulation"
              aria-label="Menu"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                {menuOpen ? (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                ) : (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                )}
              </svg>
            </button>
          </div>
        </div>
        {menuOpen && (
          <div className="md:hidden border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 px-4 py-3 flex flex-col gap-3">
            {navLinks}
            <div className="flex items-center justify-between pt-2 border-t border-gray-200 dark:border-gray-800">
              {user ? (
                <>
                  <span className="text-sm text-gray-500 dark:text-gray-400">{user.username}</span>
                  <button onClick={handleLogout} className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors py-2 touch-manipulation">
                    {t.nav.logout}
                  </button>
                </>
              ) : (
                <TelegramLoginButton
                  label={t.nav.login}
                  onAuth={handleTelegramAuth}
                  onError={handleTelegramError}
                  disabled={loginLoading}
                  className="flex items-center gap-1.5 bg-[#54a9eb] hover:bg-[#4a96d2] disabled:opacity-50 text-white text-sm font-medium rounded-md px-3 py-1.5 transition-colors touch-manipulation"
                />
              )}
            </div>
          </div>
        )}
      </nav>
      <main className="mx-auto max-w-6xl w-full px-3 sm:px-4 py-4 sm:py-8 flex-1">
        <Outlet />
      </main>
      <footer className="border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 py-4">
        <div className="mx-auto max-w-6xl px-4 text-center text-xs text-gray-400 dark:text-gray-500">
          {t.footer.feedback}{' '}
          <a
            href="https://t.me/oddwallet"
            target="_blank"
            rel="noopener noreferrer"
            className="text-indigo-500 dark:text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-300 transition-colors"
          >
            @oddwallet
          </a>
        </div>
      </footer>
    </div>
  );
}
