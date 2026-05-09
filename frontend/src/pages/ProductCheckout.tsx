import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { beginCell } from '@ton/core';
import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import {
  paymentApi,
  productApi,
  type ProductQuote,
  type ProductType,
} from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import RecipientPicker from '../components/RecipientPicker';

type PayMethod = 'sbp' | 'cryptobot' | 'ton';

const STARS_PRESETS = [50, 100, 250, 500, 1000, 2500, 5000];
const PREMIUM_OPTIONS = [3, 6, 12];

interface Props {
  type: ProductType;
}

function CheckCircleIcon() {
  return (
    <svg className="w-12 h-12 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < bytes.length; i += 1) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

export default function ProductCheckout({ type }: Props) {
  const { user } = useAuth();
  const { t } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();

  const isMiniApp = !!window.Telegram?.WebApp?.initData;
  const ownUsername =
    window.Telegram?.WebApp?.initDataUnsafe?.user?.username || undefined;

  // ─── Form state ────────────────────────────────────────────────────────────
  const [quantity, setQuantity] = useState<number>(
    type === 'stars' ? 100 : 3,
  );
  const [customQty, setCustomQty] = useState<string>('');
  const [recipient, setRecipient] = useState<string>(ownUsername || '');
  const [method, setMethod] = useState<PayMethod>('sbp');
  const [quote, setQuote] = useState<ProductQuote | null>(null);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [error, setError] = useState('');
  const [processing, setProcessing] = useState(false);
  const [showSuccess, setShowSuccess] = useState<'pending' | 'delivered' | null>(null);

  const labels = type === 'stars' ? t.stars : t.premium;

  // ─── Quote (debounced) ─────────────────────────────────────────────────────
  useEffect(() => {
    if (!quantity || quantity <= 0) {
      setQuote(null);
      return;
    }
    const handle = setTimeout(async () => {
      setQuoteLoading(true);
      try {
        const res =
          type === 'stars'
            ? await productApi.starsQuote(quantity)
            : await productApi.premiumQuote(quantity);
        setQuote(res.data);
      } catch (e: any) {
        setQuote(null);
        setError(e?.response?.data?.error || labels.failedQuote);
      } finally {
        setQuoteLoading(false);
      }
    }, 300);
    return () => clearTimeout(handle);
  }, [quantity, type, labels.failedQuote]);

  // ─── Buy ───────────────────────────────────────────────────────────────────
  const handleBuy = async () => {
    if (!user || processing) return;
    setError('');
    if (!recipient || recipient.length < 5) {
      setError(labels.invalidRecipient);
      return;
    }
    setProcessing(true);
    try {
      // Fragment-side username check before charging the user.
      try {
        const check = await productApi.checkUsername(recipient);
        if (!check.data.ok) {
          setError(check.data.reason || labels.invalidRecipient);
          return;
        }
      } catch {
        // worker may be unreachable; let backend revalidate
      }

      const res = await productApi.createOrder({
        type,
        recipient,
        quantity,
        method,
        source: isMiniApp ? 'tg' : 'web',
      });
      const { order_id, payment_url, address, amount, comment } = res.data;

      if (method === 'ton') {
        if (!wallet) {
          tonConnectUI.openModal();
          return;
        }
        const commentCell = beginCell()
          .storeUint(0, 32)
          .storeStringTail(comment!)
          .endCell();
        const payloadB64 = bytesToBase64(commentCell.toBoc());
        await tonConnectUI.sendTransaction({
          validUntil: Math.floor(Date.now() / 1000) + 600,
          messages: [
            {
              address: address!,
              amount: amount!,
              payload: payloadB64,
            },
          ],
        });
      } else if (payment_url) {
        if (isMiniApp && window.Telegram?.WebApp?.openLink) {
          window.Telegram.WebApp.openLink(payment_url, { try_browser: true });
        } else {
          window.location.href = payment_url;
          return;
        }
      }

      // Poll order status (up to ~3 minutes).
      setShowSuccess('pending');
      for (let i = 0; i < 36; i += 1) {
        // eslint-disable-next-line no-await-in-loop
        await new Promise((r) => setTimeout(r, 5000));
        try {
          // eslint-disable-next-line no-await-in-loop
          await paymentApi.checkPendingPayments();
        } catch {
          // ignore transient errors — order status is the source of truth
        }
        try {
          // eslint-disable-next-line no-await-in-loop
          const s = await productApi.getOrder(order_id);
          if (s.data.status === 'delivered') {
            setShowSuccess('delivered');
            break;
          }
          if (s.data.status === 'failed') {
            setError(s.data.error || labels.failedOrder);
            setShowSuccess(null);
            break;
          }
        } catch {
          // ignore
        }
      }
    } catch (e: any) {
      if (e?.message !== 'Reject request') {
        setError(e?.response?.data?.error || labels.failedOrder);
      }
    } finally {
      setProcessing(false);
    }
  };

  // ─── UI ────────────────────────────────────────────────────────────────────
  const presets = type === 'stars' ? STARS_PRESETS : PREMIUM_OPTIONS;

  const methods: { id: PayMethod; label: string; desc: string; disabled?: boolean }[] = useMemo(() => [
    { id: 'sbp', label: t.payment.sbp, desc: t.payment.sbpDesc },
    { id: 'cryptobot', label: t.payment.cryptobot, desc: t.payment.cryptobotDesc },
    { id: 'ton', label: t.payment.ton, desc: wallet ? t.payment.tonDesc : t.payment.tonNotConnected },
  ], [t.payment, wallet]);

  return (
    <div className="max-w-xl mx-auto">
      {showSuccess && (
        <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center p-4">
          <div className="absolute inset-0 bg-black/60" onClick={() => setShowSuccess(null)} />
          <div className="relative bg-white dark:bg-gray-900 rounded-2xl w-full max-w-sm p-8 flex flex-col items-center text-center shadow-xl">
            <CheckCircleIcon />
            <h2 className="text-lg font-bold text-gray-900 dark:text-white mt-3 mb-2">
              {showSuccess === 'delivered' ? labels.deliveredTitle : labels.processingTitle}
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
              {showSuccess === 'delivered' ? labels.deliveredDesc : labels.processingDesc}
            </p>
            <button
              type="button"
              onClick={() => setShowSuccess(null)}
              className="w-full bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-xl px-4 py-3 transition-colors touch-manipulation"
            >
              {t.payment.ok}
            </button>
          </div>
        </div>
      )}

      <div className="text-center mb-6">
        <h1 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white mb-1">
          {labels.title}
        </h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">{labels.subtitle}</p>
      </div>

      <div className="space-y-5 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-5">
        {/* Recipient */}
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-2">
            {labels.recipientTitle}
          </p>
          <RecipientPicker
            value={recipient}
            onChange={setRecipient}
            ownUsername={ownUsername}
            selfLabel={labels.recipientSelf}
            giftLabel={labels.recipientGift}
            placeholder="username"
            hint={labels.recipientHint}
          />
        </div>

        {/* Quantity */}
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-2">
            {type === 'stars' ? labels.quantityTitle : labels.monthsTitle}
          </p>
          <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
            {presets.map((q) => (
              <button
                key={q}
                type="button"
                onClick={() => {
                  setQuantity(q);
                  setCustomQty('');
                }}
                className={`text-sm font-medium rounded-xl px-3 py-2 transition-colors ${
                  quantity === q && !customQty
                    ? 'bg-indigo-600 text-white'
                    : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700'
                }`}
              >
                {type === 'stars' ? `${q.toLocaleString('ru-RU')} ⭐` : `${q} ${labels.monthsShort}`}
              </button>
            ))}
          </div>
          {type === 'stars' && (
            <div className="mt-2">
              <input
                type="number"
                min={50}
                step={50}
                inputMode="numeric"
                value={customQty}
                onChange={(e) => {
                  const v = e.target.value;
                  setCustomQty(v);
                  const n = Number(v);
                  if (Number.isFinite(n) && n >= 50) setQuantity(Math.floor(n));
                }}
                placeholder={labels.customPlaceholder}
                className="w-full px-3 py-2.5 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
          )}
        </div>

        {/* Price */}
        <div className="flex items-baseline justify-between border-t border-gray-100 dark:border-gray-800 pt-4">
          <span className="text-sm text-gray-500 dark:text-gray-400">{labels.totalLabel}</span>
          {quoteLoading ? (
            <span className="text-sm text-gray-400">…</span>
          ) : quote ? (
            <span className="text-2xl font-extrabold text-gray-900 dark:text-white">
              {quote.price_rub}
              <span className="ml-2 text-sm font-normal text-gray-400">({quote.price_usd})</span>
            </span>
          ) : (
            <span className="text-sm text-gray-400">—</span>
          )}
        </div>

        {/* Method */}
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-2">
            {t.payment.selectMethod}
          </p>
          <div className="space-y-1.5">
            {methods.map((m) => (
              <button
                key={m.id}
                type="button"
                disabled={m.disabled}
                onClick={() => setMethod(m.id)}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-left transition-colors ${
                  m.id === method
                    ? 'bg-indigo-50 dark:bg-indigo-500/15 ring-1 ring-indigo-500/40'
                    : 'bg-gray-50 dark:bg-gray-800/50 hover:bg-gray-100 dark:hover:bg-gray-800'
                } ${m.disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-900 dark:text-white">{m.label}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{m.desc}</p>
                </div>
              </button>
            ))}
          </div>
        </div>

        {error && (
          <div className="bg-red-500/10 border border-red-500/20 text-red-500 dark:text-red-400 text-sm rounded-xl px-3 py-2">
            {error}
          </div>
        )}

        <button
          type="button"
          onClick={handleBuy}
          disabled={!user || processing || !quote || !recipient}
          className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-medium rounded-xl px-4 py-3 transition-colors touch-manipulation"
        >
          {processing ? t.payment.processing : labels.buyButton}
        </button>
      </div>

      <div className="mt-6 text-center">
        <Link to="/" className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors">
          {t.showcase.back}
        </Link>
      </div>
    </div>
  );
}
