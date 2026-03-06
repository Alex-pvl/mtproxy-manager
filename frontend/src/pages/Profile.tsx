import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import { useAuth } from '../context/AuthContext';
import { useTheme } from '../context/ThemeContext';
import { useLanguage } from '../context/LanguageContext';
import { referralApi } from '../api/client';

// ─── Icons ────────────────────────────────────────────────────────────────────

function ChevronIcon() {
  return (
    <svg className="w-4 h-4 text-gray-400 dark:text-gray-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 18l6-6-6-6" />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
      <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
    </svg>
  );
}

function GiftIcon() {
  return (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
    </svg>
  );
}

function SupportIcon() {
  return (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
    </svg>
  );
}

function TelegramIcon() {
  return (
    <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
      <path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.48.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z" />
    </svg>
  );
}

function ShieldIcon() {
  return (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
    </svg>
  );
}

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

// ─── Row components ───────────────────────────────────────────────────────────

function ListRow({
  icon,
  label,
  sublabel,
  onClick,
  href,
  right,
}: {
  icon: React.ReactNode;
  label: string;
  sublabel?: string;
  onClick?: () => void;
  href?: string;
  right?: React.ReactNode;
}) {
  const inner = (
    <div className="flex items-center gap-3 px-4 py-3.5 w-full">
      <span className="shrink-0 text-indigo-500 dark:text-indigo-400">{icon}</span>
      <div className="flex-1 min-w-0">
        <span className="text-sm font-medium text-gray-900 dark:text-white">{label}</span>
        {sublabel && (
          <p className="text-xs text-gray-500 dark:text-gray-400 truncate mt-0.5">{sublabel}</p>
        )}
      </div>
      {right ?? <ChevronIcon />}
    </div>
  );

  const base = 'w-full text-left transition-colors active:bg-gray-100 dark:active:bg-gray-800 touch-manipulation';

  if (href) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer" className={base}>
        {inner}
      </a>
    );
  }
  return (
    <button type="button" onClick={onClick} className={base}>
      {inner}
    </button>
  );
}

