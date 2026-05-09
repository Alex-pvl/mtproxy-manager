import { Link } from 'react-router-dom';
import Sticker from '../components/Sticker';
import { useLanguage } from '../context/LanguageContext';

interface ProductCard {
  to: string;
  sticker: string;
  title: string;
  desc: string;
  badge?: string;
}

export default function Home() {
  const { t } = useLanguage();

  const cards: ProductCard[] = [
    {
      to: '/pricing',
      sticker: 'tariffs',
      title: t.showcase.vpn.title,
      desc: t.showcase.vpn.desc,
    },
    {
      to: '/stars',
      sticker: 'cipher',
      title: t.showcase.stars.title,
      desc: t.showcase.stars.desc,
      badge: t.showcase.fragmentBadge,
    },
    {
      to: '/premium',
      sticker: 'no_logs',
      title: t.showcase.premium.title,
      desc: t.showcase.premium.desc,
      badge: t.showcase.fragmentBadge,
    },
  ];

  return (
    <div>
      <div className="text-center mb-6 sm:mb-8">
        <h1 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white mb-2">
          {t.showcase.title}
        </h1>
        <p className="text-gray-500 dark:text-gray-400 text-sm sm:text-base px-2 max-w-xl mx-auto">
          {t.showcase.subtitle}
        </p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 max-w-4xl mx-auto">
        {cards.map((c) => (
          <Link
            key={c.to}
            to={c.to}
            className="group relative bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 hover:border-indigo-500 hover:ring-1 hover:ring-indigo-500/40 rounded-2xl p-5 flex flex-col items-center text-center transition-all touch-manipulation"
          >
            {c.badge && (
              <span className="absolute top-3 right-3 text-[10px] uppercase tracking-wider bg-indigo-50 dark:bg-indigo-500/15 text-indigo-600 dark:text-indigo-300 px-2 py-0.5 rounded-full">
                {c.badge}
              </span>
            )}
            <Sticker name={c.sticker} className="w-20 h-20 sm:w-24 sm:h-24 mb-3" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white mb-1">
              {c.title}
            </h3>
            <p className="text-xs sm:text-sm text-gray-500 dark:text-gray-400 leading-relaxed">
              {c.desc}
            </p>
            <span className="mt-3 text-xs font-medium text-indigo-600 dark:text-indigo-400 group-hover:underline">
              {t.showcase.openCard} →
            </span>
          </Link>
        ))}
      </div>
    </div>
  );
}
