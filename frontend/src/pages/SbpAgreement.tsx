import { Link } from 'react-router-dom';
import { useLanguage } from '../context/LanguageContext';

export default function SbpAgreement() {
  const { t } = useLanguage();
  const { sbpAgreement: a } = t;

  return (
    <div className="max-w-2xl mx-auto">
      <Link
        to="/pricing"
        className="inline-block text-sm text-indigo-600 dark:text-indigo-400 hover:underline mb-6"
      >
        {a.back}
      </Link>
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">{a.title}</h1>
      <p className="text-xs text-gray-500 dark:text-gray-400 mb-8">{a.lastUpdated}</p>

      <div className="space-y-8 text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
        {a.sections.map((section, i) => (
          <section key={i}>
            <h2 className="text-base font-semibold text-gray-900 dark:text-white mb-3">{section.heading}</h2>
            <div className="space-y-3">
              {section.paragraphs.map((p, j) => (
                <p key={j}>{p}</p>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}
