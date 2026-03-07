import { useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import { useAuth } from '../context/AuthContext';
import { useTheme } from '../context/ThemeContext';
import { useLanguage } from '../context/LanguageContext';
import { useTonBalance } from '../hooks/useTonBalance';
import ReferralModal from './ReferralModal';
import TelegramLoginButton from './TelegramLoginButton';

// ─── Theme icons ──────────────────────────────────────────────────────────────

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

// ─── Bottom-bar navigation icons ──────────────────────────────────────────────

function HomeNavIcon() {
  return (
    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
    </svg>
  );
}

function ServicesNavIcon() {
  return (
    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.14 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
    </svg>
  );
}

function PricingNavIcon() {
  return (
    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
    </svg>
  );
}

function ProfileNavIcon() {
  return (
    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
    </svg>
  );
}

function AdminNavIcon() {
  return (
    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
  );
}

// ─── Shared NavLink (desktop layout) ──────────────────────────────────────────

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

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatSubscriptionDate(dateStr: string, locale: string): string {
  return new Date(dateStr).toLocaleDateString(locale, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

// ─── Main Layout ──────────────────────────────────────────────────────────────

export default function Layout() {
  const { user, logout, isMiniApp, telegramPhotoUrl } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, toggleTheme } = useTheme();
  const { language, setLanguage, t } = useLanguage();
  const [referralOpen, setReferralOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();
  const tonBalance = useTonBalance();

  const handleLogout = () => {
    logout();
    navigate('/');
    setMenuOpen(false);
  };

  const themeToggle = (
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
  );

  const langToggle = (
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
  );

  // ── Mini App layout ────────────────────────────────────────────────────────
  if (isMiniApp) {
    const locale = language === 'ru' ? 'ru-RU' : 'en-US';

    const subText = user?.subscription?.active && user.subscription.expires_at
      ? `${t.proxies.subscriptionUntil} ${formatSubscriptionDate(user.subscription.expires_at, locale)}`
      : t.miniApp.noSubscription;

    type BottomItem = { to: string; icon: React.ReactNode; ariaLabel: string };

    const bottomItems: BottomItem[] = [
      { to: '/', icon: <HomeNavIcon />, ariaLabel: 'Home' },
      { to: '/proxies', icon: <ServicesNavIcon />, ariaLabel: t.nav.myServices },
      { to: '/pricing', icon: <PricingNavIcon />, ariaLabel: t.nav.pricing },
      { to: '/profile', icon: <ProfileNavIcon />, ariaLabel: t.profile.title },
    ];

    if (user?.role === 'admin') {
      bottomItems.push({ to: '/admin', icon: <AdminNavIcon />, ariaLabel: t.nav.admin });
    }

    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col">
        {/* ── Mini App header ── */}
        <header className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 px-4 py-3 sticky top-0 z-40">
          <div className="flex items-center justify-between gap-3">

            {/* Avatar + name + subscription — tapping navigates to /profile */}
            <button
              type="button"
              onClick={() => navigate('/profile')}
              className="flex items-center gap-3 min-w-0 flex-1 text-left touch-manipulation"
            >
              {user ? (
                <>
                  {telegramPhotoUrl ? (
                    <img
                      src={telegramPhotoUrl}
                      alt={user.username}
                      className="w-10 h-10 rounded-full object-cover shrink-0"
                    />
                  ) : (
                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-indigo-700 flex items-center justify-center text-white font-semibold text-base shrink-0 select-none">
                      {user.username.charAt(0).toUpperCase()}
                    </div>
                  )}
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <span className="font-semibold text-sm text-gray-900 dark:text-white truncate">
                        {user.username}
                      </span>
                      {user.role === 'admin' && (
                        <span className="text-xs bg-indigo-600/30 text-indigo-400 dark:text-indigo-300 px-1.5 py-0.5 rounded shrink-0">
                          admin
                        </span>
                      )}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
                      {subText}
                    </div>
                  </div>
                </>
              ) : (
                <span className="font-bold text-base text-gray-900 dark:text-white">
                  {t.common.appName}
                </span>
              )}
            </button>

            {/* TON Connect: toncoin.jpg icon + balance */}
            <button
              type="button"
              onClick={() => tonConnectUI.openModal()}
              aria-label="TON wallet"
              className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-xs font-medium transition-colors touch-manipulation shrink-0 ${
                wallet
                  ? 'bg-sky-500/15 text-sky-600 dark:text-sky-400'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
              }`}
            >
              <img src="/toncoin.jpg" alt="TON" className="w-5 h-5 rounded-full object-cover shrink-0" />
              <span className="tabular-nums whitespace-nowrap">
                {wallet
                  ? (tonBalance !== null ? `${tonBalance} TON` : '...')
                  : 'TON'}
              </span>
            </button>
          </div>
        </header>

        {/* ── Page content ── */}
        <main
          className="w-full flex-1 px-3 py-4"
          style={{ paddingBottom: 'calc(4.5rem + env(safe-area-inset-bottom))' }}
        >
          <Outlet />
        </main>

        {/* ── Bottom navigation bar ── */}
        <nav
          className="fixed bottom-0 left-0 right-0 z-40 bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-800"
          style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}
        >
          <div className="flex items-center justify-around h-16">
            {bottomItems.map((item, idx) => {
              const isActive = item.to === '/'
                ? location.pathname === '/'
                : location.pathname.startsWith(item.to);

              return (
                <Link
                  key={idx}
                  to={item.to}
                  aria-label={item.ariaLabel}
                  className={`flex items-center justify-center w-full h-full transition-colors touch-manipulation ${
                    isActive
                      ? 'text-indigo-600 dark:text-indigo-400'
                      : 'text-gray-400 dark:text-gray-500 active:text-gray-600 dark:active:text-gray-300'
                  }`}
                >
                  {item.icon}
                </Link>
              );
            })}
          </div>
        </nav>
      </div>
    );
  }

  // ── Standard (desktop/web) layout ─────────────────────────────────────────

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
            {/* TON Connect widget */}
            <button
              type="button"
              onClick={() => tonConnectUI.openModal()}
              aria-label="TON wallet"
              className={`hidden sm:flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-xs font-medium transition-colors touch-manipulation shrink-0 ${
                wallet
                  ? 'bg-sky-500/15 text-sky-600 dark:text-sky-400 hover:bg-sky-500/25 dark:hover:bg-sky-500/25'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'
              }`}
            >
              <img src="/toncoin.jpg" alt="TON" className="w-5 h-5 rounded-full object-cover shrink-0" />
              <span className="tabular-nums whitespace-nowrap">
                {wallet
                  ? (tonBalance !== null ? `${tonBalance} TON` : '...')
                  : 'TON'}
              </span>
            </button>

            {themeToggle}
            {langToggle}

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
            <button
              type="button"
              onClick={() => { tonConnectUI.openModal(); setMenuOpen(false); }}
              className={`sm:hidden flex items-center gap-2 w-full py-2 rounded-lg text-left transition-colors ${
                wallet
                  ? 'bg-sky-500/15 text-sky-600 dark:text-sky-400'
                  : 'text-gray-500 dark:text-gray-400'
              }`}
            >
              <img src="/toncoin.jpg" alt="TON" className="w-5 h-5 rounded-full object-cover" />
              <span className="text-sm font-medium tabular-nums whitespace-nowrap">
                {wallet ? (tonBalance !== null ? `${tonBalance} TON` : 'TON') : t.payment.tonNotConnected}
              </span>
            </button>
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
