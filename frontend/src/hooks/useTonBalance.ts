import { useState, useEffect } from 'react';
import { useTonWallet } from '@tonconnect/ui-react';

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
        if (!cancelled && typeof data.balance === 'number') {
          setBalance((data.balance / 1e9).toFixed(2));
        }
      })
      .catch(() => {
        if (!cancelled) setBalance(null);
      });

    return () => { cancelled = true; };
  }, [wallet?.account.address]);

  return balance;
}
