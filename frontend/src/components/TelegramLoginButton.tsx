import { useCallback } from 'react';

export interface TelegramUser {
  id: number;
  first_name: string;
  last_name?: string;
  username?: string;
  photo_url?: string;
  auth_date: number;
  hash: string;
}

declare global {
  interface Window {
    Telegram?: {
      Login: {
        auth: (
          options: { bot_id: number; request_access?: string; lang?: string },
          callback: (result: false | TelegramUser) => void,
        ) => void;
      };
    };
  }
}

interface TelegramLoginButtonProps {
  label: string;
  onAuth: (user: TelegramUser) => void;
  onError: (error: string) => void;
  disabled?: boolean;
}

const TG_BOT_ID = import.meta.env.VITE_TG_CLIENT_ID
  ? Number(import.meta.env.VITE_TG_CLIENT_ID)
  : 0;

export default function TelegramLoginButton({
  label,
  onAuth,
  onError,
  disabled,
}: TelegramLoginButtonProps) {
  const handleClick = useCallback(() => {
    if (!window.Telegram?.Login) {
      onError('Telegram Login SDK not loaded');
      return;
    }

    if (!TG_BOT_ID) {
      onError('Telegram Bot ID not configured');
      return;
    }

    window.Telegram.Login.auth(
      { bot_id: TG_BOT_ID, request_access: 'write' },
      (result) => {
        if (!result) {
          onError('Telegram login cancelled');
          return;
        }
        onAuth(result);
      },
    );
  }, [onAuth, onError]);

  if (!TG_BOT_ID) return null;

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={disabled}
      className="w-full flex items-center justify-center gap-2 bg-[#54a9eb] hover:bg-[#4a96d2] disabled:opacity-50 text-white text-sm font-medium rounded px-4 py-2.5 transition-colors touch-manipulation"
    >
      <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
        <path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.479.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/>
      </svg>
      {label}
    </button>
  );
}
