import { useState, useEffect } from 'react';
import { paymentApi } from '../api/client';
import type { Plan } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { Link } from 'react-router-dom';
import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import Sticker from '../components/Sticker';

const POPULAR_PLAN = 'year_1';

// ─── Payment method icons ─────────────────────────────────────────────────────

function CryptoBotIcon({ className = 'w-6 h-6' }: { className?: string }) {
  return <img src="/cryptobot.jpg" alt="CryptoBot" className={`${className} rounded-xl object-cover`} />;
}

function StarsPayIcon({ className = 'w-6 h-6' }: { className?: string }) {
  return <img src="/stars.jpg" alt="Telegram Stars" className={`${className} rounded-xl object-cover`} />;
}

function TonPayIcon({ className = 'w-6 h-6' }: { className?: string }) {
  return <img src="/toncoin.jpg" alt="TON" className={`${className} rounded-xl object-cover`} />;
}

function CheckCircleIcon() {
  return (
    <svg className="w-12 h-12 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function XIcon() {
  return (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}

// ─── Payment method modal ─────────────────────────────────────────────────────

type PayMethod = 'cryptobot' | 'stars' | 'ton';

interface PaymentMethodSheetProps {
  plan: Plan;
  onClose: () => void;
  onSuccess: () => void;
}


function PaymentMethodSheet({ plan, onClose, onSuccess }: PaymentMethodSheetProps) {
  const { t } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();
  const [loading, setLoading] = useState<PayMethod | null>(null);
  const [error, setError] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);

  const isMiniApp = !!(window.Telegram?.WebApp?.initData);

  const handleCryptoBot = async () => {
    setLoading('cryptobot');
    setError('');
    try {
      const res = await paymentApi.createPayment(plan.id);
      window.location.href = res.data.payment_url;
    } catch (err: any) {
      setError(err.response?.data?.error || t.pricing.failedPayment);
      setLoading(null);
    }
  };

  const handleStars = async () => {
    if (!isMiniApp) {
      setError('Оплата звёздами доступна только в Telegram');
      return;
    }
    setLoading('stars');
    setError('');
    try {
      const res = await paymentApi.createStarsPayment(plan.id);
      window.Telegram!.WebApp!.openInvoice(res.data.invoice_link, (status) => {
        setLoading(null);
        if (status === 'paid') {
          setShowSuccess(true);
          setTimeout(() => {
            onSuccess();
            onClose();
          }, 2000);
        } else if (status === 'failed' || status === 'cancelled') {
          if (status === 'failed') setError(t.pricing.failedPayment);
        }
      });
    } catch (err: any) {
      setError(err.response?.data?.error || t.pricing.failedPayment);
      setLoading(null);
    }
  };

  const handleTon = async () => {
    if (!wallet) {
      tonConnectUI.openModal();
      return;
    }
    setLoading('ton');
    setError('');
    try {
      const res = await paymentApi.createTonPayment(plan.id);
      await tonConnectUI.sendTransaction({
        validUntil: Math.floor(Date.now() / 1000) + 600,
        messages: [
          {
            address: res.data.address,
            amount: res.data.amount,
            payload: res.data.comment,
          },
        ],
      });
      setShowSuccess(true);
      setTimeout(() => {
        onSuccess();
        onClose();
      }, 2000);
    } catch (err: any) {
      if (err?.message !== 'Reject request') {
        setError(err.response?.data?.error || t.pricing.failedPayment);
      }
      setLoading(null);
    }
  };

  if (showSuccess) {
    return (
      <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center p-4">
        <div className="absolute inset-0 bg-black/60" onClick={onClose} />
        <div className="relative bg-white dark:bg-gray-900 rounded-2xl w-full max-w-sm p-8 flex flex-col items-center text-center shadow-xl">
          <CheckCircleIcon />
          <h2 className="text-lg font-bold text-gray-900 dark:text-white mt-3 mb-2">{t.payment.successTitle}</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">{t.payment.successDesc}</p>
          <button
            type="button"
            onClick={() => { onSuccess(); onClose(); }}
            className="w-full bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-xl px-4 py-3 transition-colors touch-manipulation"
          >
            {t.payment.ok}
          </button>
        </div>
      </div>
    );
  }

  const methods: { id: PayMethod; icon: React.ReactNode; label: string; desc: string; action: () => void; disabled?: boolean; badge?: string }[] = [
    {
      id: 'stars',
      icon: <StarsPayIcon className="w-10 h-10" />,
      label: t.payment.stars,
      desc: plan.stars_price ? `${plan.stars_price} ⭐` : t.payment.starsDesc,
      action: handleStars,
      disabled: !isMiniApp,
      badge: !isMiniApp ? 'Только в TG' : undefined,
    },
    {
      id: 'cryptobot',
      icon: <CryptoBotIcon className="w-10 h-10" />,
      label: t.payment.cryptobot,
      desc: plan.price_usd_label ? `${plan.price_usd_label} · ${t.payment.cryptobotDesc}` : t.payment.cryptobotDesc,
      action: handleCryptoBot,
    },
    {
      id: 'ton',
      icon: <TonPayIcon className="w-10 h-10" />,
      label: t.payment.ton,
      desc: wallet
        ? (plan.ton_amount ? `${plan.ton_amount} TON` : t.payment.tonDesc)
        : t.payment.tonNotConnected,
      action: handleTon,
    },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center p-4">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative bg-white dark:bg-gray-900 rounded-2xl w-full max-w-sm shadow-xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 pt-5 pb-3">
          <div>
            <h2 className="text-base font-bold text-gray-900 dark:text-white">{t.payment.selectMethod}</h2>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
              {plan.price_label} · {t.pricing.planNames[plan.id] ?? plan.name}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-2 -m-2 text-gray-400 hover:text-gray-700 dark:hover:text-white transition-colors touch-manipulation"
          >
            <XIcon />
          </button>
        </div>

        {error && (
          <div className="mx-5 mb-2 bg-red-500/10 border border-red-500/20 text-red-500 text-xs rounded-lg px-3 py-2">
            {error}
          </div>
        )}

        {/* Methods list */}
        <div className="px-3 pb-5 space-y-2">
          {methods.map((m) => (
            <button
              key={m.id}
              type="button"
              onClick={m.action}
              disabled={loading !== null}
              className="w-full flex items-center gap-3 px-3 py-3.5 rounded-xl bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-750 active:bg-gray-100 dark:active:bg-gray-700 transition-colors touch-manipulation disabled:opacity-60 text-left"
            >
              <span className="shrink-0">{m.icon}</span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold text-gray-900 dark:text-white">{m.label}</span>
                  {m.badge && (
                    <span className="text-xs bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 px-1.5 py-0.5 rounded">
                      {m.badge}
                    </span>
                  )}
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 truncate">{m.desc}</p>
              </div>
              {loading === m.id && (
                <svg className="w-4 h-4 animate-spin text-indigo-500 shrink-0" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
              )}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function Pricing() {
  const { user, refreshUser } = useAuth();
  const { t } = useLanguage();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedPlan, setSelectedPlan] = useState<Plan | null>(null);

  const sub = user?.subscription;

  useEffect(() => {
    paymentApi.listPlans()
      .then((res) => setPlans(res.data))
      .catch(() => setError(t.pricing.failedLoad))
      .finally(() => setLoading(false));
  }, [t.pricing.failedLoad]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.has('payment')) {
      paymentApi
        .checkPendingPayments()
        .then(() => refreshUser())
        .catch(() => refreshUser())
        .finally(() => window.history.replaceState({}, '', '/pricing'));
    }
  }, [refreshUser]);

  if (loading) {
    return <div className="text-gray-500 dark:text-gray-400">{t.pricing.loading}</div>;
  }

  return (
    <div>
      {selectedPlan && (
        <PaymentMethodSheet
          plan={selectedPlan}
          onClose={() => setSelectedPlan(null)}
          onSuccess={() => { refreshUser(); setSelectedPlan(null); }}
        />
      )}

      <div className="text-center mb-6 sm:mb-8">
        <h1 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white mb-2">{t.pricing.title}</h1>
        <p className="text-gray-500 dark:text-gray-400 text-sm sm:text-base px-2">{t.pricing.subtitle}</p>
      </div>

      <div className="mb-8">
        <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-3">{t.pricing.whyTitle}</h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-4 flex gap-3">
            <Sticker name="no_logs" className="w-10 h-10 shrink-0" />
            <div>
              <h3 className="text-indigo-500 dark:text-indigo-400 font-semibold text-sm mb-0.5">{t.pricing.featureNoLogs}</h3>
              <p className="text-gray-500 dark:text-gray-400 text-xs">{t.pricing.featureNoLogsDesc}</p>
            </div>
          </div>
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-4 flex gap-3">
            <Sticker name="cipher" className="w-10 h-10 shrink-0" />
            <div>
              <h3 className="text-indigo-500 dark:text-indigo-400 font-semibold text-sm mb-0.5">{t.pricing.featureCipher}</h3>
              <p className="text-gray-500 dark:text-gray-400 text-xs">{t.pricing.featureCipherDesc}</p>
            </div>
          </div>
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-4 flex gap-3">
            <Sticker name="speed" className="w-10 h-10 shrink-0" />
            <div>
              <h3 className="text-indigo-500 dark:text-indigo-400 font-semibold text-sm mb-0.5">{t.pricing.featureSpeed}</h3>
              <p className="text-gray-500 dark:text-gray-400 text-xs">{t.pricing.featureSpeedDesc}</p>
            </div>
          </div>
        </div>
      </div>

      {sub?.active && (
        <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-lg px-4 py-3 mb-6 text-center">
          <p className="text-emerald-600 dark:text-emerald-400 text-sm">
            {t.pricing.activeSubscription}{' '}
            <span className="font-semibold">{(sub.plan_id && t.pricing.planNames[sub.plan_id]) || sub.plan_name}</span>
            {sub.expires_at && (
              <span className="text-emerald-500 ml-2">
                {t.pricing.until} {new Date(sub.expires_at).toLocaleDateString('ru-RU')}
              </span>
            )}
          </p>
        </div>
      )}

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 dark:text-red-400 text-sm rounded px-3 py-2 mb-6 text-center">
          {error}
          <button onClick={() => setError('')} className="ml-2 text-red-400 dark:text-red-300 hover:text-red-600 dark:hover:text-white">&times;</button>
        </div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        {plans.map((plan) => {
          const isPopular = plan.id === POPULAR_PLAN;
          return (
            <div
              key={plan.id}
              className={`relative bg-white dark:bg-gray-900 border rounded-lg p-5 flex flex-col ${
                isPopular
                  ? 'border-indigo-500 ring-1 ring-indigo-500/50'
                  : 'border-gray-200 dark:border-gray-800'
              }`}
            >
              {isPopular && (
                <span className="absolute -top-2.5 left-1/2 -translate-x-1/2 bg-indigo-600 text-white text-xs font-medium px-3 py-0.5 rounded-full">
                  {t.pricing.popular}
                </span>
              )}

              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-1">{t.pricing.planNames[plan.id] ?? plan.name}</h3>

              <div className="mb-4 flex items-baseline gap-2 flex-wrap">
                <span className="text-2xl font-bold text-gray-900 dark:text-white">{plan.price_label}</span>
                {plan.price_usd_label && (
                  <span className="text-base text-gray-400 dark:text-gray-500">({plan.price_usd_label})</span>
                )}
                {plan.original_price_label && (
                  <>
                    <span className="text-base text-gray-400 dark:text-gray-500 line-through">{plan.original_price_label}</span>
                    {plan.discount_percent != null && plan.discount_percent > 0 && (
                      <span className="text-sm font-medium text-blue-500 dark:text-blue-400">−{plan.discount_percent}%</span>
                    )}
                  </>
                )}
              </div>

              <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
                {plan.per_month}{t.pricing.perMonth}
              </p>

              <ul className="text-sm text-gray-600 dark:text-gray-300 space-y-2 mb-4 flex-1">
                <li className="flex items-center gap-2">
                  <span className="text-emerald-500 dark:text-emerald-400">&#10003;</span>
                  {t.pricing.proxy(plan.max_proxies)}
                </li>
                {plan.stars_price && (
                  <li className="flex items-center gap-2">
                    <span className="text-yellow-500">⭐</span>
                    <span className="text-xs text-gray-500 dark:text-gray-400">{plan.stars_price} Stars</span>
                  </li>
                )}
              </ul>

              {/* Payment methods hint */}
              <div className="flex items-center gap-1.5 mb-3">
                <StarsPayIcon className="w-5 h-5" />
                <CryptoBotIcon className="w-5 h-5" />
                <TonPayIcon className="w-5 h-5" />
              </div>

              <button
                onClick={() => user ? setSelectedPlan(plan) : undefined}
                disabled={!user}
                className={`w-full text-sm font-medium rounded px-4 py-2.5 transition-colors disabled:opacity-50 touch-manipulation ${
                  isPopular
                    ? 'bg-indigo-600 hover:bg-indigo-500 text-white'
                    : 'bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-200'
                }`}
              >
                {sub?.active ? t.pricing.renew : t.pricing.buy}
              </button>
            </div>
          );
        })}
      </div>

      <div className="mt-8 text-center">
        <Link to="/proxies" className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors">
          {t.pricing.back}
        </Link>
      </div>
    </div>
  );
}
