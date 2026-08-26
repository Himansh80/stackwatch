// React entry point. Mounts <App /> into #root.
import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import './styles.css';
import './styles-tier2.css';
import './styles/proxmox.css';
import './styles/proxmox-detail.css';
import './styles/proxmox-detail-tables.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
);
