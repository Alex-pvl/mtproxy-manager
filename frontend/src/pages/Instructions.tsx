import { Link } from 'react-router-dom';
import { useLanguage } from '../context/LanguageContext';

function ExternalLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="text-indigo-500 dark:text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-300 transition-colors"
    >
      {children}
    </a>
  );
}

export default function Instructions() {
  const { t } = useLanguage();

  return (
    <div className="max-w-3xl mx-auto space-y-8 pb-8">
      {/* Заголовок */}
      <div className="text-center mb-4">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t.instructions.title}
        </h1>
      </div>

      {/* Раздел MTProto / SOCKS5 */}
      <section className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-5">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          {t.instructions.mtprotoTitle}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
          {t.instructions.mtprotoDesc}
        </p>

        <div className="space-y-4">
          <div>
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">📱 {t.common.appName} (Android / iOS)</h3>
            <ol className="list-decimal list-inside text-sm text-gray-600 dark:text-gray-400 space-y-1 pl-2">
              <li>{t.instructions.stepPhone1}</li>
              <li>{t.instructions.stepPhone2}</li>
              <li>{t.instructions.stepPhone3}</li>
              <li>{t.instructions.stepPhone4}</li>
              <li>{t.instructions.stepPhone5}</li>
            </ol>
          </div>
          <div>
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">💻 ПК (Windows / macOS / Linux)</h3>
            <ol className="list-decimal list-inside text-sm text-gray-600 dark:text-gray-400 space-y-1 pl-2">
              <li>{t.instructions.stepPc1}</li>
              <li>{t.instructions.stepPc2}</li>
              <li>{t.instructions.stepPc3}</li>
            </ol>
          </div>
        </div>
      </section>

      {/* Раздел VLESS */}
      <section className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-5">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          {t.instructions.vlessTitle}
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
          {t.instructions.vlessDesc}
        </p>

        <div className="mb-4">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t.instructions.vlessApps}
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-sm">
            <ExternalLink href={t.instructions.appShadowrocketLink}>
              {t.instructions.appShadowrocket}
            </ExternalLink>
            <ExternalLink href={t.instructions.appV2BoxLink}>
              {t.instructions.appV2Box}
            </ExternalLink>
            <ExternalLink href={t.instructions.appStreisandLink}>
              {t.instructions.appStreisand}
            </ExternalLink>
            <ExternalLink href={t.instructions.appV2RayTunLink}>
              {t.instructions.appV2RayTun}
            </ExternalLink>
            <ExternalLink href={t.instructions.appNPVTunnelLink}>
              {t.instructions.appNPVTunnel}
            </ExternalLink>
            <ExternalLink href={t.instructions.appHappLink}>
              {t.instructions.appHapp}
            </ExternalLink>
          </div>
        </div>

        <div>
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Пошаговая настройка
          </h3>
          <ol className="list-decimal list-inside text-sm text-gray-600 dark:text-gray-400 space-y-1 pl-2">
            <li>{t.instructions.vlessStep1}</li>
            <li>{t.instructions.vlessStep2}</li>
            <li>{t.instructions.vlessStep3}</li>
            <li>{t.instructions.vlessStep4}</li>
          </ol>
        </div>
      </section>

      {/* Кнопка назад */}
      <div className="text-center">
        <Link
          to="/proxies"
          className="inline-block text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-white transition-colors"
        >
          {t.instructions.back}
        </Link>
      </div>
    </div>
  );
}