import { useState } from 'react';

type Props = {
  text: string;
  type: 'mtproxy' | 'socks5';
};

export function BlurredLink({ text, type }: Props) {
  const [revealed, setRevealed] = useState(false);

  const spoilerStyle = revealed
    ? {}
    : {
        color: 'transparent',
        textShadow: '0 0 18px rgba(156, 163, 175, 0.85)',
        userSelect: 'none' as const,
        transition: 'color 0.15s ease, text-shadow 0.15s ease',
      };

  const SpoilerSpan = ({
    children,
    className = '',
  }: {
    children: string;
    className?: string;
  }) => (
    <span className={`relative inline-block ${className}`}>
      <span
        className={revealed ? '' : 'select-none'}
        style={{ ...spoilerStyle, transition: 'color 0.15s ease, text-shadow 0.15s ease' }}
      >
        {children}
      </span>
      {!revealed && (
        <span
          className="absolute inset-0 pointer-events-none min-w-[8px] min-h-[1em]"
          style={{
            backgroundImage: `radial-gradient(circle, rgba(148, 163, 184, 0.8) 1px, transparent 1px)`,
            backgroundSize: '2.5px 2.5px',
          }}
          aria-hidden
        />
      )}
    </span>
  );

  if (type === 'mtproxy') {
    const match = text.match(/^(.*?secret=)([^&]+)(.*)$/);
    if (!match) return <span>{text}</span>;
    const [, prefix, secret, suffix] = match;
    return (
      <span>
        {prefix}
        <SpoilerSpan>{secret}</SpoilerSpan>
        {suffix}
        <button
          type="button"
          onClick={() => setRevealed((v) => !v)}
          className="ml-2 text-indigo-400 hover:text-indigo-300 text-xs"
        >
          {revealed ? 'Скрыть' : 'Показать'}
        </button>
      </span>
    );
  }

  if (type === 'socks5') {
    const userMatch = text.match(/user=([^&]*)/);
    const passMatch = text.match(/pass=([^&]*)/);
    if (!userMatch || !passMatch) return <span>{text}</span>;

    const userVal = userMatch[1];
    const passVal = passMatch[1];
    const beforeUser = text.substring(0, userMatch.index! + 5);
    const between = text.substring(userMatch.index! + 5 + userVal.length, passMatch.index! + 5);
    const afterPass = text.substring(passMatch.index! + 5 + passVal.length);

    return (
      <span>
        {beforeUser}
        <SpoilerSpan>{userVal}</SpoilerSpan>
        {between}
        <SpoilerSpan>{passVal}</SpoilerSpan>
        {afterPass}
        <button
          type="button"
          onClick={() => setRevealed((v) => !v)}
          className="ml-2 text-indigo-400 hover:text-indigo-300 text-xs"
        >
          {revealed ? 'Скрыть' : 'Показать'}
        </button>
      </span>
    );
  }

  return <span>{text}</span>;
}
