import { useState, useEffect } from 'react';
import { beginCell } from '@ton/core';
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

function SbpPayIcon({ className = 'w-6 h-6' }: { className?: string }) {
  return <img src="/sbp.jpg" alt="SBP" className={`${className} rounded-xl object-cover`} />;
}

function PaymentIconFrame({
  children,
  outlined = false,
}: {
  children: React.ReactNode;
  outlined?: boolean;
}) {
  return (
    <span
      className={`shrink-0 rounded-xl ${outlined ? 'ring-1 ring-gray-200 dark:ring-gray-700' : ''}`}
    >
      {children}
    </span>
  );
}

function CheckCircleIcon() {
  return (
    <svg className="w-12 h-12 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function ChevronDownIcon({ open }: { open: boolean }) {
  return (
    <svg
      className={`w-5 h-5 text-gray-400 dark:text-gray-500 transition-transform ${open ? 'rotate-180' : ''}`}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
      strokeWidth={2}
    >
      <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
    </svg>
  );
}

type PayMethod = 'cryptobot' | 'stars' | 'ton' | 'sbp';

function bytesToBase64(bytes: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < bytes.length; i += 1) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function Pricing() {
  const { user, refreshUser } = useAuth();
  const { t } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedMethod, setSelectedMethod] = useState<PayMethod>('sbp');
  const [methodMenuOpen, setMethodMenuOpen] = useState(false);
  const [processingPlanId, setProcessingPlanId] = useState<string | null>(null);
  const [showSuccess, setShowSuccess] = useState(false);

  const sub = user?.subscription;
  const isMiniApp = !!(window.Telegram?.WebApp?.initData);

  const methods: { id: PayMethod; icon: React.ReactNode; label: string; desc: string; disabled?: boolean; badge?: string }[] = [
    {
      id: 'sbp',
      icon: (
        <PaymentIconFrame outlined>
          <SbpPayIcon className="w-10 h-10" />
        </PaymentIconFrame>
      ),
      label: t.payment.sbp,
      desc: t.payment.sbpDesc,
    },
    {
      id: 'stars',
      icon: (
        <PaymentIconFrame outlined>
          <StarsPayIcon className="w-10 h-10" />
        </PaymentIconFrame>
      ),
      label: t.payment.stars,
      desc: t.payment.starsDesc,
      disabled: !isMiniApp,
      badge: !isMiniApp ? 'Только в TG' : undefined,
    },
    {
      id: 'cryptobot',
      icon: <CryptoBotIcon className="w-10 h-10" />,
      label: t.payment.cryptobot,
      desc: t.payment.cryptobotDesc,
    },
    {
      id: 'ton',
      icon: <TonPayIcon className="w-10 h-10" />,
      label: t.payment.ton,
      desc: wallet ? t.payment.tonDesc : t.payment.tonNotConnected,
    },
  ];
  const selectedMethodMeta = methods.find((m) => m.id === selectedMethod) ?? methods[0];

  const openPaymentLink = (url: string, options?: { stripReferrer?: boolean }) => {
    if (isMiniApp && window.Telegram?.WebApp?.openLink) {
      // try_browser: открыть в системном браузере, а не во встроенном WebView Telegram.
      // В Telegram политика no-referrer не требуется.
      window.Telegram.WebApp.openLink(url, { try_browser: true });
      return;
    }
    if (options?.stripReferrer) {
      // СБП-шлюз требует политику no-referrer для стабильной работы прямой ссылки НСПК.
      const a = document.createElement('a');
      a.href = url;
      a.rel = 'noreferrer noopener';
      a.referrerPolicy = 'no-referrer';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      return;
    }
    window.location.href = url;
  };

  useEffect(() => {
    paymentApi.listPlans()
      .then((res) => setPlans(res.data))
      .catch(() => setError(t.pricing.failedLoad))
      .finally(() => setLoading(false));
  }, [t.pricing.failedLoad]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const hasLegacyPaymentFlag = params.has('payment');
    const startParam =
      params.get('startapp') ||
      params.get('tgWebAppStartParam') ||
      params.get('start_param');
    const isMiniAppPaymentReturn = startParam === 'payment_success';

    if (!hasLegacyPaymentFlag && !isMiniAppPaymentReturn) {
      return;
    }

    paymentApi
      .checkPendingPayments()
      .then(async (res) => {
        if (res.data?.updated) {
          setShowSuccess(true);
        }

        await refreshUser();

        // Fallback for race conditions: webhook may activate subscription
        // a little later than checkPendingPayments returns.
        if (!res.data?.updated) {
          try {
            const subRes = await paymentApi.getSubscription();
            if (subRes.data?.active) {
              setShowSuccess(true);
            }
          } catch {
            // ignore subscription polling errors
          }
        }
      })
      .catch(() => refreshUser())
      .finally(() => {
        params.delete('payment');
        params.delete('startapp');
        params.delete('tgWebAppStartParam');
        params.delete('start_param');
        const nextSearch = params.toString();
        window.history.replaceState({}, '', nextSearch ? `/pricing?${nextSearch}` : '/pricing');
      });
  }, [refreshUser]);

  const handleBuyPlan = async (plan: Plan) => {
    if (!user || processingPlanId) return;
    setError('');
    setProcessingPlanId(plan.id);
    const paymentSource: 'web' | 'tg' = isMiniApp ? 'tg' : 'web';

    try {
      if (selectedMethod === 'sbp') {
        const res = await paymentApi.createSbpPayment(plan.id, paymentSource);
        openPaymentLink(res.data.payment_url, { stripReferrer: true });
        return;
      }

      if (selectedMethod === 'cryptobot') {
        const res = await paymentApi.createPayment(plan.id, paymentSource);
        openPaymentLink(res.data.payment_url);
        return;
      }

      if (selectedMethod === 'stars') {
        if (!isMiniApp) {
          setError('Оплата звёздами доступна только в Telegram');
          return;
        }
        const res = await paymentApi.createStarsPayment(plan.id);
        window.Telegram!.WebApp!.openInvoice(res.data.invoice_link, (status) => {
          setProcessingPlanId(null);
          if (status === 'paid') {
            setShowSuccess(true);
            refreshUser();
          } else if (status === 'failed') {
            setError(t.pricing.failedPayment);
          }
        });
        return;
      }

      // TON
      if (!wallet) {
        tonConnectUI.openModal();
        return;
      }
      const res = await paymentApi.createTonPayment(plan.id);
      const commentCell = beginCell().storeUint(0, 32).storeStringTail(res.data.comment).endCell();
      const payloadB64 = bytesToBase64(commentCell.toBoc());

      await tonConnectUI.sendTransaction({
        validUntil: Math.floor(Date.now() / 1000) + 600,
        messages: [
          {
            address: res.data.address,
            amount: res.data.amount,
            payload: payloadB64,
          },
        ],
      });
      setShowSuccess(true);
      setProcessingPlanId(null);

      for (let i = 0; i < 12; i++) {
        await new Promise((r) => setTimeout(r, 5000));
        try {
          await paymentApi.checkPendingPayments();
          await refreshUser();
          const subRes = await paymentApi.getSubscription();
          if (subRes.data?.active) break;
        } catch {
          // ignore transient polling errors
        }
      }
      return;
    } catch (err: any) {
      if (err?.message !== 'Reject request') {
        setError(err.response?.data?.error || t.pricing.failedPayment);
      }
    } finally {
      setProcessingPlanId(null);
    }
  };

  if (loading) {
    return <div className="text-gray-500 dark:text-gray-400">{t.pricing.loading}</div>;
  }

  return (
    <div>
      {showSuccess && (
        <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center p-4">
          <div className="absolute inset-0 bg-black/60" onClick={() => setShowSuccess(false)} />
          <div className="relative bg-white dark:bg-gray-900 rounded-2xl w-full max-w-sm p-8 flex flex-col items-center text-center shadow-xl">
            <CheckCircleIcon />
            <h2 className="text-lg font-bold text-gray-900 dark:text-white mt-3 mb-2">{t.payment.successTitle}</h2>
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">{t.payment.successDesc}</p>
            <button
              type="button"
              onClick={() => setShowSuccess(false)}
              className="w-full bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-xl px-4 py-3 transition-colors touch-manipulation"
            >
              {t.payment.ok}
            </button>
          </div>
        </div>
      )}

      <div className="text-center mb-6 sm:mb-8">
        <h1 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white mb-2">{t.pricing.title}</h1>
        <p className="text-gray-500 dark:text-gray-400 text-sm sm:text-base px-2">{t.pricing.subtitle}</p>
      </div>

      <div className="max-w-3xl mx-auto mb-6">
        <div className="rounded-2xl border border-emerald-200 dark:border-emerald-900/60 bg-gradient-to-r from-emerald-50 to-indigo-50 dark:from-emerald-950/40 dark:to-indigo-950/30 px-4 py-3 text-center">
          <p className="text-sm sm:text-base font-semibold text-emerald-700 dark:text-emerald-300">
            Оплата через СБП снова работает
          </p>
          <p className="text-xs sm:text-sm text-emerald-700/90 dark:text-emerald-300/90 mt-1">
            Технические работы завершены — действует скидка 20% на все тарифы
          </p>
        </div>
      </div>

      <div className="max-w-xl mx-auto mb-6">
        <p className="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-2">
          {t.payment.selectMethod}
        </p>
        <div className="relative">
          <button
            type="button"
            onClick={() => setMethodMenuOpen((v) => !v)}
            className="w-full flex items-center gap-3 px-4 py-3.5 rounded-2xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-left"
          >
            <span className="shrink-0">{selectedMethodMeta.icon}</span>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">{selectedMethodMeta.label}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{selectedMethodMeta.desc}</p>
            </div>
            <ChevronDownIcon open={methodMenuOpen} />
          </button>

          {methodMenuOpen && (
            <div className="absolute z-30 mt-2 w-full rounded-2xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-xl p-2 space-y-1">
              {methods.map((m) => (
                <button
                  key={m.id}
                  type="button"
                  disabled={m.disabled}
                  onClick={() => {
                    if (!m.disabled) {
                      setSelectedMethod(m.id);
                      setMethodMenuOpen(false);
                    }
                  }}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-left transition-colors ${
                    m.id === selectedMethod
                      ? 'bg-indigo-50 dark:bg-indigo-500/15'
                      : 'hover:bg-gray-100 dark:hover:bg-gray-800'
                  } ${m.disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
                >
                  <span className="shrink-0">{m.icon}</span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-gray-900 dark:text-white">{m.label}</span>
                      {m.badge && (
                        <span className="text-[10px] bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-300 px-1.5 py-0.5 rounded">
                          {m.badge}
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{m.desc}</p>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
        {selectedMethod === 'sbp' && !selectedMethodMeta.disabled && (
          <p className="mt-3 text-center">
            <Link
              to="/legal/sbp"
              className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
            >
              {t.payment.sbpAgreementShort}
            </Link>
          </p>
        )}
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

              <div className="mb-4">
                {plan.discount_percent != null && plan.discount_percent > 0 && (
                  <div className="mb-1.5 flex items-center gap-2">
                    <span className="inline-flex items-center rounded-full bg-rose-100 dark:bg-rose-500/20 text-rose-600 dark:text-rose-300 text-xs font-semibold px-2.5 py-1">
                      Скидка {plan.discount_percent}%
                    </span>
                    {plan.original_price_label && (
                      <span className="text-sm text-gray-400 dark:text-gray-500 line-through">{plan.original_price_label}</span>
                    )}
                  </div>
                )}

                <div className="flex items-baseline gap-2 flex-wrap">
                  <span className="text-3xl font-extrabold text-gray-900 dark:text-white">{plan.price_label}</span>
                  {plan.price_usd_label && (
                    <span className="text-base text-gray-400 dark:text-gray-500">({plan.price_usd_label})</span>
                  )}
                </div>

              </div>

              <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
                {plan.per_month}{t.pricing.perMonth}
              </p>

              <ul className="text-sm text-gray-600 dark:text-gray-300 space-y-2 mb-4 flex-1">
                <li className="flex items-center gap-2">
                  <span className="text-emerald-500 dark:text-emerald-400">&#10003;</span>
                  {t.pricing.proxy(plan.max_proxies)}
                </li>
              </ul>

              {/* Payment methods hint */}
              <div className="flex items-center gap-1.5 mb-3">
                <PaymentIconFrame outlined>
                  <SbpPayIcon className="w-5 h-5" />
                </PaymentIconFrame>
                <PaymentIconFrame outlined>
                  <StarsPayIcon className="w-5 h-5" />
                </PaymentIconFrame>
                <PaymentIconFrame>
                  <CryptoBotIcon className="w-5 h-5" />
                </PaymentIconFrame>
                <PaymentIconFrame>
                  <TonPayIcon className="w-5 h-5" />
                </PaymentIconFrame>
              </div>

              <button
                onClick={() => user ? handleBuyPlan(plan) : undefined}
                disabled={!user || processingPlanId === plan.id}
                className={`w-full text-sm font-medium rounded px-4 py-2.5 transition-colors disabled:opacity-50 touch-manipulation ${
                  isPopular
                    ? 'bg-indigo-600 hover:bg-indigo-500 text-white'
                    : 'bg-indigo-50 border border-indigo-200 dark:border-gray-700 dark:bg-gray-800 hover:bg-indigo-100 dark:hover:bg-gray-700 text-indigo-700 dark:text-gray-100'
                }`}
              >
                {processingPlanId === plan.id ? t.payment.processing : (sub?.active ? t.pricing.renew : t.pricing.buy)}
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
