import { useState, useEffect } from 'react';
import { paymentApi } from '../api/client';
import type { Plan } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { Link, useNavigate } from 'react-router-dom';
import Sticker from '../components/Sticker';

const POPULAR_PLAN = 'year_1';

export default function Pricing() {
  const { user, refreshUser } = useAuth();
  const { t } = useLanguage();
  const navigate = useNavigate();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [buyingPlan, setBuyingPlan] = useState<string | null>(null);
  const [error, setError] = useState('');

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

  const handleBuy = async (planId: string) => {
    if (!user) { navigate('/login'); return; }
    setBuyingPlan(planId);
    setError('');
    try {
      const res = await paymentApi.createPayment(planId);
      window.location.href = res.data.payment_url;
    } catch (err: any) {
      setError(err.response?.data?.error || t.pricing.failedPayment);
      setBuyingPlan(null);
    }
  };

  if (loading) {
    return <div className="text-gray-500 dark:text-gray-400">{t.pricing.loading}</div>;
  }

  return (
    <div>
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

              <ul className="text-sm text-gray-600 dark:text-gray-300 space-y-2 mb-6 flex-1">
                <li className="flex items-center gap-2">
                  <span className="text-emerald-500 dark:text-emerald-400">&#10003;</span>
                  {t.pricing.proxy(plan.max_proxies)}
                </li>
              </ul>

              <button
                onClick={() => handleBuy(plan.id)}
                disabled={buyingPlan !== null}
                className={`w-full text-sm font-medium rounded px-4 py-2.5 transition-colors disabled:opacity-50 touch-manipulation ${
                  isPopular
                    ? 'bg-indigo-600 hover:bg-indigo-500 text-white'
                    : 'bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-200'
                }`}
              >
                {buyingPlan === plan.id
                  ? t.pricing.redirecting
                  : sub?.active
                    ? t.pricing.renew
                    : t.pricing.buy}
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
