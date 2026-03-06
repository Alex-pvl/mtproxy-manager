import { useState, useEffect } from 'react';
import { referralApi } from '../api/client';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';

// ─── SVG illustration ─────────────────────────────────────────────────────────

function ReferralIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-28 h-28">
      <circle cx="60" cy="60" r="56" fill="url(#grad1)" opacity="0.15" />
      <circle cx="60" cy="60" r="42" fill="url(#grad1)" opacity="0.2" />
      <circle cx="60" cy="60" r="28" fill="url(#grad1)" />
      {/* Person left */}
      <circle cx="42" cy="52" r="8" fill="white" opacity="0.9" />
      <path d="M28 72c0-7.732 6.268-14 14-14s14 6.268 14 14" stroke="white" strokeWidth="2.5" strokeLinecap="round" opacity="0.9" />
      {/* Person right */}
      <circle cx="78" cy="52" r="8" fill="white" opacity="0.9" />
      <path d="M64 72c0-7.732 6.268-14 14-14s14 6.268 14 14" stroke="white" strokeWidth="2.5" strokeLinecap="round" opacity="0.9" />
      {/* Link arrow */}
      <path d="M55 60h10" stroke="white" strokeWidth="2.5" strokeLinecap="round" />
      <path d="M62 57l3 3-3 3" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
      <defs>
        <linearGradient id="grad1" x1="0" y1="0" x2="120" y2="120" gradientUnits="userSpaceOnUse">
          <stop stopColor="#6366f1" />
          <stop offset="1" stopColor="#4f46e5" />
        </linearGradient>
      </defs>
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

function CheckIcon() {
  return (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
    </svg>
  );
}

function StarIcon() {
  return (
    <svg className="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
    </svg>
  );
}

function GiftCardIcon() {
  return (
    <svg className="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M20 12v10H4V12M22 7H2v5h20V7zM12 22V7M12 7H7.5a2.5 2.5 0 010-5C11 2 12 7 12 7zM12 7h4.5a2.5 2.5 0 000-5C13 2 12 7 12 7z" />
    </svg>
  );
}

function UserPlusIcon() {
  return (
    <svg className="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M16 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
      <circle cx="8.5" cy="7" r="4" />
      <line x1="20" y1="8" x2="20" y2="14" />
      <line x1="23" y1="11" x2="17" y2="11" />
    </svg>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function Referral() {
  const { t } = useLanguage();
  const { user } = useAuth();
  const [link, setLink] = useState('');
  const [invitedCount, setInvitedCount] = useState(0);
  const [bonusDays, setBonusDays] = useState(0);
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    setLoading(true);
    referralApi
      .get()
      .then((res) => {
        setLink(res.data.referral_link);
        setInvitedCount(res.data.invited_count);
        setBonusDays(res.data.bonus_days_received);
      })
      .catch(() => setLink(''))
      .finally(() => setLoading(false));
  }, []);

  const handleCopy = async () => {
    if (!link) return;
    try {
      await navigator.clipboard.writeText(link);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
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
    <div className="max-w-lg mx-auto pb-4">
      {/* ── Illustration + title ── */}
      <div className="flex flex-col items-center text-center pt-4 pb-6 px-4">
        <ReferralIllustration />
        <h1 className="text-xl font-bold text-gray-900 dark:text-white mt-4 mb-2">
          {t.referral.title}
        </h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 leading-relaxed">
          {t.referral.description}
        </p>
      </div>

      {/* ── Stats row ── */}
      <div className="grid grid-cols-2 gap-3 px-4 mb-4">
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-4 text-center">
          <p className="text-2xl font-bold text-indigo-600 dark:text-indigo-400">{invitedCount}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t.referral.invited.replace(':', '')}</p>
        </div>
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-4 text-center">
          <p className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{bonusDays}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t.referral.bonusDays.replace(':', '')}</p>
        </div>
      </div>

      {/* ── Referral link ── */}
      <div className="mx-4 mb-4">
        <p className="text-xs text-gray-400 dark:text-gray-500 mb-1.5 px-0.5">{t.referral.link}</p>
        <div className="flex items-center gap-2 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl px-4 py-3">
          <p className="flex-1 text-sm text-gray-600 dark:text-gray-300 truncate font-mono">
            {loading ? t.referral.loading : (link || '—')}
          </p>
          <button
            type="button"
            onClick={handleCopy}
            disabled={loading || !link}
            className={`shrink-0 p-2 rounded-xl transition-colors touch-manipulation disabled:opacity-40 ${
              copied
                ? 'text-emerald-500 bg-emerald-500/10'
                : 'text-indigo-500 bg-indigo-500/10 hover:bg-indigo-500/20'
            }`}
            aria-label={t.referral.copy}
          >
            {copied ? <CheckIcon /> : <CopyIcon />}
          </button>
        </div>
      </div>

      {/* ── Copy button ── */}
      <div className="mx-4 mb-5">
        <button
          type="button"
          onClick={handleCopy}
          disabled={loading || !link}
          className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-medium rounded-2xl px-4 py-3.5 transition-colors touch-manipulation"
        >
          {copied ? t.referral.copied : t.referral.copy}
        </button>
      </div>

      {/* ── How it works ── */}
      <div className="mx-4 space-y-3">
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-4 flex gap-3">
          <span className="text-indigo-500 dark:text-indigo-400 mt-0.5">
            <StarIcon />
          </span>
          <div>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">
              15% бонусного времени
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">
              Получайте дополнительные дни подписки каждый раз, когда ваш друг продлевает подписку.
            </p>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-4 flex gap-3">
          <span className="text-emerald-500 dark:text-emerald-400 mt-0.5">
            <GiftCardIcon />
          </span>
          <div>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">
              Бонусные подарки
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">
              Бонусные дни начисляются автоматически при каждом продлении подписки приглашённого друга.
            </p>
          </div>
        </div>

        {invitedCount > 0 && (
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-4 flex gap-3">
            <span className="text-violet-500 dark:text-violet-400 mt-0.5">
              <UserPlusIcon />
            </span>
            <div>
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">
                Приглашено пользователей
              </h3>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {invitedCount} чел. → +{bonusDays} бонусных дней
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
