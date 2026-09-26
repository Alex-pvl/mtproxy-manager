import { useCallback, useState } from 'react';

/**
 * Copies text to the clipboard and remembers which key was copied for `ms`,
 * so the UI can show "Copied" next to the right button.
 */
export function useCopy<K = string>(ms = 1500) {
  const [copied, setCopied] = useState<K | null>(null);
  const copy = useCallback(
    async (text: string, key: K) => {
      try {
        await navigator.clipboard.writeText(text);
      } catch {
        return; // clipboard unavailable (insecure context, denied permission)
      }
      setCopied(key);
      setTimeout(() => setCopied((cur) => (cur === key ? null : cur)), ms);
    },
    [ms],
  );
  return { copied, copy };
}