function ListSection({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl overflow-hidden divide-y divide-gray-100 dark:divide-gray-800">
      {children}
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function Profile() {
  const { user, telegramPhotoUrl } = useAuth();
  const navigate = useNavigate();
  const { t } = useLanguage();
  const { theme, toggleTheme } = useTheme();
  const { language, setLanguage } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();
  const [copiedLink, setCopiedLink] = useState(false);
  const [referralLink, setReferralLink] = useState('');

  useEffect(() => {
    referralApi.get().then((res) => setReferralLink(res.data.referral_link)).catch(() => {});
  }, []);

  const handleCopyLink = async () => {
    if (!referralLink) return;
    try {
      await navigator.clipboard.writeText(referralLink);
      setCopiedLink(true);
      setTimeout(() => setCopiedLink(false), 2000);
    } catch {}
  };

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-gray-500 dark:text-gray-400 text-sm">
        {t.proxies.loginToView}
      </div>
    );
  }

  return (
    <div className="max-w-lg mx-auto space-y-4 pb-4">
      {/* ── Avatar + name ── */}
      <div className="flex flex-col items-center pt-6 pb-2">
        {telegramPhotoUrl ? (
          <img
            src={telegramPhotoUrl}
            alt={user.username}
            className="w-24 h-24 rounded-full object-cover shadow-lg mb-3"
          />
        ) : (
          <div className="w-24 h-24 rounded-full bg-gradient-to-br from-indigo-500 to-indigo-700 flex items-center justify-center text-white text-4xl font-bold select-none shadow-lg mb-3">
            {user.username.charAt(0).toUpperCase()}
          </div>
        )}
        <h1 className="text-xl font-bold text-gray-900 dark:text-white">{user.username}</h1>
        {user.subscription?.active && user.subscription.expires_at && (
          <p className="text-sm text-emerald-500 dark:text-emerald-400 mt-1">
            {t.proxies.subscriptionUntil}{' '}
            {new Date(user.subscription.expires_at).toLocaleDateString(
              language === 'ru' ? 'ru-RU' : 'en-US',
              { day: 'numeric', month: 'long', year: 'numeric' }
            )}
          </p>
        )}
        {user.role === 'admin' && (
          <span className="mt-2 text-xs bg-indigo-600/30 text-indigo-400 dark:text-indigo-300 px-2 py-0.5 rounded-full">
            admin
          </span>
        )}
      </div>

      {/* ── Referral section ── */}
      <ListSection>
        {referralLink && (
          <div className="flex items-center gap-2 px-4 py-3">
            <p className="text-xs text-gray-500 dark:text-gray-400 flex-1 truncate font-mono">{referralLink}</p>
            <button
              type="button"
              onClick={handleCopyLink}
              className={`shrink-0 p-2 rounded-lg transition-colors touch-manipulation ${
                copiedLink
                  ? 'text-emerald-500 bg-emerald-500/10'
                  : 'text-indigo-500 bg-indigo-500/10 hover:bg-indigo-500/20'
              }`}
              aria-label={t.profile.copyLink}
            >
              <CopyIcon />
            </button>
          </div>
        )}
        <ListRow
          icon={<GiftIcon />}
          label={t.profile.referralProgram}
          onClick={() => navigate('/referral')}
        />
      </ListSection>

      {/* ── Wallet section ── */}
      <ListSection>
        <ListRow
          icon={<img src="/toncoin.jpg" alt="TON" className="w-5 h-5 rounded-full object-cover" />}
          label={t.profile.wallet}
          sublabel={wallet ? wallet.account.address.slice(0, 6) + '...' + wallet.account.address.slice(-4) : undefined}
          onClick={() => tonConnectUI.openModal()}
          right={
            wallet ? (
              <button
                type="button"
                onClick={(e) => { e.stopPropagation(); tonConnectUI.disconnect(); }}
                className="text-xs text-red-500 bg-red-500/10 px-2 py-0.5 rounded-full shrink-0 touch-manipulation"
              >
                {t.profile.walletDisconnect}
              </button>
            ) : undefined
          }
        />
      </ListSection>

      {/* ── Support & channel ── */}
      <ListSection>
        <ListRow
          icon={<TelegramIcon />}
          label={t.profile.telegramChannel}
          href="https://t.me/staytg_news"
        />
        <ListRow
          icon={<SupportIcon />}
          label={t.profile.support}
          href="https://t.me/staytg_news"
        />
      </ListSection>

      {/* ── Admin (if admin) ── */}
      {user.role === 'admin' && (
        <ListSection>
          <ListRow
            icon={<ShieldIcon />}
            label={t.profile.admin}
            onClick={() => navigate('/admin')}
          />
        </ListSection>
      )}

      {/* ── App settings row: logo + toggles ── */}
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <img src="/logo.png" alt="Stay" className="w-8 h-8 rounded-xl object-cover shrink-0" />
          <span className="font-semibold text-sm text-gray-900 dark:text-white">Stay</span>
        </div>
        <div className="flex items-center gap-2">
          {/* Theme toggle */}
          <button
            type="button"
            onClick={toggleTheme}
            aria-label="Toggle theme"
            className="relative inline-flex items-center w-14 h-7 rounded-full transition-colors focus:outline-none bg-gray-200 dark:bg-gray-700 touch-manipulation"
          >
            <span className={`absolute inset-0 flex items-center transition-all duration-200 ${theme === 'dark' ? 'justify-end pr-1' : 'justify-start pl-1'}`}>
              <span className="w-5 h-5 rounded-full bg-white shadow flex items-center justify-center text-gray-600 dark:text-gray-700">
                {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
              </span>
            </span>
          </button>
          {/* Language toggle */}
          <button
            type="button"
            onClick={() => setLanguage(language === 'ru' ? 'en' : 'ru')}
            aria-label="Toggle language"
            className="relative inline-flex items-center w-14 h-7 rounded-full transition-colors focus:outline-none bg-gray-200 dark:bg-gray-700 touch-manipulation"
          >
            <span className={`absolute inset-0 flex items-center transition-all duration-200 ${language === 'en' ? 'justify-end pr-1' : 'justify-start pl-1'}`}>
              <span className="w-5 h-5 rounded-full bg-white shadow flex items-center justify-center text-[10px] font-bold text-gray-700">
                {language === 'ru' ? 'RU' : 'EN'}
              </span>
            </span>
          </button>
        </div>
      </div>
    </div>
  );
}
