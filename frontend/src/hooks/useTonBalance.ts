import { useState, useEffect } from 'react';
import { useTonWallet } from '@tonconnect/ui-react';

function parseBalance(data: unknown): string | null {
  if (!data || typeof data !== 'object') return null;
  const obj = data as Record<string, unknown>;
  // TonCenter: { result: "1592521995920473" } (string, nanotons)
  // TonAPI v2: { balance: number } or { balance: string } (nanotons)
  let nano = obj.result ?? obj.balance;
  if (nano === undefined) return null;
  const num = typeof nano === 'string' ? parseInt(nano, 10) : typeof nano === 'number' ? nano : NaN;
  if (!Number.isFinite(num) || num < 0) return null;
  return (num / 1e9).toFixed(2); // nanoTON → TON
}

/**
 * Fetches the TON balance for the connected wallet from TonAPI.
 * Returns a formatted string like "2.45" (in TON), or null when not available.
 */
export function useTonBalance(): string | null {
  const wallet = useTonWallet();
  const [balance, setBalance] = useState<string | null>(null);

  useEffect(() => {
    const address = wallet?.account.address;
    if (!address) {
      setBalance(null);
      return;
    }

    let cancelled = false;

    fetch(`https://tonapi.io/v2/accounts/${encodeURIComponent(address)}`)
      .then((r) => r.json())
      .then((data) => {
        if (!cancelled) {
          const b = parseBalance(data);
          setBalance(b);
        }
      })
      .catch(() => {
        if (!cancelled) setBalance(null);
      });

    return () => { cancelled = true; };
  }, [wallet?.account.address]);

  return balance;
}
