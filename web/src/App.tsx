import { Navigate, Route, Routes } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Landing from './pages/Landing';
import Login from './pages/Login';
import Signup from './pages/Signup';
import ForgotPassword from './pages/ForgotPassword';
import ResetPassword from './pages/ResetPassword';
import ProfilePage from './pages/ProfilePage';
import BillingPage from './pages/BillingPage';
import SettingsPage from './pages/SettingsPage';
import TrueNASWorkspace from './pages/TrueNASWorkspace';
import ApmPage from './pages/ApmPage';
import ApmServicePage from './pages/ApmServicePage';
import LogsFullPage from './pages/LogsFullPage';
import RumFullPage from './pages/RumFullPage';
import RumSessionPage from './pages/RumSessionPage';
import SyntheticsPage from './pages/SyntheticsPage';
import SecurityPage from './pages/SecurityPage';
import CspmPage from './pages/CspmPage';
import ProxmoxWorkspace from './components/proxmox/ProxmoxWorkspace';
import { useAuthState } from './lib/useAuthState';

export default function App() {
  // useAuthState subscribes to localStorage changes (via a custom event
  // fired by setToken / clearToken) so route guards update the same
  // render cycle the token is written. Prevents the 'signed in but
  // bounced back to /login' bug where App.tsx re-rendered before
  // localStorage was visible.
  const { isLoggedIn } = useAuthState();

  return (
    <Routes>
      <Route path="/" element={isLoggedIn ? <Dashboard /> : <Landing />} />
      <Route path="/login" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <Login />} />
      <Route path="/signup" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <Signup />} />
      <Route path="/forgot-password" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <ForgotPassword />} />
      <Route path="/reset-password" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <ResetPassword />} />
      <Route path="/dashboard" element={isLoggedIn ? <Dashboard /> : <Navigate to="/login" replace />} />
      <Route path="/profile" element={isLoggedIn ? <ProfilePage /> : <Navigate to="/login" replace />} />
      <Route path="/billing" element={isLoggedIn ? <BillingPage /> : <Navigate to="/login" replace />} />
      <Route path="/settings" element={isLoggedIn ? <SettingsPage /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox" element={isLoggedIn ? <ProxmoxWorkspace /> : <Navigate to="/login" replace />} />
      <Route path="/truenas" element={isLoggedIn ? <TrueNASWorkspace /> : <Navigate to="/login" replace />} />
      <Route path="/apm" element={isLoggedIn ? <ApmPage /> : <Navigate to="/login" replace />} />
      <Route path="/apm/service" element={isLoggedIn ? <ApmServicePage /> : <Navigate to="/login" replace />} />
      <Route path="/logs" element={isLoggedIn ? <LogsFullPage /> : <Navigate to="/login" replace />} />
      <Route path="/rum" element={isLoggedIn ? <RumFullPage /> : <Navigate to="/login" replace />} />
      <Route path="/rum/session" element={isLoggedIn ? <RumSessionPage /> : <Navigate to="/login" replace />} />
      <Route path="/synthetics" element={isLoggedIn ? <SyntheticsPage /> : <Navigate to="/login" replace />} />
      <Route path="/security" element={isLoggedIn ? <SecurityPage /> : <Navigate to="/login" replace />} />
      <Route path="/cspm" element={isLoggedIn ? <CspmPage /> : <Navigate to="/login" replace />} />
      <Route path="/tier0" element={isLoggedIn ? <Dashboard /> : <Navigate to="/login" replace />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
