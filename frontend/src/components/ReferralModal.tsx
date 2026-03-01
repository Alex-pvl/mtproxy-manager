import { useState, useEffect } from 'react';
import { referralApi } from '../api/client';
import Sticker from './Sticker';
import { useLanguage } from '../context/LanguageContext';

interface ReferralModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function ReferralModal({ isOpen, onClose }: ReferralModalProps) {
  const { t } = useLanguage();
  const [link, setLink] = useState('');
  const [invitedCount, setInvitedCount] = useState(0);
  const [bonusDays, setBonusDays] = useState(0);
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setLoading(true);
    referralApi
      .get()
      .then((res) => {
        setLink(res.data.referral_link);
        setInvitedCount(res.data.invited_count);
        setBonusDays(res.data.bonus_days_received);
      })
      .catch(() => {
        setLink('');
      })
      .finally(() => setLoading(false));
  }, [isOpen]);

  const handleCopy = async () => {
    if (!link) return;
    try {
      await navigator.clipboard.writeText(link);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {}
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} aria-hidden="true" />
      <div className="relative bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl shadow-xl max-w-md w-full p-6">
        <button
          type="button"
          onClick={onClose}
          className="absolute top-4 right-4 p-2 -m-2 text-gray-400 hover:text-gray-700 dark:hover:text-white transition-colors touch-manipulation"
          aria-label="Close"
        >
          <span className="text-xl leading-none">&times;</span>
        </button>

        <div className="flex justify-center mb-4">
          <Sticker name="referals" className="w-14 h-14" />
        </div>

        <h2 className="text-xl font-bold text-gray-900 dark:text-white text-center mb-2">
          {t.referral.title}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400 text-center mb-6">
          {t.referral.description}
        </p>

        <div className="mb-4">
          <label className="block text-xs text-gray-400 dark:text-gray-500 mb-1">{t.referral.link}</label>
          <div className="bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg px-3 py-2.5 text-sm text-gray-700 dark:text-gray-300 break-all">
            {loading ? t.referral.loading : link || '—'}
          </div>
        </div>

        <div className="flex flex-col sm:flex-row sm:gap-4 gap-1 text-sm text-gray-500 dark:text-gray-400 mb-6">
          <span>{t.referral.invited} <span className="text-gray-900 dark:text-white font-medium">{invitedCount}</span></span>
          <span>{t.referral.bonusDays} <span className="text-gray-900 dark:text-white font-medium">{bonusDays}</span></span>
        </div>

        <div className="space-y-3">
          <button
            type="button"
            onClick={handleCopy}
            disabled={loading || !link}
            className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-4 py-3 transition-colors touch-manipulation"
          >
            {copied ? t.referral.copied : t.referral.copy}
          </button>
        </div>
      </div>
    </div>
  );
}
