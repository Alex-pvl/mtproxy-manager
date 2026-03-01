import { Link } from 'react-router-dom';
import Sticker from '../components/Sticker';
import { useLanguage } from '../context/LanguageContext';

export default function Home() {
  const { t } = useLanguage();

  return (
    <div>
      <div className="flex flex-col items-center text-center mb-10">
        <Sticker name="tariffs" className="w-24 h-24 sm:w-32 sm:h-32 mb-6" />
        <h1 className="text-xl sm:text-2xl md:text-3xl font-bold text-gray-900 dark:text-white mb-3 px-2">
          <span style={{ color: '#08c' }}>Telegram</span>{' '}
          {t.home.title.replace(/^Telegram\s?/, '')}
        </h1>
        <p className="text-gray-500 dark:text-gray-400 text-sm sm:text-base leading-relaxed max-w-2xl px-2">
          {t.home.description}
        </p>
      </div>

      <div className="mb-8 max-w-2xl mx-auto px-1">
        <div className="bg-white dark:bg-gray-900/60 border border-gray-200 dark:border-gray-800 rounded-xl px-4 py-4 sm:px-6 sm:py-6">
          <h2 className="text-base font-semibold text-gray-900 dark:text-white mb-4">{t.home.featureTitle}</h2>
          <ul className="space-y-2.5 text-gray-500 dark:text-gray-400 text-sm">
            {t.home.features.map((feature, i) => (
              <li key={i} className="flex items-start gap-2.5">
                <span className="text-indigo-500 dark:text-indigo-400 shrink-0 mt-0.5">✓</span>
                <span>{feature}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      <div className="flex flex-wrap gap-3 justify-center">
        <Link
          to="/pricing"
          className="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg px-5 py-2.5 transition-colors touch-manipulation"
        >
          {t.home.choosePlan}
        </Link>
        <Link
          to="/proxies"
          className="bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-200 text-sm font-medium rounded-lg px-5 py-2.5 transition-colors touch-manipulation"
        >
          {t.home.myServices}
        </Link>
      </div>
    </div>
  );
}
