import { useState, useEffect } from 'react';
import { beginCell } from '@ton/core';
import { apiError, paymentApi } from '../api/client';
import type { Plan } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { Link } from 'react-router-dom';
import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import PromoBanner from '../components/PromoBanner';
import Sticker from '../components/Sticker';
import PaymentMethodPicker, { type PaymentMethod } from '../components/PaymentMethodPicker';

const POPULAR_PLAN = 'year_1';

function CheckCircleIcon() {
  return (
    <svg className="w-12 h-12 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
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

// TON Connect transactions must declare an expiry (unix seconds).
const tenMinutesFromNow = () => Math.floor(Date.now() / 1000) + 600;

export default function Pricing() {
  const { user, refreshUser } = useAuth();
  const { t } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedMethod, setSelectedMethod] = useState<PayMethod>('sbp');
  const [processingPlanId, setProcessingPlanId] = useState<string | null>(null);
  const [selectedPlanId, setSelectedPlanId] = useState(POPULAR_PLAN);
  const [showSuccess, setShowSuccess] = useState(false);

  const sub = user?.subscription;
  const isMiniApp = !!(window.Telegram?.WebApp?.initData);

  const methods: PaymentMethod[] = [
    { id: 'sbp', icon: '/sbp.jpg', label: t.payment.sbpLabel ?? 'RUB (СБП)' },
    { id: 'ton', icon: '/toncoin.jpg', label: t.payment.tonLabel ?? 'GRAM' },
    { id: 'cryptobot', icon: '/cryptobot.jpg', label: t.payment.cryptobotOther ?? 'Другая криптовалюта' },
    {
      id: 'stars',
      icon: '/stars.jpg',
      label: t.payment.stars,
      disabled: !isMiniApp,
      badge: !isMiniApp ? 'TG' : undefined,
    },
  ];

  const formatTon = (nano: string) => {
    const n = Number(nano);
    if (!Number.isFinite(n)) return nano;
    const ton = n / 1_000_000_000;
    const str = ton.toFixed(2).replace(/\.?0+$/, '');
    return `${str} GRAM`;
  };

  const getDisplayPrice = (plan: Plan): { main: string; secondary?: string } => {
    switch (selectedMethod) {
      case 'cryptobot':
        return { main: plan.price_usd_label ? plan.price_usd_label.replace(/^~/, '') : plan.price_label };
      case 'stars':
        return plan.stars_price
          ? { main: `${plan.stars_price.toLocaleString('ru-RU')} ⭐` }
          : { main: plan.price_label };
      case 'ton':
        return plan.ton_amount ? { main: formatTon(plan.ton_amount) } : { main: plan.price_label };
      case 'sbp':
      default:
        return plan.sbp_price_label
          ? { main: plan.sbp_price_label }
          : { main: plan.price_label, secondary: plan.price_usd_label };
    }
  };

  const openPaymentLink = (url: string) => {
    if (isMiniApp && window.Telegram?.WebApp?.openLink) {
      // try_browser: открыть в системном браузере, а не во встроенном WebView Telegram.
      // В Telegram политика no-referrer не требуется.
      window.Telegram.WebApp.openLink(url, { try_browser: true });
      return;
    }
    window.location.assign(url);
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

        // ponytail: no modal if the webhook credited first; refreshUser still shows the new subscription.
        await refreshUser();
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
        openPaymentLink(res.data.payment_url);
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
        validUntil: tenMinutesFromNow(),
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
    } catch (err) {
      const walletRejected = err instanceof Error && err.message === 'Reject request';
      if (!walletRejected) setError(apiError(err, t.pricing.failedPayment));
    } finally {
      setProcessingPlanId(null);
    }
  };

  if (loading) {
    return <div className="text-gray-500 dark:text-gray-400">{t.pricing.loading}</div>;
  }

  const selectedPlan = plans.find((p) => p.id === selectedPlanId) ?? plans.find((p) => p.id === POPULAR_PLAN) ?? plans[0];
  const features = [
    { sticker: 'no_logs', title: t.pricing.featureNoLogs, desc: t.pricing.featureNoLogsDesc },
    { sticker: 'cipher', title: t.pricing.featureCipher, desc: t.pricing.featureCipherDesc },
    { sticker: 'speed', title: t.pricing.featureSpeed, desc: t.pricing.featureSpeedDesc },
  ];

  return (
    <div className="max-w-xl mx-auto">
      {showSuccess && (
        <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center p-4">
          <div className="absolute inset-0 bg-black/60" onClick={() => setShowSuccess(false)} />
          <div className="card relative w-full max-w-sm p-8 flex flex-col items-center text-center shadow-xl">
            <CheckCircleIcon />
            <h2 className="text-lg font-bold text-gray-900 dark:text-white mt-3 mb-2">{t.payment.successTitle}</h2>
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">{t.payment.successDesc}</p>
            <button type="button" onClick={() => setShowSuccess(false)} className="spend-cta touch-manipulation">
              {t.payment.ok}
            </button>
          </div>
        </div>
      )}

      <div className="card flex items-center gap-4 p-5 mb-6">
        <Sticker name="tariffs" className="w-16 h-16 shrink-0" />
        <div className="min-w-0">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white leading-tight">{t.pricing.title}</h1>
          <p className="text-gray-500 dark:text-gray-400 text-[15px] mt-1">{t.pricing.subtitle}</p>
        </div>
      </div>

      {sub?.active && !user?.hide_sub_banner && (
        <div className="card px-4 py-3 mb-6 text-center">
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

      <PromoBanner plans={plans} onClick={() => setSelectedMethod('sbp')} />

      {error && (
        <div className="bg-red-500/10 text-red-500 dark:text-red-400 text-sm rounded-2xl px-4 py-3 mb-6 text-center">
          {error}
          <button onClick={() => setError('')} className="ml-2 text-red-400 dark:text-red-300 hover:text-red-600 dark:hover:text-white">&times;</button>
        </div>
      )}

      <p className="section-title">{t.pricing.planTitle}</p>
      <div role="radiogroup" aria-label={t.pricing.planTitle} className="card overflow-hidden mb-6">
        {plans.map((plan) => {
          const { main, secondary } = getDisplayPrice(plan);
          const discount = selectedMethod === 'sbp' && plan.discount_percent != null && plan.discount_percent > 0;
          const on = plan.id === selectedPlan?.id;
          return (
            <button
              key={plan.id}
              type="button"
              role="radio"
              aria-checked={on}
              onClick={() => setSelectedPlanId(plan.id)}
              className="choice-row touch-manipulation"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-[17px] font-semibold text-gray-900 dark:text-white">{t.pricing.planNames[plan.id] ?? plan.name}</span>
                  {discount && <span className="badge">&minus;{plan.discount_percent}%</span>}
                  {!discount && plan.id === POPULAR_PLAN && (
                    <span className="badge !bg-indigo-500/10 !text-indigo-600 dark:!text-indigo-400">{t.pricing.popular}</span>
                  )}
                </div>
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
                  {selectedMethod === 'sbp' && <>{plan.sbp_per_month ?? plan.per_month}{t.pricing.perMonth} · </>}
                  {t.pricing.proxy(plan.max_proxies)}
                </p>
              </div>
              <div className="text-right shrink-0">
                <div className="text-[17px] font-bold text-gray-900 dark:text-white whitespace-nowrap">{main}</div>
                {discount && plan.original_price_label ? (
                  <div className="text-sm text-gray-400 dark:text-gray-500 line-through whitespace-nowrap">{plan.original_price_label}</div>
                ) : (
                  secondary && <div className="text-sm text-gray-400 dark:text-gray-500 whitespace-nowrap">{secondary}</div>
                )}
              </div>
              <span className={`radio ${on ? 'radio--on' : ''}`} aria-hidden="true" />
            </button>
          );
        })}
      </div>

      <p className="section-title">{t.payment.selectMethod}</p>
      <div className="mb-6">
        <PaymentMethodPicker
          methods={methods}
          value={selectedMethod}
          onChange={(v) => setSelectedMethod(v as PayMethod)}
          placeholder={t.payment.selectMethod}
        />
        {selectedMethod === 'sbp' && (
          <p className="mt-3 text-center">
            <Link to="/legal/sbp" className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline">
              {t.payment.sbpAgreementShort}
            </Link>
          </p>
        )}
      </div>

      <p className="section-title">{t.pricing.whyTitle}</p>
      <div className="card overflow-hidden">
        {features.map((f) => (
          <div key={f.sticker} className="choice-row cursor-default">
            <Sticker name={f.sticker} className="w-10 h-10 shrink-0" />
            <div>
              <h3 className="font-semibold text-[15px] text-gray-900 dark:text-white">{f.title}</h3>
              <p className="text-gray-500 dark:text-gray-400 text-sm">{f.desc}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-6 mb-2 text-center">
        <Link to="/proxies" className="text-sm text-indigo-600 dark:text-indigo-400 hover:underline">
          {t.pricing.back}
        </Link>
      </div>

      {selectedPlan && (
        <div className="main-button">
          <button
            onClick={() => user ? handleBuyPlan(selectedPlan) : undefined}
            disabled={!user || processingPlanId !== null}
            className="spend-cta touch-manipulation"
          >
            {processingPlanId
              ? t.payment.processing
              : `${sub?.active ? t.pricing.renew : t.pricing.buy} — ${getDisplayPrice(selectedPlan).main}`}
          </button>
        </div>
      )}
    </div>
  );
}
