import { useState, useEffect } from 'react';
import { referralApi } from '../api/client';
import { useCopy } from '../hooks/useCopy';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import Sticker from '../components/Sticker';

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

function ShareIcon() {
  return (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.75}>
      <circle cx="18" cy="5" r="3" />
      <circle cx="6" cy="12" r="3" />
      <circle cx="18" cy="19" r="3" />
      <path d="M8.59 13.51L15.42 17.49" />
      <path d="M15.41 6.51L8.59 10.49" />
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
  const { user, isMiniApp } = useAuth();
  const [link, setLink] = useState('');
  const [invitedCount, setInvitedCount] = useState(0);
  const [bonusDays, setBonusDays] = useState(0);
  const [loading, setLoading] = useState(true);
  const { copied, copy } = useCopy<'link'>(2000);
  const [shared, setShared] = useState(false);

  useEffect(() => {
    referralApi
      .get(isMiniApp)
      .then((res) => {
        setLink(res.data.referral_link);
        setInvitedCount(res.data.invited_count);
        setBonusDays(res.data.bonus_days_received);
      })
      .catch(() => setLink(''))
      .finally(() => setLoading(false));
  }, [isMiniApp]);

  const handleCopy = () => link && copy(link, 'link');

  const handleShare = () => {
    if (!link) return;
    try {
      const shareUrl = `https://t.me/share/url?url=${encodeURIComponent(link)}&text=${encodeURIComponent(t.referral.shareText)}`;

      if (isMiniApp && window.Telegram?.WebApp?.openTelegramLink) {
        window.Telegram.WebApp.openTelegramLink(shareUrl);
      } else {
        const popup = window.open(shareUrl, '_blank', 'noopener,noreferrer');
        if (!popup) window.location.href = shareUrl;
      }

      setShared(true);
      setTimeout(() => setShared(false), 2000);
    } catch {
      // popup blocked and navigation refused — nothing else to try
    }
  };

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-gray-500 dark:text-gray-400 text-sm">
        {t.proxies.loginToView}
      </div>
    );
  }

  return (
    <div className="max-w-xl mx-auto pb-4">
      {/* ── Illustration + title ── */}
      <div className="card flex flex-col items-center text-center p-5 mb-4">
        <Sticker name="referals" className="w-28 h-28" />
        <h1 className="text-xl font-bold text-gray-900 dark:text-white mt-4 mb-2">
          {t.referral.title}
        </h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 leading-relaxed">
          {t.referral.description}
        </p>
      </div>

      {/* ── Stats row ── */}
      <div className="grid grid-cols-2 gap-3 mb-6">
        <div className="card p-4 text-center">
          <p className="text-2xl font-bold text-indigo-500 dark:text-indigo-400">{invitedCount}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t.referral.invitedLabel}</p>
        </div>
        <div className="card p-4 text-center">
          <p className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{bonusDays}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t.referral.bonusDaysLabel}</p>
        </div>
      </div>

      {/* ── Referral link ── */}
      <div className="mb-4">
        <p className="section-title">{t.referral.link}</p>
        <div className="flex items-center gap-2 card pl-4 pr-2 py-2">
          <p className="flex-1 text-sm text-gray-600 dark:text-gray-300 truncate font-mono">
            {loading ? t.referral.loading : (link || '—')}
          </p>
          <button
            type="button"
            onClick={handleCopy}
            disabled={loading || !link}
            className={`shrink-0 p-2.5 rounded-full transition-colors touch-manipulation disabled:opacity-40 ${
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
      <div className="mb-5 space-y-2.5">
        <button
          type="button"
          onClick={handleCopy}
          disabled={loading || !link}
          className="spend-cta !py-3.5 touch-manipulation"
        >
          {copied ? t.referral.copied : t.referral.copy}
        </button>
        <button
          type="button"
          onClick={handleShare}
          disabled={loading || !link}
          className="w-full flex items-center justify-center gap-2 bg-indigo-500/10 disabled:opacity-50 text-indigo-600 dark:text-indigo-400 text-base font-semibold rounded-full px-4 py-3.5 transition-colors touch-manipulation"
        >
          <ShareIcon />
          {shared ? t.referral.shared : t.referral.share}
        </button>
      </div>

      {/* ── How it works ── */}
      <div className="space-y-3">
        {invitedCount > 0 && (
          <div className="card p-4 flex gap-3">
            <span className="text-indigo-500 dark:text-indigo-400 mt-0.5">
              <UserPlusIcon />
            </span>
            <div>
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">
                {t.referral.invitedUsersTitle}
              </h3>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t.referral.invitedUsersSummary(invitedCount, bonusDays)}
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
