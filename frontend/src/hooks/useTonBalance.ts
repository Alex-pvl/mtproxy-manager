import { useState, useEffect } from 'react';
import { useTonWallet } from '@tonconnect/ui-react';

function parseBalance(data: unknown): string | null {
  if (!data || typeof data !== 'object') return null;
  const obj = data as Record<string, unknown>;
  // TonCenter: { result: "1592521995920473" } (string, nanotons)
  // TonAPI v2: { balance: number } or { balance: string } (nanotons)
  const nano = obj.result ?? obj.balance;
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
  const address = useTonWallet()?.account.address;
  // Keyed by address so a stale balance is never shown for another (or no) wallet.
  const [fetched, setFetched] = useState<{ address: string; balance: string | null } | null>(null);

  useEffect(() => {
    if (!address) return;
    let cancelled = false;
    fetch(`https://tonapi.io/v2/accounts/${encodeURIComponent(address)}`)
      .then((r) => r.json())
      .then((data) => !cancelled && setFetched({ address, balance: parseBalance(data) }))
      .catch(() => !cancelled && setFetched({ address, balance: null }));
    return () => { cancelled = true; };
  }, [address]);

  return address && fetched?.address === address ? fetched.balance : null;
}
