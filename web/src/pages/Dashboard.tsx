import { useEffect, useState } from 'react';
import { clearToken, me, health } from '../lib/api';

export default function Dashboard() {
  const [meData, setMeData] = useState<any>(null);
  const [healthData, setHealthData] = useState<any>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    me().then(setMeData).catch((e) => setErr(String(e)));
    health().then(setHealthData).catch(() => {});
  }, []);

  const logout = () => {
    clearToken();
    window.location.href = '/login';
  };

  return (
    <div className="dashboard">
      <header>
        <h1>StackWatch — Tier 0</h1>
        <button onClick={logout}>Sign out</button>
      </header>
      <section className="cards">
        <div className="card">
          <h3>User</h3>
          <pre>{meData ? JSON.stringify(meData.user, null, 2) : '...'}</pre>
        </div>
        <div className="card">
          <h3>Tenant</h3>
          <pre>{meData ? JSON.stringify(meData.tenant, null, 2) : '...'}</pre>
        </div>
        <div className="card">
          <h3>Backend health</h3>
          <pre>{healthData ? JSON.stringify(healthData, null, 2) : '...'}</pre>
        </div>
      </section>
      {err && <div className="error">{err}</div>}
    </div>
  );
}
