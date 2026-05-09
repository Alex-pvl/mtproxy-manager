import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { beginCell } from '@ton/core';
import { useTonConnectUI, useTonWallet } from '@tonconnect/ui-react';
import {
  paymentApi,
  productApi,
  type ProductQuote,
  type ProductType,
  type UsernameCheck,
} from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import PaymentMethodPicker, { type PaymentMethod } from '../components/PaymentMethodPicker';

type PayMethod = 'sbp' | 'cryptobot' | 'ton';
const PREMIUM_OPTIONS = [3, 6, 12];

interface Props { type: ProductType }

// ─── Inline icons ────────────────────────────────────────────────────────────

function ChevronDown({ open }: { open: boolean }) {
  return (
    <svg className="spend-pill__chevron"
      style={{ transform: open ? 'rotate(180deg)' : undefined }}
      width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
    >
      <polyline points="6 9 12 15 18 9" />
    </svg>
  );
}
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

// Deterministic accent color from username for the fallback avatar.
function accentFromString(s: string): string {
  let h = 0;
  for (let i = 0; i < s.length; i += 1) h = (h * 31 + s.charCodeAt(i)) & 0xffffffff;
  const palette = ['#3aa8fc', '#a48bff', '#34d399', '#f59e0b', '#fb7185', '#22d3ee', '#facc15'];
  return palette[Math.abs(h) % palette.length];
}

function Avatar({ photo, fallback, size = 28 }: { photo?: string; fallback: string; size?: number }) {
  if (photo) {
    return <img src={photo} alt="" className="rounded-full object-cover flex-shrink-0" style={{ width: size, height: size }} />;
  }
  const ch = fallback.replace(/^@/, '').slice(0, 1).toUpperCase() || '?';
  return (
    <span
      className="rounded-full flex items-center justify-center text-xs font-semibold flex-shrink-0"
      style={{ width: size, height: size, background: accentFromString(fallback), color: '#fff' }}
    >
      {ch}
    </span>
  );
}

// Format the price for display in the currency of the selected payment method.
// Rubles always shown in parens (except SBP, which already pays in RUB).
function formatMethodPrice(q: ProductQuote, method: PayMethod): string {
  const rub = q.price_rub;            // already formatted "X ₽"
  const usd = q.price_usd.replace(/^~/, ''); // "$X.XX"
  if (method === 'sbp') return rub;
  if (method === 'ton') {
    const ton = (Number(q.ton_amount) / 1_000_000_000).toFixed(2);
    return `${ton} TON (${rub})`;
  }
  // cryptobot — billed in USD-equivalent crypto
  return `${usd} (${rub})`;
}

// Compact price for the CTA: "~65 руб" / "~1.23 TON" / "~$1.20".
function formatCtaPrice(q: ProductQuote, method: PayMethod): string {
  if (method === 'ton') {
    const ton = (Number(q.ton_amount) / 1_000_000_000).toFixed(2);
    return `~${ton} TON`;
  }
  if (method === 'cryptobot') {
    return `~${q.price_usd.replace(/^~/, '')}`;
  }
  // sbp — RUB
  const num = q.price_rub.replace(/[^\d]/g, '');
  return `~${num} руб`;
}

// Map raw Fragment SDK / worker error codes to user-friendly text.
function friendlyError(raw: string, labels: { alreadyPremium: string; failedQuote: string; failedOrder: string }): string {
  const s = (raw || '').toLowerCase();
  if (s.includes('already_premium') || s.includes('already premium')) return labels.alreadyPremium;
  if (s.includes('quote') && s.includes('fail')) return labels.failedQuote;
  return raw || labels.failedOrder;
}

// ─── Page ────────────────────────────────────────────────────────────────────

