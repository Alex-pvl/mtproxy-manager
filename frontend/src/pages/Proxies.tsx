import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { proxyApi } from '../api/client';
import type { Proxy } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { BlurredLink } from '../components/BlurredLink';

function WifiIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M5 12.55a11 11 0 0 1 14.08 0" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M1.42 9a16 16 0 0 1 21.16 0" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M8.53 16.11a6 6 0 0 1 6.95 0" />
      <circle cx="12" cy="20" r="1" fill="currentColor" stroke="none" />
    </svg>
  );
}

function MobileIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <rect x="5" y="2" width="14" height="20" rx="2" ry="2" />
      <line x1="12" y1="18" x2="12.01" y2="18" strokeLinecap="round" strokeWidth={3} />
    </svg>
  );
}

function ShieldIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    </svg>
  );
}

export default function Proxies() {
  const { user, isLoading: authLoading } = useAuth();
  const { t } = useLanguage();
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [loading, setLoading] = useState(true);
  const [createLoading, setCreateLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState<string | null>(null);

  const fetchProxies = useCallback(async () => {
    try {
      const res = await proxyApi.list();
      setProxies(res.data);
    } catch {
      setError(t.proxies.failedLoad);
    } finally {
      setLoading(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (authLoading) return;
    if (!user) { setLoading(false); return; }
    fetchProxies();
  }, [authLoading, user, fetchProxies]);

  const handleCreate = async () => {
    setCreateLoading(true);
    setError('');
    try {
      await proxyApi.create('google.com', undefined);
      await fetchProxies();
    } catch (err: any) {
      setError(err.response?.data?.error || t.proxies.failedCreate);
    } finally {
      setCreateLoading(false);
    }
  };

  const handleStart = async (id: number) => {
    setActionLoading(id);
    try {
      await proxyApi.start(id);
      await fetchProxies();
    } catch (err: any) {
      setError(err.response?.data?.error || t.proxies.failedStart);
    } finally {
      setActionLoading(null);
    }
  };

  const handleStop = async (id: number) => {
    setActionLoading(id);
    try {
      await proxyApi.stop(id);
      await fetchProxies();
    } catch (err: any) {
      setError(err.response?.data?.error || t.proxies.failedStop);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm(t.proxies.deleteConfirm)) return;
    setActionLoading(id);
    try {
      await proxyApi.delete(id);
      await fetchProxies();
    } catch (err: any) {
      setError(err.response?.data?.error || t.proxies.failedDelete);
    } finally {
      setActionLoading(null);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 1500);
  };

  if (authLoading || loading) {
    return <div className="text-gray-500 dark:text-gray-400">{t.common.loading}</div>;
  }

  if (!user) {
    return (
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-12 text-center">
        <p className="text-gray-500 dark:text-gray-400 mb-4">{t.proxies.loginToView}</p>
        <Link
          to="/login"
          className="inline-block bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded px-5 py-2.5 transition-colors"
        >
          {t.proxies.loginToCreate}
        </Link>
      </div>
    );
  }

  const sub = user.subscription;
  const isAdmin = user.role === 'admin';
  const canCreate = isAdmin || sub?.active;

  return (
    <div>
      {!isAdmin && !sub?.active && (
        <div className="bg-amber-500/10 border border-amber-500/20 rounded-lg px-4 py-3 mb-6">
          <p className="text-amber-600 dark:text-amber-400 text-sm text-center">
            {t.proxies.needSubscription}{' '}
            <Link to="/pricing" className="text-indigo-500 dark:text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-300 font-medium underline">
              {t.proxies.choosePlan}
            </Link>
          </p>
        </div>
      )}

      {!isAdmin && sub?.active && (
        <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-lg px-4 py-3 mb-6">
          <p className="text-emerald-600 dark:text-emerald-400 text-sm text-center">
            {t.proxies.subscription}{' '}
            <span className="font-semibold">{(sub.plan_id && t.pricing.planNames[sub.plan_id]) || sub.plan_name}</span>
            {sub.expires_at && (
              <span className="text-emerald-500 ml-2">
                {t.proxies.subscriptionUntil}{' '}
                {new Date(sub.expires_at).toLocaleDateString('ru-RU')}
              </span>
            )}
          </p>
        </div>
      )}

      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h1 className="text-xl font-bold text-gray-900 dark:text-white">{t.proxies.title}</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {t.proxies.count(proxies.length, user.max_proxies ?? 0)}
          </p>
        </div>
        <button
          onClick={handleCreate}
          disabled={!canCreate || createLoading}
          className="w-full sm:w-auto bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium rounded px-4 py-2.5 transition-colors touch-manipulation"
        >
          {createLoading ? t.proxies.creating : t.proxies.createProxy}
        </button>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 dark:text-red-400 text-sm rounded px-3 py-2 mb-4">
          {error}
          <button onClick={() => setError('')} className="ml-2 text-red-400 dark:text-red-300 hover:text-red-600 dark:hover:text-white">&times;</button>
        </div>
      )}

      {proxies.length === 0 ? (
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-12 text-center">
          <p className="text-gray-500 dark:text-gray-400 mb-4">{t.proxies.noProxies}</p>
        </div>
      ) : (
        <div className="grid gap-4">
          {proxies.map((proxy) => (
            <div
              key={proxy.id}
              className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg p-4 overflow-hidden min-w-0"
            >
              {/* Card header: status + actions */}
              <div className="flex items-center justify-between gap-2 mb-3">
                <span
                  className={`text-xs px-2 py-0.5 rounded font-medium ${
                    proxy.status === 'running'
                      ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                      : proxy.status === 'stopped'
                      ? 'bg-gray-500/15 text-gray-500 dark:text-gray-400'
                      : 'bg-red-500/15 text-red-500 dark:text-red-400'
                  }`}
                >
                  {proxy.status}
                </span>
                <div className="flex items-center gap-2 shrink-0">
                  {proxy.status === 'running' ? (
                    <button
                      onClick={() => handleStop(proxy.id)}
                      disabled={actionLoading === proxy.id}
                      className="text-xs bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-300 rounded px-3 py-2 transition-colors disabled:opacity-50 touch-manipulation"
                    >
                      {t.proxies.stop}
                    </button>
                  ) : (
                    <button
                      onClick={() => handleStart(proxy.id)}
                      disabled={actionLoading === proxy.id}
                      className="text-xs bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-600 dark:text-emerald-400 rounded px-3 py-2 transition-colors disabled:opacity-50 touch-manipulation"
                    >
                      {t.proxies.start}
                    </button>
                  )}
                  <button
                    onClick={() => handleDelete(proxy.id)}
                    disabled={actionLoading === proxy.id}
                    className="text-xs bg-red-600/10 hover:bg-red-600/20 text-red-500 dark:text-red-400 rounded px-3 py-2 transition-colors disabled:opacity-50 touch-manipulation"
                  >
                    {t.proxies.delete}
                  </button>
                </div>
              </div>

              {/* MTProto link */}
              {proxy.link && (
                <div className="mt-2 rounded-lg border border-indigo-500/20 bg-indigo-500/5 p-3">
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <div className="flex items-center gap-2">
                      <WifiIcon className="w-4 h-4 text-indigo-500 dark:text-indigo-400 shrink-0" />
                      <div>
                        <span className="text-xs font-medium text-indigo-600 dark:text-indigo-400">MTProto</span>
                        <span className="text-xs text-gray-400 dark:text-gray-500 ml-1.5">{t.proxies.mtprotoHint}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <a
                        href={proxy.link}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-indigo-500 dark:text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-300 transition-colors py-1 touch-manipulation"
                      >
                        {t.proxies.open}
                      </a>
                      <button
                        onClick={() => copyToClipboard(proxy.link!, `link-${proxy.id}`)}
                        className="text-xs text-gray-400 dark:text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 whitespace-nowrap transition-colors py-1 touch-manipulation"
                      >
                        {copied === `link-${proxy.id}` ? t.proxies.copied : t.proxies.copy}
                      </button>
                    </div>
                  </div>
                  <code className="text-xs text-gray-500 dark:text-gray-400 bg-white/60 dark:bg-gray-800/60 rounded px-2 py-1.5 block overflow-x-auto break-all sm:break-normal sm:whitespace-nowrap">
                    <BlurredLink text={proxy.link} type="mtproxy" />
                  </code>
                </div>
              )}

              {/* SOCKS5 link */}
              {proxy.link_socks5 && (
                <div className="mt-2 rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-3">
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <div className="flex items-center gap-2">
                      <MobileIcon className="w-4 h-4 text-emerald-500 dark:text-emerald-400 shrink-0" />
                      <div>
                        <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400">SOCKS5</span>
                        <span className="text-xs text-gray-400 dark:text-gray-500 ml-1.5">{t.proxies.socks5Hint}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <a
                        href={proxy.link_socks5}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-emerald-500 dark:text-emerald-400 hover:text-emerald-600 dark:hover:text-emerald-300 transition-colors py-1 touch-manipulation"
                      >
                        {t.proxies.open}
                      </a>
                      <button
                        onClick={() => copyToClipboard(proxy.link_socks5!, `link-socks5-${proxy.id}`)}
                        className="text-xs text-gray-400 dark:text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 whitespace-nowrap transition-colors py-1 touch-manipulation"
                      >
                        {copied === `link-socks5-${proxy.id}` ? t.proxies.copied : t.proxies.copy}
                      </button>
                    </div>
                  </div>
                  <code className="text-xs text-gray-500 dark:text-gray-400 bg-white/60 dark:bg-gray-800/60 rounded px-2 py-1.5 block overflow-x-auto break-all sm:break-normal sm:whitespace-nowrap">
                    <BlurredLink text={proxy.link_socks5} type="socks5" />
                  </code>
                </div>
              )}

              {/* VLESS link */}
              {proxy.link_vless && (
                <div className="mt-2 rounded-lg border border-violet-500/20 bg-violet-500/5 p-3">
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <div className="flex items-center gap-2">
                      <ShieldIcon className="w-4 h-4 text-violet-500 dark:text-violet-400 shrink-0" />
                      <div>
                        <span className="text-xs font-medium text-violet-600 dark:text-violet-400">{t.proxies.vlessLabel}</span>
                        <span className="text-xs text-gray-400 dark:text-gray-500 ml-1.5">{t.proxies.vlessHint}</span>
                      </div>
                    </div>
                    <button
                      onClick={() => copyToClipboard(proxy.link_vless!, `link-vless-${proxy.id}`)}
                      className="text-xs text-gray-400 dark:text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 whitespace-nowrap transition-colors py-1 touch-manipulation shrink-0"
                    >
                      {copied === `link-vless-${proxy.id}` ? t.proxies.copied : t.proxies.copy}
                    </button>
                  </div>
                  <code className="text-xs text-gray-500 dark:text-gray-400 bg-white/60 dark:bg-gray-800/60 rounded px-2 py-1.5 block overflow-x-auto break-all sm:break-normal sm:whitespace-nowrap">
                    {proxy.link_vless}
                  </code>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
