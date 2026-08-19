import { Navigate, Route, Routes } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import TrueNASWorkspace from './pages/TrueNASWorkspace';
import ProxmoxWorkspace from './components/proxmox/ProxmoxWorkspace';
import { isLoggedIn } from './lib/api';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={isLoggedIn() ? <Dashboard /> : <Navigate to="/login" replace />} />
      <Route path="/login" element={<Login />} />
      <Route path="/dashboard" element={isLoggedIn() ? <Dashboard /> : <Navigate to="/login" replace />} />
      <Route path="/proxmox" element={isLoggedIn() ? <ProxmoxWorkspace /> : <Navigate to="/login" replace />} />
      <Route path="/truenas" element={isLoggedIn() ? <TrueNASWorkspace /> : <Navigate to="/login" replace />} />
      <Route path="/tier0" element={isLoggedIn() ? <Dashboard /> : <Navigate to="/login" replace />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
