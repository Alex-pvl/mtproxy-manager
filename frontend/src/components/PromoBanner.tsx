import type { Plan } from '../api/client';
import { useLanguage } from '../context/LanguageContext';

/** SBP promo banner; renders nothing when no plan carries a promo. */
export default function PromoBanner({ plans, onClick }: { plans: Plan[]; onClick: () => void }) {
  const { t, language } = useLanguage();
  const promo = plans.find((p) => p.sbp_promo_until);
  if (!promo) return null;
  const until = new Date(promo.sbp_promo_until!).toLocaleDateString(language === 'ru' ? 'ru-RU' : 'en-US', {
    day: 'numeric',
    month: 'long',
    timeZone: 'Europe/Moscow',
  });
  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full max-w-xl mx-auto block bg-indigo-500/10 border border-indigo-500/20 text-indigo-600 dark:text-indigo-300 text-sm font-medium rounded-lg px-4 py-3 mb-6 text-center touch-manipulation"
    >
      {t.pricing.promoBanner(promo.discount_percent ?? 0, until)}
    </button>
  );
}
