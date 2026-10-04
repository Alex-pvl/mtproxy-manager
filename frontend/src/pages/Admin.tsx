import { useState, useEffect, useCallback } from 'react';
import { adminApi, apiError } from '../api/client';
import type { UserWithCount, Proxy } from '../api/client';

export default function Admin() {
  const [users, setUsers] = useState<UserWithCount[]>([]);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [tab, setTab] = useState<'users' | 'proxies'>('users');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editingUser, setEditingUser] = useState<number | null>(null);
  const [editMaxProxies, setEditMaxProxies] = useState('');
  const [editRole, setEditRole] = useState('');

  const fetchData = useCallback(
    () =>
      Promise.all([adminApi.listUsers(), adminApi.listProxies()])
        .then(([usersRes, proxiesRes]) => {
          setUsers(usersRes.data);
          setProxies(proxiesRes.data);
        })
        .catch(() => setError('Failed to load data'))
        .finally(() => setLoading(false)),
    [],
  );

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleEditUser = (user: UserWithCount) => {
    setEditingUser(user.id);
    setEditMaxProxies(String(user.max_proxies));
    setEditRole(user.role);
  };

  const handleSaveUser = async (id: number) => {
    try {
      await adminApi.updateUser(id, {
        role: editRole,
        max_proxies: parseInt(editMaxProxies),
      });
      setEditingUser(null);
      fetchData();
    } catch (err) {
      setError(apiError(err, 'Failed to update user'));
    }
  };

  const handleToggleSubBanner = async (user: UserWithCount) => {
    try {
      await adminApi.updateUser(user.id, { hide_sub_banner: !user.hide_sub_banner });
      fetchData();
    } catch (err) {
      setError(apiError(err, 'Failed to update user'));
    }
  };

  const handleDeleteUser = async (id: number) => {
    if (!confirm('Delete this user and all their connections?')) return;
    try {
      await adminApi.deleteUser(id);
      fetchData();
    } catch (err) {
      setError(apiError(err, 'Failed to delete user'));
    }
  };

  const handleDeleteProxy = async (id: number) => {
    if (!confirm('Delete this connection?')) return;
    try {
      await adminApi.deleteProxy(id);
      fetchData();
    } catch (err) {
      setError(apiError(err, 'Failed to delete connection'));
    }
  };

  if (loading) {
    return <div className="text-gray-500 dark:text-gray-400">Loading...</div>;
  }

  return (
    <div>
      <h1 className="text-xl font-bold text-gray-900 dark:text-white mb-6">Admin Panel</h1>

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 text-sm rounded px-3 py-2 mb-4">
          {error}
          <button onClick={() => setError('')} className="ml-2 text-red-500 dark:text-red-300 hover:text-gray-900 dark:hover:text-white">&times;</button>
        </div>
      )}

      <div className="flex gap-1 mb-6">
        <button
          onClick={() => setTab('users')}
          className={`text-sm px-4 py-2 rounded transition-colors ${
            tab === 'users'
              ? 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-white'
              : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
          }`}
        >
          Users ({users.length})
        </button>
        <button
          onClick={() => setTab('proxies')}
          className={`text-sm px-4 py-2 rounded transition-colors ${
            tab === 'proxies'
              ? 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-white'
              : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
          }`}
        >
          All connections ({proxies.length})
        </button>
      </div>

      {tab === 'users' && (
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-800 text-gray-500 dark:text-gray-400">
                <th className="text-left px-4 py-3 font-medium">ID</th>
                <th className="text-left px-4 py-3 font-medium">Username</th>
                <th className="text-left px-4 py-3 font-medium">Role</th>
                <th className="text-left px-4 py-3 font-medium">Connections</th>
                <th className="text-left px-4 py-3 font-medium">Limit</th>
                <th className="text-left px-4 py-3 font-medium">Sub banner</th>
                <th className="text-left px-4 py-3 font-medium">Created</th>
                <th className="text-right px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr key={user.id} className="border-b border-gray-100 dark:border-gray-800/50 hover:bg-gray-50 dark:hover:bg-gray-800/30">
                  <td className="px-4 py-3 text-gray-500 dark:text-gray-400">{user.id}</td>
                  <td className="px-4 py-3 text-gray-900 dark:text-white">{user.username}</td>
                  <td className="px-4 py-3">
                    {editingUser === user.id ? (
                      <select
                        value={editRole}
                        onChange={(e) => setEditRole(e.target.value)}
                        className="bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded px-2 py-1 text-gray-900 dark:text-white text-xs"
                      >
                        <option value="user">user</option>
                        <option value="admin">admin</option>
                      </select>
                    ) : (
                      <span className={`text-xs px-2 py-0.5 rounded ${
                        user.role === 'admin'
                          ? 'bg-indigo-100 dark:bg-indigo-600/30 text-indigo-700 dark:text-indigo-300'
                          : 'bg-gray-100 dark:bg-gray-700/50 text-gray-700 dark:text-gray-300'
                      }`}>
                        {user.role}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-700 dark:text-gray-300">{user.proxy_count}</td>
                  <td className="px-4 py-3">
                    {editingUser === user.id ? (
                      <input
                        type="number"
                        value={editMaxProxies}
                        onChange={(e) => setEditMaxProxies(e.target.value)}
                        className="bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded px-2 py-1 text-gray-900 dark:text-white text-xs w-16"
                        min={0}
                      />
                    ) : (
                      <span className="text-gray-700 dark:text-gray-300">{user.max_proxies}</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => handleToggleSubBanner(user)}
                      title="Show the active subscription banner on the pricing page"
                      className={`text-xs px-2 py-0.5 rounded transition-colors ${
                        user.hide_sub_banner
                          ? 'bg-gray-100 dark:bg-gray-700/50 text-gray-500 dark:text-gray-400'
                          : 'bg-emerald-100 dark:bg-emerald-600/20 text-emerald-700 dark:text-emerald-300'
                      }`}
                    >
                      {user.hide_sub_banner ? 'hidden' : 'shown'}
                    </button>
                  </td>
                  <td className="px-4 py-3 text-gray-400 dark:text-gray-500 text-xs">
                    {new Date(user.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {editingUser === user.id ? (
                      <div className="flex gap-2 justify-end">
                        <button
                          onClick={() => handleSaveUser(user.id)}
                          className="text-xs bg-indigo-600/20 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-600/30 rounded px-2 py-1 transition-colors"
                        >
                          Save
                        </button>
                        <button
                          onClick={() => setEditingUser(null)}
                          className="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <div className="flex gap-2 justify-end">
                        <button
                          onClick={() => handleEditUser(user)}
                          className="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => handleDeleteUser(user.id)}
                          className="text-xs text-red-600 dark:text-red-400 hover:text-red-500 dark:hover:text-red-300 transition-colors"
                        >
                          Delete
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'proxies' && (
        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-lg overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-800 text-gray-500 dark:text-gray-400">
                <th className="text-left px-4 py-3 font-medium">ID</th>
                <th className="text-left px-4 py-3 font-medium">User</th>
                <th className="text-left px-4 py-3 font-medium">Name</th>
                <th className="text-left px-4 py-3 font-medium">Link</th>
                <th className="text-right px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {proxies.map((proxy) => (
                <tr key={proxy.id} className="border-b border-gray-100 dark:border-gray-800/50 hover:bg-gray-50 dark:hover:bg-gray-800/30">
                  <td className="px-4 py-3 text-gray-500 dark:text-gray-400">{proxy.id}</td>
                  <td className="px-4 py-3 text-gray-700 dark:text-gray-300">{proxy.user_id}</td>
                  <td className="px-4 py-3 font-mono text-gray-900 dark:text-white">{proxy.name}</td>
                  <td className="px-4 py-3">
                    {proxy.link_sub && (
                      <button
                        onClick={() => navigator.clipboard.writeText(proxy.link_sub!)}
                        className="text-xs text-indigo-600 dark:text-indigo-400 hover:text-indigo-500 dark:hover:text-indigo-300 transition-colors"
                      >
                        Copy link
                      </button>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => handleDeleteProxy(proxy.id)}
                      className="text-xs text-red-600 dark:text-red-400 hover:text-red-500 dark:hover:text-red-300 transition-colors"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
              {proxies.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-gray-400 dark:text-gray-500">
                    No connections
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
