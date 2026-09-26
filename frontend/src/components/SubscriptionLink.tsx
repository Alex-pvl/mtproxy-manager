import { useState } from 'react';
import { useLanguage } from '../context/LanguageContext';

const hiddenStyle = {
  color: 'transparent',
  textShadow: '0 0 18px rgba(156, 163, 175, 0.85)',
  transition: 'color 0.15s ease, text-shadow 0.15s ease',
};

// Shows a subscription URL with its secret last path segment blurred until revealed.
export function SubscriptionLink({ url }: { url: string }) {
  const [revealed, setRevealed] = useState(false);
  const { t } = useLanguage();
  const cut = url.lastIndexOf('/') + 1;

  return (
    <span>
      {url.slice(0, cut)}
      <span className={revealed ? '' : 'select-none'} style={revealed ? undefined : hiddenStyle}>
        {url.slice(cut)}
      </span>
      <button
        type="button"
        onClick={() => setRevealed((v) => !v)}
        className="ml-2 text-indigo-500 dark:text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-300 text-xs"
      >
        {revealed ? t.proxies.hide : t.proxies.show}
      </button>
    </span>
  );
}
