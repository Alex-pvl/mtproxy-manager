import { useEffect, useMemo, useRef, useState } from 'react';
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

type PayMethod = 'sbp' | 'cryptobot' | 'ton';

const STARS_PRESETS = [50, 100, 250, 500, 1000, 2500, 5000];
const PREMIUM_OPTIONS = [3, 6, 12];

interface Props { type: ProductType }

// ─── Inline icons ────────────────────────────────────────────────────────────

function ChevronDown() {
  return (
    <svg className="spend-pill__chevron" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="6 9 12 15 18 9" />
    </svg>
  );
}
function AtIcon() { return <span className="font-medium">@</span>; }
function DollarIcon() { return <span className="font-medium">$</span>; }
function StarIcon({ color = 'currentColor' }: { color?: string }) {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill={color} aria-hidden>
      <path d="M12 2.5l2.95 6 6.6.96-4.78 4.66 1.13 6.58L12 17.6l-5.9 3.1L7.23 14.1 2.45 9.46l6.6-.96L12 2.5z" />
    </svg>
  );
}
function CheckCircleIcon() {
  return (
    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#3aa8fc" strokeWidth="1.5">
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < bytes.length; i += 1) binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

// ─── Pill button (a click-to-expand selector) ────────────────────────────────

interface PillButtonProps {
  icon: React.ReactNode;
  label: React.ReactNode;
  valueRight?: React.ReactNode;
  open?: boolean;
  onClick?: () => void;
}
function PillButton({ icon, label, valueRight, open, onClick }: PillButtonProps) {
  return (
    <button type="button" onClick={onClick} aria-expanded={open ? 'true' : 'false'} className="spend-pill">
      <span className="spend-pill__icon">{icon}</span>
      <span className="spend-pill__label">{label}</span>
      {valueRight !== undefined && <span className="spend-pill__value-right">{valueRight}</span>}
      <ChevronDown />
    </button>
  );
}

// ─── Page ────────────────────────────────────────────────────────────────────

export default function ProductCheckout({ type }: Props) {
  const { user } = useAuth();
  const { t } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();

  const isMiniApp = !!window.Telegram?.WebApp?.initData;
  const ownUsername = window.Telegram?.WebApp?.initDataUnsafe?.user?.username || undefined;

  const [recipient, setRecipient] = useState<string>(ownUsername || '');
  const [recipientEditing, setRecipientEditing] = useState(false);
  const [quantity, setQuantity] = useState<number>(type === 'stars' ? 100 : 3);
  const [customQty, setCustomQty] = useState('');
  const [method, setMethod] = useState<PayMethod>('sbp');

  const [methodOpen, setMethodOpen] = useState(false);
  const [amountOpen, setAmountOpen] = useState(false);

  const [quote, setQuote] = useState<ProductQuote | null>(null);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [error, setError] = useState('');
  const [processing, setProcessing] = useState(false);
  const [showSuccess, setShowSuccess] = useState<'pending' | 'delivered' | null>(null);

  const labels = type === 'stars' ? t.stars : t.premium;

  // Debounced quote
  useEffect(() => {
    if (!quantity || quantity <= 0) { setQuote(null); return; }
    const handle = setTimeout(async () => {
      setQuoteLoading(true);
      try {
        const res = type === 'stars'
          ? await productApi.starsQuote(quantity)
          : await productApi.premiumQuote(quantity);
        setQuote(res.data);
      } catch (e: any) {
        setQuote(null);
        setError(e?.response?.data?.error || labels.failedQuote);
      } finally { setQuoteLoading(false); }
    }, 300);
    return () => clearTimeout(handle);
  }, [quantity, type, labels.failedQuote]);

  // ─── Method picker ─────────────────────────────────────────────────────────
  const methods: { id: PayMethod; label: string; desc: string; disabled?: boolean }[] = useMemo(() => [
    { id: 'sbp', label: t.payment.sbp, desc: t.payment.sbpDesc },
    { id: 'cryptobot', label: t.payment.cryptobot, desc: t.payment.cryptobotDesc },
    { id: 'ton', label: t.payment.ton, desc: wallet ? t.payment.tonDesc : t.payment.tonNotConnected },
  ], [t.payment, wallet]);
  const selectedMethod = methods.find((m) => m.id === method);

  // ─── Recipient avatar (Mini App only — gives photo URL) ────────────────────
  const ownPhoto = window.Telegram?.WebApp?.initDataUnsafe?.user?.photo_url;
  const showSelfChip = !!ownUsername && !recipientEditing && recipient === ownUsername;
  const recipientInputRef = useRef<HTMLInputElement>(null);

  // ─── Buy ───────────────────────────────────────────────────────────────────
  const handleBuy = async () => {
    if (!user || processing) return;
    setError('');
    if (!recipient || recipient.length < 5) { setError(labels.invalidRecipient); return; }
    setProcessing(true);
    try {
      try {
        const check = await productApi.checkUsername(recipient);
        if (!check.data.ok) {
          setError(check.data.reason || labels.invalidRecipient);
          return;
        }
      } catch { /* worker may be down — backend will revalidate */ }

      const res = await productApi.createOrder({
        type, recipient, quantity, method,
        source: isMiniApp ? 'tg' : 'web',
      });
      const { order_id, payment_url, address, amount, comment } = res.data;

      if (method === 'ton') {
        if (!wallet) { tonConnectUI.openModal(); return; }
        const cell = beginCell().storeUint(0, 32).storeStringTail(comment!).endCell();
        await tonConnectUI.sendTransaction({
          validUntil: Math.floor(Date.now() / 1000) + 600,
          messages: [{ address: address!, amount: amount!, payload: bytesToBase64(cell.toBoc()) }],
        });
      } else if (payment_url) {
        if (isMiniApp && window.Telegram?.WebApp?.openLink) {
          window.Telegram.WebApp.openLink(payment_url, { try_browser: true });
        } else {
          window.location.href = payment_url;
          return;
        }
      }

      setShowSuccess('pending');
      for (let i = 0; i < 36; i += 1) {
        // eslint-disable-next-line no-await-in-loop
        await new Promise((r) => setTimeout(r, 5000));
        try { await paymentApi.checkPendingPayments(); } catch { /* ignore */ }
        try {
          // eslint-disable-next-line no-await-in-loop
          const s = await productApi.getOrder(order_id);
          if (s.data.status === 'delivered') { setShowSuccess('delivered'); break; }
          if (s.data.status === 'failed') { setError(s.data.error || labels.failedOrder); setShowSuccess(null); break; }
        } catch { /* ignore */ }
      }
    } catch (e: any) {
      if (e?.message !== 'Reject request') setError(e?.response?.data?.error || labels.failedOrder);
    } finally {
      setProcessing(false);
    }
  };

  // ─── Render ────────────────────────────────────────────────────────────────
  const amountLabel = type === 'stars'
    ? `${quantity.toLocaleString('ru-RU')} ⭐`
    : `${quantity} ${labels.monthsShort}`;

  return (
    <div className="spend-surface min-h-screen px-4 pt-6 pb-12 sm:pt-10 sm:pb-16 -mx-4 sm:-mx-6 lg:-mx-8">
      {/* Tab pill */}
      <div className="flex justify-center mb-8">
        <div className="spend-tabs">
          <Link to="/stars" className={`spend-tab ${type === 'stars' ? 'spend-tab--active' : ''}`}>
            Stars
          </Link>
          <Link to="/premium" className={`spend-tab ${type === 'premium' ? 'spend-tab--active' : ''}`}>
            Premium
          </Link>
        </div>
      </div>

      {/* Heading */}
      <div className="mx-auto max-w-md mb-6 px-1">
        <h1 className="text-2xl sm:text-3xl font-bold mb-2 leading-tight">{labels.title}</h1>
        <p className="text-[#8b93b3] text-sm sm:text-base leading-relaxed">{labels.subtitle}</p>
      </div>

      {/* Form */}
      <div className="mx-auto max-w-md flex flex-col gap-3">
        {/* Recipient */}
        {showSelfChip ? (
          <button
            type="button"
            onClick={() => { setRecipientEditing(true); setRecipient(''); setTimeout(() => recipientInputRef.current?.focus(), 0); }}
            className="spend-pill"
          >
            {ownPhoto ? (
              <img src={ownPhoto} alt="" className="w-7 h-7 rounded-full object-cover" />
            ) : (
              <span className="w-7 h-7 rounded-full bg-[#3aa8fc] flex items-center justify-center text-xs font-semibold">
                {ownUsername!.slice(0, 1).toUpperCase()}
              </span>
            )}
            <span className="spend-pill__label">{ownUsername}</span>
            <span className="spend-pill__value-right">{labels.recipientChange}</span>
          </button>
        ) : (
          <div className="spend-pill">
            <span className="spend-pill__icon"><AtIcon /></span>
            <input
              ref={recipientInputRef}
              type="text"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              value={recipient.replace(/^@/, '')}
              placeholder={labels.recipientPlaceholder}
              onChange={(e) => setRecipient(e.target.value.replace(/^@/, '').trim())}
              onBlur={() => { if (ownUsername && !recipient) { setRecipient(ownUsername); setRecipientEditing(false); } }}
            />
            {ownUsername && recipient !== ownUsername && (
              <button
                type="button"
                onClick={() => { setRecipient(ownUsername); setRecipientEditing(false); }}
                className="spend-pill__value-right hover:text-white"
              >
                {labels.recipientUseSelf}
              </button>
            )}
          </div>
        )}

        {/* Method */}
        <div>
          <PillButton
            icon={<DollarIcon />}
            label={selectedMethod ? selectedMethod.label : <span className="spend-pill__placeholder">{t.payment.selectMethod}</span>}
            open={methodOpen}
            onClick={() => { setMethodOpen((v) => !v); setAmountOpen(false); }}
          />
          {methodOpen && (
            <div className="spend-sheet">
              {methods.map((m) => (
                <button
                  key={m.id}
                  type="button"
                  onClick={() => { setMethod(m.id); setMethodOpen(false); }}
                  className={`spend-sheet__item ${method === m.id ? 'spend-sheet__item--active' : ''}`}
                >
                  <div className="flex-1 min-w-0">
                    <div className="font-medium">{m.label}</div>
                    <div className="spend-sheet__item-desc">{m.desc}</div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Amount / Duration */}
        <div>
          <PillButton
            icon={type === 'stars' ? <StarIcon color="#f4d03f" /> : <StarIcon color="#a48bff" />}
            label={amountLabel}
            valueRight={
              quoteLoading
                ? '…'
                : quote ? <>≈{quote.price_usd.replace(/^~/, '')}</> : undefined
            }
            open={amountOpen}
            onClick={() => { setAmountOpen((v) => !v); setMethodOpen(false); }}
          />
          {amountOpen && (
            <div className="spend-sheet">
              {(type === 'stars' ? STARS_PRESETS : PREMIUM_OPTIONS).map((q) => (
                <button
                  key={q}
                  type="button"
                  onClick={() => { setQuantity(q); setCustomQty(''); setAmountOpen(false); }}
                  className={`spend-sheet__item ${quantity === q && !customQty ? 'spend-sheet__item--active' : ''}`}
                >
                  <span className="flex-1">
                    {type === 'stars' ? `${q.toLocaleString('ru-RU')} ⭐` : `${q} ${labels.monthsShort}`}
                  </span>
                </button>
              ))}
              {type === 'stars' && (
                <div className="px-2 pt-1">
                  <input
                    type="number"
                    inputMode="numeric"
                    min={50}
                    step={50}
                    value={customQty}
                    placeholder={labels.customPlaceholder}
                    onChange={(e) => {
                      const v = e.target.value;
                      setCustomQty(v);
                      const n = Number(v);
                      if (Number.isFinite(n) && n >= 50) setQuantity(Math.floor(n));
                    }}
                    className="w-full bg-[#252a44] rounded-2xl px-4 py-3 text-sm text-white placeholder-[#8b93b3] focus:outline-none focus:ring-2 focus:ring-[#3aa8fc]"
                  />
                </div>
              )}
            </div>
          )}
        </div>

        {/* Total summary line — small, like under-input hint */}
        {quote && !amountOpen && !methodOpen && (
          <p className="text-center text-sm text-[#8b93b3] -mt-1">
            {labels.totalLabel}: <span className="text-white font-semibold">{quote.price_rub}</span>
          </p>
        )}

        {error && (
          <div className="text-sm text-red-300 bg-red-500/10 rounded-2xl px-4 py-3 border border-red-500/20">
            {error}
          </div>
        )}

        <button
          type="button"
          onClick={handleBuy}
          disabled={!user || processing || !quote || !recipient}
          className="spend-cta mt-2"
        >
          {processing ? t.payment.processing : `${labels.buyButton} ${type === 'stars' ? 'Stars' : 'Premium'}`}
        </button>

        <Link to="/" className="text-center text-sm text-[#8b93b3] hover:text-white mt-2">
          {t.showcase.back}
        </Link>
      </div>

      {/* Success modal */}
      {showSuccess && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/70" onClick={() => setShowSuccess(null)} />
          <div className="relative w-full max-w-sm rounded-3xl p-7 flex flex-col items-center text-center" style={{ background: '#1e2337' }}>
            <CheckCircleIcon />
            <h2 className="text-lg font-bold mt-3 mb-2">
              {showSuccess === 'delivered' ? labels.deliveredTitle : labels.processingTitle}
            </h2>
            <p className="text-sm text-[#8b93b3] mb-6">
              {showSuccess === 'delivered' ? labels.deliveredDesc : labels.processingDesc}
            </p>
            <button type="button" onClick={() => setShowSuccess(null)} className="spend-cta">
              {t.payment.ok}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
