import { Link, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 flex flex-col">
      <nav className="border-b border-gray-800 bg-gray-900">
        <div className="mx-auto max-w-6xl px-4 py-3 flex items-center justify-between gap-2">
          <div className="flex items-center gap-3 sm:gap-6 min-w-0">
            <Link to="/" className="text-base sm:text-lg font-bold text-white tracking-tight whitespace-nowrap">
              MTProxy
            </Link>
            <Link to="/" className="text-sm text-gray-400 hover:text-white transition-colors whitespace-nowrap">
              Dashboard
            </Link>
            <Link to="/pricing" className="text-sm text-gray-400 hover:text-white transition-colors whitespace-nowrap">
              Тарифы
            </Link>
            {user?.role === 'admin' && (
              <Link to="/admin" className="text-sm text-gray-400 hover:text-white transition-colors whitespace-nowrap">
                Admin
              </Link>
            )}
          </div>
          <div className="flex items-center gap-2 sm:gap-4 shrink-0">
            <span className="text-sm text-gray-400 hidden sm:inline">
              {user?.username}
              {user?.role === 'admin' && (
                <span className="ml-1.5 text-xs bg-indigo-600/30 text-indigo-300 px-1.5 py-0.5 rounded">
                  admin
                </span>
              )}
            </span>
            <button
              onClick={handleLogout}
              className="text-sm text-gray-400 hover:text-white transition-colors whitespace-nowrap"
            >
              Выйти
            </button>
          </div>
        </div>
      </nav>
      <main className="mx-auto max-w-6xl w-full px-3 sm:px-4 py-4 sm:py-8 flex-1">
        <Outlet />
      </main>
      <footer className="border-t border-gray-800 bg-gray-900 py-4">
        <div className="mx-auto max-w-6xl px-4 text-center text-xs text-gray-500" />
      </footer>
    </div>
  );
}
