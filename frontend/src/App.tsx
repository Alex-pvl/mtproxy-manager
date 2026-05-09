import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { TonConnectUIProvider } from '@tonconnect/ui-react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider } from './context/ThemeContext';
import { LanguageProvider } from './context/LanguageContext';
import Layout from './components/Layout';
import type { ReactNode } from 'react';

const MANIFEST_URL = `${window.location.origin}/tonconnect-manifest.json`;
const Home = lazy(() => import('./pages/Home'));
const Proxies = lazy(() => import('./pages/Proxies'));
const Admin = lazy(() => import('./pages/Admin'));
const Pricing = lazy(() => import('./pages/Pricing'));
const Profile = lazy(() => import('./pages/Profile'));
const Referral = lazy(() => import('./pages/Referral'));
const SbpAgreement = lazy(() => import('./pages/SbpAgreement'));
const Stars = lazy(() => import('./pages/Stars'));
const Premium = lazy(() => import('./pages/Premium'));

function PageFallback() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex items-center justify-center text-gray-500 dark:text-gray-400">
      Loading...
    </div>
  );
}

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
              <Suspense fallback={<PageFallback />}>
                <Routes>
                  <Route element={<Layout />}>
                    <Route path="/" element={<Home />} />
                    <Route path="/proxies" element={<Proxies />} />
                    <Route path="/pricing" element={<Pricing />} />
                  <Route path="/stars" element={<Stars />} />
                  <Route path="/premium" element={<Premium />} />
                    <Route path="/profile" element={<Profile />} />
                    <Route path="/referral" element={<Referral />} />
                    <Route path="/legal/sbp" element={<SbpAgreement />} />
                    <Route path="/admin" element={<AdminRoute><Admin /></AdminRoute>} />
                  </Route>
                  <Route path="*" element={<Navigate to="/" />} />
                </Routes>
              </Suspense>
            </AuthProvider>
          </BrowserRouter>
        </LanguageProvider>
      </ThemeProvider>
    </TonConnectUIProvider>
  );
}
