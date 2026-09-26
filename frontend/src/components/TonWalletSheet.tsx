import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import { useLanguage } from '../context/LanguageContext';
import { useCopy } from '../hooks/useCopy';
import { toFriendlyAddress } from '../utils/tonAddress';

/** Bottom sheet with "copy address" / "disconnect" for the connected TON wallet. */
export function TonWalletSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();
  const { copied, copy } = useCopy<'wallet'>();

  if (!open || !wallet) return null;

  const disconnect = async () => {
    try {
      await tonConnectUI.disconnect();
    } finally {
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center p-4">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />
      <div className="relative w-full max-w-sm rounded-2xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-3 shadow-xl">
        <button
          type="button"
          onClick={() => copy(toFriendlyAddress(wallet.account.address), 'wallet')}
          className="w-full rounded-xl px-4 py-3 text-left text-sm font-medium text-gray-800 dark:text-gray-100 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
        >
          {copied ? t.proxies.copied : t.proxies.copy}
        </button>
        <button
          type="button"
          onClick={disconnect}
          className="mt-2 w-full rounded-xl px-4 py-3 text-left text-sm font-medium text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-500/10 hover:bg-red-100 dark:hover:bg-red-500/20 transition-colors"
        >
          {t.profile.walletDisconnect}
        </button>
        <button
          type="button"
          onClick={onClose}
          className="mt-2 w-full rounded-xl px-4 py-3 text-center text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
        >
          {t.payment.cancel}
        </button>
      </div>
    </div>
  );
}
