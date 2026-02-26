import { useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import ReferralModal from './ReferralModal';

function NavLink({ to, children, className = '', onClick }: { to: string; children: React.ReactNode; className?: string; onClick?: () => void }) {
  const location = useLocation();
  const isActive = location.pathname === to;
  return (
    <Link
      to={to}
      onClick={onClick}
      className={`text-sm transition-colors whitespace-nowrap ${isActive ? 'text-white font-medium' : 'text-gray-400 hover:text-white'} ${className}`}
    >
      {children}
    </Link>
  );
}

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [referralOpen, setReferralOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
    setMenuOpen(false);
  };

  const navLinks = (
    <>
      <NavLink to="/proxies" onClick={() => setMenuOpen(false)}>
        My
      </NavLink>
      <NavLink to="/pricing" onClick={() => setMenuOpen(false)}>
        Tariffs
      </NavLink>
      <button
        type="button"
        onClick={() => { setReferralOpen(true); setMenuOpen(false); }}
        className="text-sm text-gray-400 hover:text-white transition-colors whitespace-nowrap text-left"
      >
        Refs
      </button>
      {user?.role === 'admin' && (
        <NavLink to="/admin" onClick={() => setMenuOpen(false)}>
          Admin
        </NavLink>
      )}
    </>
  );

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 flex flex-col">
      <ReferralModal isOpen={referralOpen} onClose={() => setReferralOpen(false)} />
      <nav className="border-b border-gray-800 bg-gray-900 sticky top-0 z-40">
        <div className="mx-auto max-w-6xl px-3 sm:px-4 py-3 flex items-center justify-between gap-2">
          <div className="flex items-center gap-3 sm:gap-6 min-w-0">
            <Link to="/" className="text-base sm:text-lg font-bold text-white tracking-tight whitespace-nowrap">
              TelegramProxy
            </Link>
            <div className="hidden md:flex items-center gap-3 sm:gap-6">
              {navLinks}
            </div>
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
              className="hidden md:block text-sm text-gray-400 hover:text-white transition-colors whitespace-nowrap"
            >
              Logout
            </button>
            <button
              type="button"
              onClick={() => setMenuOpen((o) => !o)}
              className="md:hidden p-2 -mr-2 text-gray-400 hover:text-white transition-colors touch-manipulation"
              aria-label="Меню"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                {menuOpen ? (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                ) : (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                )}
              </svg>
            </button>
          </div>
        </div>
        {menuOpen && (
          <div className="md:hidden border-t border-gray-800 bg-gray-900 px-4 py-3 flex flex-col gap-3">
            {navLinks}
            <div className="flex items-center justify-between pt-2 border-t border-gray-800">
              <span className="text-sm text-gray-400">{user?.username}</span>
              <button onClick={handleLogout} className="text-sm text-gray-400 hover:text-white transition-colors py-2 touch-manipulation">
                Logout
              </button>
            </div>
          </div>
        )}
      </nav>
      <main className="mx-auto max-w-6xl w-full px-3 sm:px-4 py-4 sm:py-8 flex-1">
        <Outlet />
      </main>
      <footer className="border-t border-gray-800 bg-gray-900 py-4">
        <div className="mx-auto max-w-6xl px-4 text-center text-xs text-gray-500">
          Обратная связь и вопросы:{' '}
          <a
            href="https://t.me/oddwallet"
            target="_blank"
            rel="noopener noreferrer"
            className="text-indigo-400 hover:text-indigo-300 transition-colors"
          >
            @oddwallet
          </a>
        </div>
      </footer>
    </div>
  );
}
