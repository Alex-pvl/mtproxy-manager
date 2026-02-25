import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { proxyApi } from '../api/client';
import type { Proxy } from '../api/client';
import { useAuth } from '../context/AuthContext';
import CreateProxyModal from '../components/CreateProxyModal';
import Sticker from '../components/Sticker';

export default function Proxies() {
  const { user } = useAuth();
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState<string | null>(null);

  const fetchProxies = useCallback(async () => {
    try {
      const res = await proxyApi.list();
      setProxies(res.data);
    } catch {
      setError('Failed to load proxies');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchProxies();
  }, [fetchProxies]);

  const handleStart = async (id: number) => {
    setActionLoading(id);
    try {
      await proxyApi.start(id);
      await fetchProxies();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to start proxy');
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
      setError(err.response?.data?.error || 'Failed to stop proxy');
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this proxy?')) return;
    setActionLoading(id);
    try {
      await proxyApi.delete(id);
      await fetchProxies();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete proxy');
    } finally {
      setActionLoading(null);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 1500);
  };

  if (loading) {
    return <div className="text-gray-400">Loading...</div>;
  }

  const sub = user?.subscription;
  const isAdmin = user?.role === 'admin';
  const canCreate = isAdmin || sub?.active;

  return (
    <div>
      {!isAdmin && !sub?.active && (
        <div className="bg-amber-500/10 border border-amber-500/20 rounded-lg px-4 py-3 mb-6">
          <p className="text-amber-400 text-sm text-center">
            Для создания прокси необходима активная подписка.{' '}
            <Link to="/pricing" className="text-indigo-400 hover:text-indigo-300 font-medium underline">
              Выбрать тариф
            </Link>
          </p>
        </div>
      )}

      {!isAdmin && sub?.active && (
        <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-lg px-4 py-3 mb-6">
          <p className="text-emerald-400 text-sm text-center">
            Подписка: <span className="font-semibold">{sub.plan_name}</span>
            {sub.expires_at && (
              <span className="text-emerald-500 ml-2">
                до {new Date(sub.expires_at).toLocaleDateString('ru-RU')}
              </span>
            )}
          </p>
        </div>
      )}

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-white">Мои прокси</h1>
          <p className="text-sm text-gray-400 mt-1">
            {proxies.length} / {user?.max_proxies} прокси
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          disabled={!canCreate}
          className="bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium rounded px-4 py-2 transition-colors"
        >
          + Создать прокси
        </button>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-400 text-sm rounded px-3 py-2 mb-4">
          {error}
          <button onClick={() => setError('')} className="ml-2 text-red-300 hover:text-white">&times;</button>
        </div>
      )}

      {proxies.length === 0 ? (
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-12 text-center">
          <p className="text-gray-400 mb-4">Прокси не найдены</p>
        </div>
      ) : (
        <div className="grid gap-4">
          {proxies.map((proxy) => (
            <div
              key={proxy.id}
              className="bg-gray-900 border border-gray-800 rounded-lg p-4 overflow-hidden"
            >
              <div className="flex items-center justify-between flex-wrap gap-2 mb-3">
                <div className="flex items-center gap-3">
                  <span className="font-mono text-white text-sm">:{proxy.port}</span>
                  <span
                    className={`text-xs px-2 py-0.5 rounded font-medium ${
                      proxy.status === 'running'
                        ? 'bg-emerald-500/15 text-emerald-400'
                        : proxy.status === 'stopped'
                        ? 'bg-gray-500/15 text-gray-400'
                        : 'bg-red-500/15 text-red-400'
                    }`}
                  >
                    {proxy.status}
                  </span>
                  <span className="text-sm text-gray-500">{proxy.domain}</span>
                </div>

                <div className="flex items-center gap-2">
                  {proxy.status === 'running' ? (
                    <button
                      onClick={() => handleStop(proxy.id)}
                      disabled={actionLoading === proxy.id}
                      className="text-xs bg-gray-800 hover:bg-gray-700 text-gray-300 rounded px-3 py-1.5 transition-colors disabled:opacity-50"
                    >
                      Stop
                    </button>
                  ) : (
                    <button
                      onClick={() => handleStart(proxy.id)}
                      disabled={actionLoading === proxy.id}
                      className="text-xs bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-400 rounded px-3 py-1.5 transition-colors disabled:opacity-50"
                    >
                      Start
                    </button>
                  )}
                  <button
                    onClick={() => handleDelete(proxy.id)}
                    disabled={actionLoading === proxy.id}
                    className="text-xs bg-red-600/10 hover:bg-red-600/20 text-red-400 rounded px-3 py-1.5 transition-colors disabled:opacity-50"
                  >
                    Delete
                  </button>
                </div>
              </div>

              {proxy.link && (
                <div className="mt-2">
                  <div className="flex items-center justify-between gap-2 mb-1">
                    <span className="text-xs text-gray-500">
                      Link
                      {!isAdmin && sub?.active && sub?.expires_at && (
                        <span className="text-gray-400 ml-1.5">
                          (До {new Date(sub.expires_at).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })})
                        </span>
                      )}
                    </span>
                    <button
                      onClick={() => copyToClipboard(proxy.link!, `link-${proxy.id}`)}
                      className="text-xs text-indigo-400 hover:text-indigo-300 whitespace-nowrap transition-colors"
                    >
                      {copied === `link-${proxy.id}` ? 'Скопировано!' : 'Копировать'}
                    </button>
                  </div>
                  <code className="text-xs text-gray-400 bg-gray-800 rounded px-2 py-1.5 block overflow-x-auto whitespace-nowrap">
                    {proxy.link}
                  </code>
                </div>
              )}

              <div className="mt-2">
                <div className="flex items-center justify-between gap-2 mb-1">
                  <span className="text-xs text-gray-500">Secret:</span>
                  <button
                    onClick={() => copyToClipboard(proxy.secret, `secret-${proxy.id}`)}
                    className="text-xs text-indigo-400 hover:text-indigo-300 whitespace-nowrap transition-colors"
                  >
                    {copied === `secret-${proxy.id}` ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <code className="text-xs text-gray-400 bg-gray-800 rounded px-2 py-1.5 block overflow-x-auto whitespace-nowrap">
                  {proxy.secret}
                </code>
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreate && (
        <CreateProxyModal
          onClose={() => setShowCreate(false)}
          onCreated={() => {
            setShowCreate(false);
            fetchProxies();
          }}
        />
      )}
    </div>
  );
}
