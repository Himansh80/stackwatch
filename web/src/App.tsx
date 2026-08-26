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
      <Route path="/proxmox-vms" element={isLoggedIn ? <ProxmoxVmListPage /> : <Navigate to="/login" replace />} />
            <Route path="/proxmox-vms/:hostId/:node/:vmid" element={isLoggedIn ? <ProxmoxVmDetailPage /> : <Navigate to="/login" replace />} />
            <Route path="/proxmox-lxc" element={isLoggedIn ? <ProxmoxLxcListPage /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox-lxc/new" element={isLoggedIn ? <ProxmoxLxcCreatePage /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox-lxc/:hostId/:node/:vmid" element={isLoggedIn ? <ProxmoxLxcDetailPage /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox-vms/new" element={isLoggedIn ? <ProxmoxVmCreatePage /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/nodes/:hostId/:node" element={isLoggedIn ? <ProxmoxNodeDashboard /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/storage/:hostId/:node/:storage" element={isLoggedIn ? <ProxmoxStorageContentPage /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/nodes/:hostId/:node/firewall" element={isLoggedIn ? <ProxmoxNodeFirewall /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/ipsets" element={isLoggedIn ? <ProxmoxIpsetManager /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/aliases" element={isLoggedIn ? <ProxmoxAliasManager /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/sdn/zones" element={isLoggedIn ? <ProxmoxSdnZones /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/users" element={isLoggedIn ? <ProxmoxUsers /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/users/:userid/tokens" element={isLoggedIn ? <ProxmoxUserTokens /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/pools" element={isLoggedIn ? <ProxmoxPools /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/tasks" element={isLoggedIn ? <ProxmoxTasksPanel /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox/backup" element={isLoggedIn ? <ProxmoxBackupJobs /> : <Navigate to="/login" replace />} />
      <Route path="/truenas" element={isLoggedIn ? <TrueNASWorkspace /> : <Navigate to="/login" replace />} />
      <Route path="/apm" element={isLoggedIn ? <ApmPage /> : <Navigate to="/login" replace />} />
      <Route path="/apm/service" element={isLoggedIn ? <ApmServicePage /> : <Navigate to="/login" replace />} />
      <Route path="/logs" element={isLoggedIn ? <LogsFullPage /> : <Navigate to="/login" replace />} />
      <Route path="/rum" element={isLoggedIn ? <RumFullPage /> : <Navigate to="/login" replace />} />
      <Route path="/rum/session" element={isLoggedIn ? <RumSessionPage /> : <Navigate to="/login" replace />} />
      <Route path="/synthetics" element={isLoggedIn ? <SyntheticsPage /> : <Navigate to="/login" replace />} />
      <Route path="/security" element={isLoggedIn ? <SecurityPage /> : <Navigate to="/login" replace />} />
      <Route path="/cspm" element={isLoggedIn ? <CspmPage /> : <Navigate to="/login" replace />} />
      <Route path="/cicd" element={isLoggedIn ? <CicdPage /> : <Navigate to="/login" replace />} />
      <Route path="/database" element={isLoggedIn ? <DatabasePage /> : <Navigate to="/login" replace />} />
      <Route path="/incidents" element={isLoggedIn ? <IncidentsPage /> : <Navigate to="/login" replace />} />
      <Route path="/notebooks" element={isLoggedIn ? <NotebookPage /> : <Navigate to="/login" replace />} />
      <Route path="/shared" element={isLoggedIn ? <SharedDashboardsPage /> : <Navigate to="/login" replace />} />
      <Route path="/tier0" element={isLoggedIn ? <Dashboard /> : <Navigate to="/login" replace />} />
      <Route path="/intelligence" element={isLoggedIn ? <IntelligencePage /> : <Navigate to="/login" replace />} />
      <Route path="/enterprise" element={isLoggedIn ? <EnterprisePage /> : <Navigate to="/login" replace />} />
      <Route path="/homelab" element={isLoggedIn ? <HomelabPage /> : <Navigate to="/login" replace />} />
      <Route path="/platform" element={isLoggedIn ? <PlatformPage /> : <Navigate to="/login" replace />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
