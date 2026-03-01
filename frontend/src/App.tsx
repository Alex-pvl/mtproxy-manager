import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider } from './context/ThemeContext';
import { LanguageProvider } from './context/LanguageContext';
import Layout from './components/Layout';
import Login from './pages/Login';
import Register from './pages/Register';
import Home from './pages/Home';
import Proxies from './pages/Proxies';
import Admin from './pages/Admin';
import Pricing from './pages/Pricing';
import type { ReactNode } from 'react';

function AdminRoute({ children }: { children: ReactNode }) {
  const { user, isLoading } = useAuth();
  if (isLoading) return <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex items-center justify-center text-gray-500 dark:text-gray-400">Loading...</div>;
  if (!user) return <Navigate to="/login" />;
  if (user.role !== 'admin') return <Navigate to="/" />;
  return <>{children}</>;
}

function PublicRoute({ children }: { children: ReactNode }) {
  const { token, isLoading } = useAuth();
  if (isLoading) return <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex items-center justify-center text-gray-500 dark:text-gray-400">Loading...</div>;
  return token ? <Navigate to="/" /> : <>{children}</>;
}

export default function App() {
  return (
    <ThemeProvider>
      <LanguageProvider>
        <BrowserRouter>
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<PublicRoute><Login /></PublicRoute>} />
              <Route path="/register" element={<PublicRoute><Register /></PublicRoute>} />
              <Route element={<Layout />}>
                <Route path="/" element={<Home />} />
                <Route path="/proxies" element={<Proxies />} />
                <Route path="/pricing" element={<Pricing />} />
                <Route path="/admin" element={<AdminRoute><Admin /></AdminRoute>} />
              </Route>
              <Route path="*" element={<Navigate to="/" />} />
            </Routes>
          </AuthProvider>
        </BrowserRouter>
      </LanguageProvider>
    </ThemeProvider>
  );
}
