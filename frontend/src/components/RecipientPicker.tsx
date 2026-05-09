import { useEffect, useState } from 'react';

interface Props {
  value: string;
  onChange: (username: string) => void;
  ownUsername?: string; // current user's TG username, if known
  selfLabel: string;
  giftLabel: string;
  placeholder: string;
  hint?: string;
}

export default function RecipientPicker({
  value,
  onChange,
  ownUsername,
  selfLabel,
  giftLabel,
  placeholder,
  hint,
}: Props) {
  const [mode, setMode] = useState<'self' | 'gift'>(ownUsername ? 'self' : 'gift');

  useEffect(() => {
    if (mode === 'self' && ownUsername) {
      onChange(ownUsername);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, ownUsername]);

  const showSelf = !!ownUsername;

  return (
    <div>
      {showSelf && (
        <div className="flex gap-2 mb-3">
          <button
            type="button"
            onClick={() => setMode('self')}
            className={`flex-1 text-sm font-medium rounded-xl px-3 py-2 transition-colors ${
              mode === 'self'
                ? 'bg-indigo-600 text-white'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700'
            }`}
          >
            {selfLabel}
          </button>
          <button
            type="button"
            onClick={() => setMode('gift')}
            className={`flex-1 text-sm font-medium rounded-xl px-3 py-2 transition-colors ${
              mode === 'gift'
                ? 'bg-indigo-600 text-white'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700'
            }`}
          >
            {giftLabel}
          </button>
        </div>
      )}

      {(mode === 'gift' || !showSelf) && (
        <div>
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500">@</span>
            <input
              type="text"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              value={value.replace(/^@/, '')}
              onChange={(e) => onChange(e.target.value.replace(/^@/, '').trim())}
              placeholder={placeholder}
              className="w-full pl-8 pr-3 py-3 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          {hint && <p className="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{hint}</p>}
        </div>
      )}

      {mode === 'self' && showSelf && (
        <div className="px-3 py-3 rounded-xl bg-gray-100 dark:bg-gray-800 text-sm text-gray-700 dark:text-gray-200">
          @{ownUsername}
        </div>
      )}
    </div>
  );
}
