import { useState, type ReactNode } from 'react';

interface StickerProps {
  name: string;
  className?: string;
  children?: ReactNode;
}

const FALLBACK: Record<string, string> = {
  lost_access: '📵',
  no_logs: '🛡️',
  cipher: '🔒',
  speed: '⚡',
  referals: '👥',
  tariffs: '💰',
};

/** Стикер по имени. Поддерживает /stickers/{name}.webp или эмодзи-fallback */
export default function Sticker({ name, className = '', children }: StickerProps) {
  const [imgFailed, setImgFailed] = useState(false);
  const src = `/stickers/${name}.webp`;
  const showImg = !imgFailed && !children;

  return (
    <span
      className={`inline-flex items-center justify-center shrink-0 overflow-hidden rounded-xl bg-gray-800/50 ${className}`}
      data-sticker={name}
    >
      {children}
      {!children && showImg && (
        <img
          src={src}
          alt=""
          className="w-full h-full object-contain"
          onError={() => setImgFailed(true)}
        />
      )}
      {!children && imgFailed && (
        <span className="text-2xl" aria-hidden>
          {FALLBACK[name] ?? '✨'}
        </span>
      )}
    </span>
  );
}
