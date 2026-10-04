import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { paymentApi, type Plan } from '../api/client';
import PromoBanner from '../components/PromoBanner';
import Sticker from '../components/Sticker';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';

const rubles = (label: string) => parseInt(label.replace(/\D/g, ''), 10);

export default function Home() {
  const { t, language } = useLanguage();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [plans, setPlans] = useState<Plan[]>([]);

  useEffect(() => {
    paymentApi.listPlans().then((res) => setPlans(res.data)).catch(() => {});
  }, []);

  const sub = user?.subscription;
  const minPerMonth = plans.length
    ? Math.min(...plans.map((p) => rubles(p.sbp_per_month ?? p.per_month)))
    : null;

  const features = [
    { sticker: 'no_logs', title: t.pricing.featureNoLogs, desc: t.pricing.featureNoLogsDesc },
    { sticker: 'cipher', title: t.pricing.featureCipher, desc: t.pricing.featureCipherDesc },
    { sticker: 'speed', title: t.pricing.featureSpeed, desc: t.pricing.featureSpeedDesc },
  ];

  return (
    <div className="max-w-xl mx-auto">
      <div className="flex flex-col items-center text-center pt-2 pb-8">
        <Sticker name="tariffs" className="w-32 h-32 sm:w-40 sm:h-40 mb-4" />
        <h1 className="text-2xl sm:text-3xl font-extrabold text-gray-900 dark:text-white leading-tight mb-6">
          {t.showcase.title}
        </h1>
        <Link
          to={sub?.active ? '/proxies' : '/pricing'}
          className="spend-cta block max-w-xs text-center touch-manipulation"
        >
          {sub?.active ? t.showcase.ctaActive : t.showcase.cta}
        </Link>
        <p className="mt-2 text-xs text-gray-500 dark:text-gray-400 h-4">
          {sub?.active && sub.expires_at
            ? t.showcase.activeUntil(new Date(sub.expires_at).toLocaleDateString(language === 'ru' ? 'ru-RU' : 'en-US'))
            : minPerMonth != null && t.showcase.priceFrom(minPerMonth)}
        </p>
      </div>

      <PromoBanner plans={plans} onClick={() => navigate('/pricing')} />

      <div className="grid grid-cols-3 gap-2 sm:gap-3 mb-8">
        {features.map((f) => (
          <div
            key={f.sticker}
            className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-3 flex flex-col items-center text-center"
          >
            <Sticker name={f.sticker} className="w-10 h-10 mb-2" />
            <h3 className="text-xs sm:text-sm font-semibold text-gray-900 dark:text-white mb-0.5">{f.title}</h3>
            <p className="text-[11px] sm:text-xs text-gray-500 dark:text-gray-400 leading-snug">{f.desc}</p>
          </div>
        ))}
      </div>

      <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-3">{t.showcase.stepsTitle}</h2>
      <ol className="space-y-2 mb-8">
        {t.showcase.steps.map((step, i) => (
          <li
            key={step}
            className="flex items-center gap-3 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl px-4 py-3"
          >
            <span className="w-7 h-7 shrink-0 rounded-full bg-indigo-600 text-white text-sm font-bold flex items-center justify-center">
              {i + 1}
            </span>
            <span className="text-sm text-gray-700 dark:text-gray-300">{step}</span>
          </li>
        ))}
      </ol>

      <Link
        to="/referral"
        className="group flex items-center gap-3 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 hover:border-indigo-500 rounded-2xl p-4 transition-colors touch-manipulation"
      >
        <Sticker name="referals" className="w-12 h-12" />
        <span className="flex-1 text-sm font-medium text-gray-900 dark:text-white">{t.referral.title}</span>
        <span className="text-indigo-600 dark:text-indigo-400 group-hover:translate-x-0.5 transition-transform">→</span>
      </Link>
    </div>
  );
}