export default function ProductCheckout({ type }: Props) {
  const { user } = useAuth();
  const { t } = useLanguage();
  const [tonConnectUI] = useTonConnectUI();
  const wallet = useTonWallet();

  const isMiniApp = !!window.Telegram?.WebApp?.initData;
  const tgUser = window.Telegram?.WebApp?.initDataUnsafe?.user;
  const ownUsername = tgUser?.username || undefined;
  const ownDisplayName = [tgUser?.first_name, tgUser?.last_name].filter(Boolean).join(' ') || ownUsername;
  const ownPhoto = tgUser?.photo_url || undefined;

  // ─── Form state ────────────────────────────────────────────────────────────
  const [recipient, setRecipient] = useState<string>(ownUsername || '');
  const [recipientEditing, setRecipientEditing] = useState<boolean>(!ownUsername);
  const [recipientInfo, setRecipientInfo] = useState<UsernameCheck | null>(
    ownUsername ? { ok: true, username: ownUsername, display_name: ownDisplayName, photo_url: ownPhoto } : null,
  );
  const recipientInputRef = useRef<HTMLInputElement>(null);

  const [quantity, setQuantity] = useState<number>(type === 'stars' ? 50 : 3);
  // Raw text for the Stars input — lets the user erase the value entirely
  // before typing a new one. `quantity` stays in sync only when the text
  // parses to a valid number; sub-50 / empty values block /quote and surface
  // an inline error on submit.
  const [starsText, setStarsText] = useState<string>(type === 'stars' ? '50' : '');
  const [premiumOpen, setPremiumOpen] = useState(false);

  const [method, setMethod] = useState<PayMethod>('sbp');

  const [quote, setQuote] = useState<ProductQuote | null>(null);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [error, setError] = useState('');
  const [processing, setProcessing] = useState(false);
  const [showSuccess, setShowSuccess] = useState<'pending' | 'delivered' | null>(null);

  const labels = type === 'stars' ? t.stars : t.premium;

  // ─── Recipient lookup — fires only on blur / Enter ─────────────────────────
  const [recipientLoading, setRecipientLoading] = useState(false);
  const [recipientError, setRecipientError] = useState('');

  // Any edit invalidates the previously resolved info — prevents firing /quote
  // against a stale (resolved) username after the user starts typing a new one.
  useEffect(() => {
    const u = recipient.trim().replace(/^@/, '');
    if (recipientInfo && recipientInfo.username !== u) {
      setRecipientInfo(null);
      setRecipientError('');
      setError('');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recipient]);

  const lookupRecipient = async () => {
    const u = recipient.trim().replace(/^@/, '');
    if (!u || u.length < 5) { setRecipientInfo(null); setRecipientError(''); return; }
    if (recipientInfo?.ok && recipientInfo.username === u) return;
    setRecipientError('');
    // Own username: resolve locally from Mini App data without hitting the worker.
    if (u === ownUsername && ownUsername) {
      setRecipientInfo({ ok: true, username: u, display_name: ownDisplayName, photo_url: ownPhoto });
      setRecipientEditing(false);
      return;
    }
    setRecipientLoading(true);
    try {
      const res = await productApi.checkUsername(u, type);
      setRecipientInfo(res.data);
      if (res.data.ok) {
        setRecipientEditing(false);
      } else {
        setRecipientError(labels.recipientNotFound.replace('{username}', u));
      }
    } catch {
      setRecipientInfo(null);
      setRecipientError(labels.recipientNotFound.replace('{username}', u));
    } finally {
      setRecipientLoading(false);
    }
  };

  // ─── Quote (debounced; needs both quantity AND a resolved recipient) ───────
  useEffect(() => {
    const u = recipient.trim().replace(/^@/, '');
    const minQty = type === 'stars' ? 50 : 1;
    if (!quantity || quantity < minQty || !u || !recipientInfo?.ok) {
      setQuote(null);
      return;
    }
    const handle = setTimeout(async () => {
      setQuoteLoading(true);
      try {
        const res = type === 'stars'
          ? await productApi.starsQuote(quantity, u)
          : await productApi.premiumQuote(quantity, u);
        setQuote(res.data);
        setError('');
      } catch (e: any) {
        setQuote(null);
        setError(friendlyError(e?.response?.data?.error || '', labels));
      } finally { setQuoteLoading(false); }
    }, 300);
    return () => clearTimeout(handle);
  }, [quantity, type, recipient, recipientInfo?.ok, labels.failedQuote]);

  // ─── Methods ───────────────────────────────────────────────────────────────
  const methodList: PaymentMethod[] = useMemo(() => [
    { id: 'sbp', label: 'RUB (СБП)', icon: '/sbp.jpg' },
    { id: 'ton', label: 'TON', icon: '/toncoin.jpg', disabled: !wallet, badge: !wallet ? undefined : undefined },
    { id: 'cryptobot', label: t.payment.cryptobotOther ?? 'Другая криптовалюта', icon: '/cryptobot.jpg' },
  ], [t.payment, wallet]);

  // ─── Buy ───────────────────────────────────────────────────────────────────
  const handleBuy = async () => {
    if (!user || processing) return;
    setError('');
    const u = recipient.trim().replace(/^@/, '');
    if (!u || u.length < 5) { setError(labels.invalidRecipient); return; }
    if (type === 'stars' && (!quantity || quantity < 50)) { setError(labels.minStars); return; }
    setProcessing(true);
    try {
      try {
        const check = await productApi.checkUsername(u, type);
        if (!check.data.ok) {
          setError(check.data.reason || labels.invalidRecipient);
          return;
        }
      } catch { /* worker may be down — backend will revalidate */ }

      const res = await productApi.createOrder({
        type, recipient: u, quantity, method,
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
          if (s.data.status === 'failed') { setError(friendlyError(s.data.error || '', labels)); setShowSuccess(null); break; }
        } catch { /* ignore */ }
      }
    } catch (e: any) {
      if (e?.message !== 'Reject request') setError(friendlyError(e?.response?.data?.error || '', labels));
    } finally {
      setProcessing(false);
    }
  };

  // ─── Render ────────────────────────────────────────────────────────────────
  const heroSrc = type === 'stars' ? '/stickers/buy_stars.webp' : '/stickers/buy_premium.webp';
  const resolvedRecipient = recipientInfo?.ok ? recipientInfo : null;
  const showResolved = !recipientEditing && !!resolvedRecipient && recipient.trim().length > 0;

  return (
    <div className="spend-scope px-1 pt-2 pb-8 sm:pt-4">
      {/* Tab pill */}
      <div className="flex justify-center mb-6">
        <div className="spend-tabs">
          <Link to="/stars" className={`spend-tab ${type === 'stars' ? 'spend-tab--active' : ''}`}>Stars</Link>
          <Link to="/premium" className={`spend-tab ${type === 'premium' ? 'spend-tab--active' : ''}`}>Premium</Link>
        </div>
      </div>

      {/* Hero gif */}
      <div className="flex justify-center mb-4">
        <img src={heroSrc} alt="" className="w-32 h-32 sm:w-40 sm:h-40 object-contain" />
      </div>

      {/* Heading */}
      <div className="mx-auto max-w-md mb-6 px-1 text-center">
        <h1 className="text-2xl sm:text-3xl font-bold leading-tight">{labels.title}</h1>
        {labels.subtitle && (
          <p className="mt-2 text-[#8b93b3] text-sm sm:text-base leading-relaxed">{labels.subtitle}</p>
        )}
      </div>

      {/* Form */}
      <div className="mx-auto max-w-md flex flex-col gap-3">
        {/* Recipient — resolved chip OR editable input */}
        {showResolved ? (
          <button
            type="button"
            onClick={() => {
              setRecipientEditing(true);
              setTimeout(() => recipientInputRef.current?.focus(), 0);
            }}
            className="spend-pill"
          >
            <Avatar photo={resolvedRecipient!.photo_url} fallback={resolvedRecipient!.username} />
            <span className="spend-pill__label truncate">
              {resolvedRecipient!.display_name || `@${resolvedRecipient!.username}`}
            </span>
            <span className="spend-pill__value-right">@{resolvedRecipient!.username}</span>
          </button>
        ) : (
          <div>
            {recipientError && (
              <p className="text-xs text-rose-400 mb-1.5 px-1">{recipientError}</p>
            )}
            <div className={`spend-pill ${recipientError ? 'spend-pill--error' : ''}`}>
              <span className="spend-pill__icon">@</span>
              <input
                ref={recipientInputRef}
                type="text"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                value={recipient.replace(/^@/, '')}
                placeholder={labels.recipientPlaceholder}
                onChange={(e) => setRecipient(e.target.value.replace(/^@/, '').trim())}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    (e.target as HTMLInputElement).blur();
                    lookupRecipient();
                  }
                }}
                onBlur={() => {
                  if (!recipient && ownUsername) {
                    setRecipient(ownUsername);
                    return;
                  }
                  lookupRecipient();
                }}
              />
              {recipientLoading && (
                <span className="spend-pill__value-right">…</span>
              )}
              {!recipientLoading && ownUsername && recipient !== ownUsername && (
                <button
                  type="button"
                  onClick={() => { setRecipient(ownUsername); setRecipientEditing(false); }}
                  className="spend-pill__value-right hover:text-white"
                >
                  {labels.recipientUseSelf}
                </button>
              )}
            </div>
          </div>
        )}

        {/* Method */}
        <PaymentMethodPicker
          methods={methodList}
          value={method}
          onChange={(v) => setMethod(v as PayMethod)}
          placeholder={t.payment.selectMethod}
        />

        {/* Amount (Stars: simple input; Premium: dropdown 3/6/12) */}
        {type === 'stars' ? (
          <div className="spend-pill">
            <span className="spend-pill__icon"><StarIcon color="#f4d03f" /></span>
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              value={starsText}
              onChange={(e) => {
                const raw = e.target.value.replace(/[^0-9]/g, '');
                setStarsText(raw);
                const n = raw === '' ? 0 : parseInt(raw, 10);
                setQuantity(Number.isFinite(n) ? n : 0);
              }}
              placeholder="50"
            />
            <span className="spend-pill__value-right">
              {quoteLoading ? '…' : quote ? formatMethodPrice(quote, method) : ''}
            </span>
          </div>
        ) : (
          <div>
            <button
              type="button"
              className="spend-pill"
              aria-expanded={premiumOpen ? 'true' : 'false'}
              onClick={() => setPremiumOpen((v) => !v)}
            >
              <span className="spend-pill__icon"><StarIcon color="#a48bff" /></span>
              <span className="spend-pill__label">{quantity} {labels.monthsShort}</span>
              <span className="spend-pill__value-right">
                {quoteLoading ? '…' : quote ? formatMethodPrice(quote, method) : ''}
              </span>
              <ChevronDown open={premiumOpen} />
            </button>
            {premiumOpen && (
              <div className="spend-sheet">
                {PREMIUM_OPTIONS.map((q) => (
                  <button
                    key={q}
                    type="button"
                    onClick={() => { setQuantity(q); setPremiumOpen(false); }}
                    className={`spend-sheet__item ${quantity === q ? 'spend-sheet__item--active' : ''}`}
                  >
                    <span className="flex-1">{q} {labels.monthsShort}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
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
          {processing
            ? t.payment.processing
            : `${labels.buyButton} ${type === 'stars' ? 'Stars' : 'Premium'}${quote ? ` (${formatCtaPrice(quote, method)})` : ''}`}
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
