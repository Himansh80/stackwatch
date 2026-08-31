import { Navigate, Route, Routes } from 'react-router-dom';
import AppShell from './components/AppShell';
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
import CicdPage from './pages/CicdPage';
import DatabasePage from './pages/DatabasePage';
import IncidentsPage from './pages/IncidentsPage';
import NotebookPage from './pages/NotebookPage';
import SharedDashboardsPage from './pages/SharedDashboardsPage';
import IntelligencePage from './pages/IntelligencePage';
import EnterprisePage from './pages/EnterprisePage';
import HomelabPage from './pages/HomelabPage';
import PlatformPage from './pages/PlatformPage';
import ProxmoxWorkspace from './components/proxmox/ProxmoxWorkspace';
import ProxmoxVmListPage from './components/proxmox/ProxmoxVmListPage';
import ProxmoxVmDetailPage from './components/proxmox/ProxmoxVmDetailPage';
import ProxmoxLxcListPage from './components/proxmox/ProxmoxLxcListPage';
import ProxmoxLxcDetailPage from './components/proxmox/ProxmoxLxcDetailPage';
import ProxmoxLxcCreatePage from './components/proxmox/ProxmoxLxcCreatePage';
import ProxmoxVmCreatePage from './components/proxmox/ProxmoxVmCreatePage';
import ProxmoxNodeDashboard from './components/proxmox/ProxmoxNodeDashboard';
import ProxmoxStorageContentPage from './components/proxmox/ProxmoxStorageContentPage';
import ProxmoxNodeFirewall from './components/proxmox/ProxmoxNodeFirewall';
import ProxmoxIpsetManager from './components/proxmox/ProxmoxIpsetManager';
import ProxmoxAliasManager from './components/proxmox/ProxmoxAliasManager';
import ProxmoxSdnZones from './components/proxmox/ProxmoxSdnZones';
import ProxmoxUsers from './components/proxmox/ProxmoxUsers';
import ProxmoxUserTokens from './components/proxmox/ProxmoxUserTokens';
import ProxmoxPools from './components/proxmox/ProxmoxPools';
import ProxmoxTasksPanel from './components/proxmox/ProxmoxTasksPanel';
import ProxmoxBackupJobs from './components/proxmox/ProxmoxBackupJobs';
import StyleGuide from './pages/StyleGuide';
import { useAuthState } from './lib/useAuthState';

/**
 * Authenticated = AppShell wraps the page (sidebar + topbar + main).
 * Public = no shell (login/signup/landing/forgot/reset render full-screen).
 */
function auth<T>(Component: React.ComponentType<T>, props: T) {
  return <AppShell><Component {...(props as T & object)} /></AppShell>;
}

export default function App() {
  const { isLoggedIn } = useAuthState();

  return (
    <Routes>
      {/* Public routes — no shell */}
      <Route path="/" element={isLoggedIn ? auth(Dashboard, {}) : <Landing />} />
      <Route path="/login" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <Login />} />
      <Route path="/signup" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <Signup />} />
      <Route path="/forgot-password" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <ForgotPassword />} />
      <Route path="/reset-password" element={isLoggedIn ? <Navigate to="/dashboard" replace /> : <ResetPassword />} />

      {/* Authenticated routes — wrapped in AppShell */}
      <Route path="/dashboard" element={isLoggedIn ? auth(Dashboard, {}) : <Navigate to="/login" replace />} />
      <Route path="/profile" element={isLoggedIn ? auth(ProfilePage, {}) : <Navigate to="/login" replace />} />
      <Route path="/billing" element={isLoggedIn ? auth(BillingPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/settings" element={isLoggedIn ? auth(SettingsPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox" element={isLoggedIn ? auth(ProxmoxWorkspace, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox-vms" element={isLoggedIn ? auth(ProxmoxVmListPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox-vms/:hostId/:node/:vmid" element={isLoggedIn ? auth(ProxmoxVmDetailPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox-lxc" element={isLoggedIn ? auth(ProxmoxLxcListPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox-lxc/new" element={isLoggedIn ? auth(ProxmoxLxcCreatePage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox-lxc/:hostId/:node/:vmid" element={isLoggedIn ? auth(ProxmoxLxcDetailPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox-vms/new" element={isLoggedIn ? auth(ProxmoxVmCreatePage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/nodes/:hostId/:node" element={isLoggedIn ? auth(ProxmoxNodeDashboard, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/storage/:hostId/:node/:storage" element={isLoggedIn ? auth(ProxmoxStorageContentPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/nodes/:hostId/:node/firewall" element={isLoggedIn ? auth(ProxmoxNodeFirewall, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/ipsets" element={isLoggedIn ? auth(ProxmoxIpsetManager, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/aliases" element={isLoggedIn ? auth(ProxmoxAliasManager, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/sdn/zones" element={isLoggedIn ? auth(ProxmoxSdnZones, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/users" element={isLoggedIn ? auth(ProxmoxUsers, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/users/:userid/tokens" element={isLoggedIn ? auth(ProxmoxUserTokens, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/pools" element={isLoggedIn ? auth(ProxmoxPools, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/tasks" element={isLoggedIn ? auth(ProxmoxTasksPanel, {}) : <Navigate to="/login" replace />} />
      <Route path="/proxmox/backup" element={isLoggedIn ? auth(ProxmoxBackupJobs, {}) : <Navigate to="/login" replace />} />
      <Route path="/truenas" element={isLoggedIn ? auth(TrueNASWorkspace, {}) : <Navigate to="/login" replace />} />
      <Route path="/apm" element={isLoggedIn ? auth(ApmPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/apm/service" element={isLoggedIn ? auth(ApmServicePage, {}) : <Navigate to="/login" replace />} />
      <Route path="/logs" element={isLoggedIn ? auth(LogsFullPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/rum" element={isLoggedIn ? auth(RumFullPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/rum/session" element={isLoggedIn ? auth(RumSessionPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/synthetics" element={isLoggedIn ? auth(SyntheticsPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/security" element={isLoggedIn ? auth(SecurityPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/cspm" element={isLoggedIn ? auth(CspmPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/cicd" element={isLoggedIn ? auth(CicdPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/database" element={isLoggedIn ? auth(DatabasePage, {}) : <Navigate to="/login" replace />} />
      <Route path="/incidents" element={isLoggedIn ? auth(IncidentsPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/notebooks" element={isLoggedIn ? auth(NotebookPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/shared" element={isLoggedIn ? auth(SharedDashboardsPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/intelligence" element={isLoggedIn ? auth(IntelligencePage, {}) : <Navigate to="/login" replace />} />
      <Route path="/enterprise" element={isLoggedIn ? auth(EnterprisePage, {}) : <Navigate to="/login" replace />} />
      <Route path="/homelab" element={isLoggedIn ? auth(HomelabPage, {}) : <Navigate to="/login" replace />} />
      <Route path="/platform" element={isLoggedIn ? auth(PlatformPage, {}) : <Navigate to="/login" replace />} />
            {/* Tier 20 — StyleGuide (unlisted from sidebar, accessible via direct URL) */}
            <Route path="/style-guide" element={isLoggedIn ? auth(StyleGuide, {}) : <Navigate to="/login" replace />} />
            <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}