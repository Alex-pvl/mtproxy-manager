import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { apiError, paymentApi, proxyApi, type Plan } from '../api/client';
import PromoBanner from '../components/PromoBanner';
import Sticker from '../components/Sticker';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';

const rubles = (label: string) => parseInt(label.replace(/\D/g, ''), 10);

export default function Home() {
  const { t, language } = useLanguage();
  const { user, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [trialBusy, setTrialBusy] = useState(false);
  const [trialError, setTrialError] = useState('');

  useEffect(() => {
    paymentApi.listPlans().then((res) => setPlans(res.data)).catch(() => {});
  }, []);

  const sub = user?.subscription;
  const minPerMonth = plans.length
    ? Math.min(...plans.map((p) => rubles(p.sbp_per_month ?? p.per_month)))
    : null;

  const startTrial = async () => {
    setTrialBusy(true);
    setTrialError('');
    try {
      await proxyApi.startTrial();
      await refreshUser();
      navigate('/proxies');
    } catch (err) {
      setTrialError(apiError(err, t.showcase.trialFailed));
    } finally {
      setTrialBusy(false);
    }
  };

  const features = [
    { sticker: 'no_logs', title: t.pricing.featureNoLogs, desc: t.pricing.featureNoLogsDesc },
    { sticker: 'cipher', title: t.pricing.featureCipher, desc: t.pricing.featureCipherDesc },
    { sticker: 'speed', title: t.pricing.featureSpeed, desc: t.pricing.featureSpeedDesc },
  ];

  return (
    <div className="max-w-xl mx-auto">
      <div className="card flex flex-col items-center text-center px-5 pt-6 pb-5 mb-6">
        <Sticker name="tariffs" className="w-28 h-28 sm:w-36 sm:h-36 mb-4" />
        <h1 className="text-2xl sm:text-3xl font-extrabold text-gray-900 dark:text-white leading-tight mb-6">
          {t.showcase.title}
        </h1>
        {sub?.trial_available ? (
          <>
            <button
              type="button"
              onClick={startTrial}
              disabled={trialBusy}
              className="spend-cta block touch-manipulation"
            >
              🎁 {trialBusy ? t.proxies.creating : t.showcase.trial}
            </button>
            {trialError && <p className="mt-2 text-xs text-red-600 dark:text-red-400">{trialError}</p>}
            <Link
              to="/pricing"
              className="mt-2.5 block w-full rounded-full bg-indigo-500/10 px-5 py-3 text-base font-semibold text-indigo-600 dark:text-indigo-400 touch-manipulation"
            >
              {t.showcase.cta}
            </Link>
            <p className="mt-2 text-xs text-gray-500 dark:text-gray-400 h-4">
              {minPerMonth != null && t.showcase.priceFrom(minPerMonth)}
            </p>
          </>
        ) : (
          <>
            <Link
              to={sub?.active ? '/proxies' : '/pricing'}
              className="spend-cta block text-center touch-manipulation"
            >
              {sub?.active ? t.showcase.ctaActive : t.showcase.cta}
            </Link>
            <p className="mt-2 text-xs text-gray-500 dark:text-gray-400 h-4">
              {sub?.active && sub.expires_at
                ? t.showcase.activeUntil(new Date(sub.expires_at).toLocaleDateString(language === 'ru' ? 'ru-RU' : 'en-US'))
                : minPerMonth != null && t.showcase.priceFrom(minPerMonth)}
            </p>
          </>
        )}
      </div>

      <PromoBanner plans={plans} onClick={() => navigate('/pricing')} />

      <div className="grid grid-cols-3 gap-2 sm:gap-3 mb-6">
        {features.map((f) => (
          <div
            key={f.sticker}
            className="card p-3 flex flex-col items-center text-center !rounded-[20px]"
          >
            <Sticker name={f.sticker} className="w-10 h-10 mb-2" />
            <h3 className="text-xs sm:text-sm font-semibold text-gray-900 dark:text-white mb-0.5">{f.title}</h3>
            <p className="text-[11px] sm:text-xs text-gray-500 dark:text-gray-400 leading-snug">{f.desc}</p>
          </div>
        ))}
      </div>

      <h2 className="section-title">{t.showcase.stepsTitle}</h2>
      <ol className="card overflow-hidden mb-6">
        {t.showcase.steps.map((step, i) => (
          <li
            key={step}
            className="choice-row cursor-default"
          >
            <span className="w-7 h-7 shrink-0 rounded-full bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 text-sm font-bold flex items-center justify-center">
              {i + 1}
            </span>
            <span className="text-[15px] text-gray-900 dark:text-gray-100">{step}</span>
          </li>
        ))}
      </ol>

      <Link
        to="/referral"
        className="group flex items-center gap-3 card p-4 transition-colors touch-manipulation"
      >
        <Sticker name="referals" className="w-12 h-12" />
        <span className="flex-1 text-sm font-medium text-gray-900 dark:text-white">{t.referral.title}</span>
        <span className="text-indigo-600 dark:text-indigo-400 group-hover:translate-x-0.5 transition-transform">→</span>
      </Link>
    </div>
  );
}
