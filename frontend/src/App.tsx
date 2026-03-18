import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { TonConnectUIProvider } from '@tonconnect/ui-react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider } from './context/ThemeContext';
import { LanguageProvider } from './context/LanguageContext';
import Layout from './components/Layout';
import Home from './pages/Home';
import Proxies from './pages/Proxies';
import Admin from './pages/Admin';
import Pricing from './pages/Pricing';
import Profile from './pages/Profile';
import Referral from './pages/Referral';
import type { ReactNode } from 'react';

const MANIFEST_URL = `${window.location.origin}/tonconnect-manifest.json`;

function AdminRoute({ children }: { children: ReactNode }) {
  const { user, isLoading } = useAuth();
  if (isLoading) return <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex items-center justify-center text-gray-500 dark:text-gray-400">Loading...</div>;
  if (!user) return <Navigate to="/" />;
  if (user.role !== 'admin') return <Navigate to="/" />;
  return <>{children}</>;
}

export default function App() {
  return (
    <TonConnectUIProvider manifestUrl={MANIFEST_URL}>
      <ThemeProvider>
        <LanguageProvider>
          <BrowserRouter>
            <AuthProvider>
              <Routes>
                <Route element={<Layout />}>
                  <Route path="/" element={<Home />} />
                  <Route path="/proxies" element={<Proxies />} />
                  <Route path="/pricing" element={<Pricing />} />
                  <Route path="/profile" element={<Profile />} />
                  <Route path="/referral" element={<Referral />} />
                  <Route path="/admin" element={<AdminRoute><Admin /></AdminRoute>} />
                </Route>
                <Route path="*" element={<Navigate to="/" />} />
              </Routes>
            </AuthProvider>
          </BrowserRouter>
        </LanguageProvider>
      </ThemeProvider>
    </TonConnectUIProvider>
  );
}
