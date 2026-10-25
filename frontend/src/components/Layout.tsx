import { useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { useTonBalance } from '../hooks/useTonBalance';
import ReferralModal from './ReferralModal';
import TelegramLoginButton from './TelegramLoginButton';
import CredentialsAuthButton from './CredentialsAuthButton';
import { TonWalletSheet } from './TonWalletSheet';
import { ThemeToggle, LangToggle } from './Toggles';
import { PRIVACY_URL, SUPPORT_URL, TERMS_URL } from '../legal';

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
  const { language, t } = useLanguage();
  const [referralOpen, setReferralOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [tonActionsOpen, setTonActionsOpen] = useState(false);
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();
  const tonBalance = useTonBalance();

  const handleLogout = () => {
    logout();
    navigate('/');
    setMenuOpen(false);
  };

  const openTonWalletActions = () => {
    if (!wallet) {
      tonConnectUI.openModal();
      return;
    }
    setTonActionsOpen(true);
  };

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
      <div
        className="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col"
        style={{ '--cta-bottom': 'calc(4rem + env(safe-area-inset-bottom))' } as React.CSSProperties}
      >
        {/* ── Mini App header ── */}
        <header className="card mx-3 mt-3 px-4 py-3">
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
                      className="w-11 h-11 rounded-full object-cover shrink-0"
                    />
                  ) : (
                    <div className="w-11 h-11 rounded-full bg-indigo-500 flex items-center justify-center text-white font-semibold text-base shrink-0 select-none">
                      {user.username.charAt(0).toUpperCase()}
                    </div>
                  )}
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <span className="font-semibold text-base text-gray-900 dark:text-white truncate">
                        {user.username}
                      </span>
                      {user.role === 'admin' && (
                        <span className="text-xs bg-indigo-600/30 text-indigo-400 dark:text-indigo-300 px-1.5 py-0.5 rounded shrink-0">
                          admin
                        </span>
                      )}
                    </div>
                    <div className="text-sm text-gray-500 dark:text-gray-400 truncate">
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
              onClick={openTonWalletActions}
              aria-label="GRAM wallet"
              className={`flex items-center gap-1.5 h-8 px-3 rounded-full text-sm font-semibold transition-colors touch-manipulation shrink-0 ${
                wallet
                  ? 'bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
              }`}
            >
              <img src="/toncoin.jpg" alt="GRAM" className="w-5 h-5 rounded-full object-cover shrink-0" />
              <span className="tabular-nums whitespace-nowrap">
                {wallet
                  ? (tonBalance !== null ? `${tonBalance} GRAM` : '...')
                  : 'GRAM'}
              </span>
            </button>
          </div>
        </header>

        {/* ── Page content ── */}
        <main
          className="w-full flex-1 px-3 pt-5"
          style={{ paddingBottom: 'calc(4.5rem + env(safe-area-inset-bottom))' }}
        >
          <Outlet />
        </main>

        <TonWalletSheet open={tonActionsOpen} onClose={() => setTonActionsOpen(false)} />

        {/* ── Bottom navigation bar ── */}
        <nav
          className="fixed bottom-0 left-0 right-0 z-40 bg-white/90 dark:bg-gray-900/90 backdrop-blur border-t border-gray-100 dark:border-gray-800"
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
                      ? 'text-indigo-500 dark:text-indigo-400'
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
    <div
      className="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col"
      style={{ '--cta-safe': 'env(safe-area-inset-bottom)' } as React.CSSProperties}
    >
      <ReferralModal isOpen={referralOpen} onClose={() => setReferralOpen(false)} />
      <nav className="card sticky top-3 z-40 mx-3 mt-3 sm:mx-auto sm:w-[calc(100%-2rem)] max-w-6xl">
        <div className="px-4 py-3 flex items-center justify-between gap-2">
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
              onClick={openTonWalletActions}
              aria-label="GRAM wallet"
              className={`hidden sm:flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-xs font-medium transition-colors touch-manipulation shrink-0 ${
                wallet
                  ? 'bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-500/20 dark:hover:bg-indigo-500/20'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'
              }`}
            >
              <img src="/toncoin.jpg" alt="GRAM" className="w-5 h-5 rounded-full object-cover shrink-0" />
              <span className="tabular-nums whitespace-nowrap">
                {wallet
                  ? (tonBalance !== null ? `${tonBalance} GRAM` : '...')
                  : 'GRAM'}
              </span>
            </button>

            <ThemeToggle />
            <LangToggle />
            <button
              type="button"
              onClick={openTonWalletActions}
              aria-label="GRAM wallet"
              className={`md:hidden flex items-center justify-center w-9 h-9 rounded-full transition-colors touch-manipulation shrink-0 ${
                wallet
                  ? 'bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'
              }`}
            >
              <img src="/toncoin.jpg" alt="GRAM" className="w-5 h-5 rounded-full object-cover" />
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
              <div className="hidden md:flex items-center gap-2">
                <TelegramLoginButton
                  label={t.nav.login}
                  className="flex items-center gap-1.5 bg-[#54a9eb] hover:bg-[#4a96d2] disabled:opacity-50 text-white text-sm font-semibold rounded-full px-3 py-1.5 transition-colors touch-manipulation whitespace-nowrap"
                />
                <CredentialsAuthButton className="bg-gray-900 hover:bg-gray-800 text-white text-sm font-semibold rounded-full px-3 py-1.5 transition-colors touch-manipulation whitespace-nowrap" />
              </div>
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
          <div className="md:hidden border-t border-gray-100 dark:border-gray-800 px-4 py-3 flex flex-col gap-3">
            {navLinks}
            <div className="flex items-center justify-between pt-2 border-t border-gray-100 dark:border-gray-800">
              {user ? (
                <>
                  <span className="text-sm text-gray-500 dark:text-gray-400">{user.username}</span>
                  <button onClick={handleLogout} className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors py-2 touch-manipulation">
                    {t.nav.logout}
                  </button>
                </>
              ) : (
                <div className="w-full flex flex-col gap-2">
                  <TelegramLoginButton
                    label={t.nav.login}
                    className="flex items-center justify-center gap-1.5 bg-[#54a9eb] hover:bg-[#4a96d2] disabled:opacity-50 text-white text-sm font-semibold rounded-full px-3 py-2 transition-colors touch-manipulation"
                  />
                  <CredentialsAuthButton
                    className="w-full bg-gray-900 hover:bg-gray-800 text-white text-sm font-semibold rounded-full px-3 py-2 transition-colors touch-manipulation"
                    onSuccess={() => setMenuOpen(false)}
                  />
                </div>
              )}
            </div>
          </div>
        )}
      </nav>
      <main className="mx-auto max-w-6xl w-full px-3 sm:px-4 pt-5 pb-4 sm:pt-8 flex-1">
        <Outlet />
      </main>

      <TonWalletSheet open={tonActionsOpen} onClose={() => setTonActionsOpen(false)} />

      <footer className="py-6">
        <div className="mx-auto max-w-6xl px-4 flex flex-wrap items-center justify-between gap-x-6 gap-y-2 text-xs text-gray-400 dark:text-gray-500">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span>
              {t.footer.feedback}{' '}
              <a
                href={SUPPORT_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="text-indigo-500 dark:text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-300 transition-colors"
              >
                @oddwallet
              </a>
            </span>
            <span aria-hidden="true">|</span>
            <a href={PRIVACY_URL} target="_blank" rel="noopener noreferrer" className="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
              {t.footer.privacy}
            </a>
            <span aria-hidden="true">|</span>
            <a href={TERMS_URL} target="_blank" rel="noopener noreferrer" className="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
              {t.footer.terms}
            </a>
          </div>
          <span>© {t.common.appName} {new Date().getFullYear()}</span>
        </div>
      </footer>
    </div>
  );
}
