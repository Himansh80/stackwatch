/**
 * Tier 14 Phase 14.3 — Real noVNC console for VM detail.
 *
 * - Fetches VNC ticket from backend
 * - Initializes noVNC client (RFB) over WSS to our /vnc-ws proxy
 * - Renders into a <canvas> ref
 * - Toolbar: Reconnect, Send Ctrl+Alt+Del, Fullscreen
 * - Disconnects cleanly on unmount
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';

interface Props {
  hostId: string;
  node: string;
  vmid: number;
}

type ConnState = 'idle' | 'connecting' | 'connected' | 'disconnected' | 'error';

export default function ProxmoxDetailConsole({ hostId, node, vmid }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const rfbRef = useRef<unknown>(null); // RFB instance from noVNC
  const containerRef = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<ConnState>('idle');
  const [error, setError] = useState('');

  const connect = useCallback(async () => {
    if (!canvasRef.current) return;
    setState('connecting');
    setError('');
    try {
      // 1. Fetch ticket + host info
      const ticketResp = await api<{ ticket: string; node: string; port: number }>(
        'POST',
        `/api/v1/proxmox/hosts/${hostId}/vnc-ticket`,
        { node, vmid },
      );
      // 2. Build WSS URL (proxied through our backend)
      const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
      const wsUrl = `${proto}://${window.location.host}/api/v1/proxmox/hosts/${hostId}/vnc-ws?node=${encodeURIComponent(node)}&vmid=${vmid}`;

      // 3. Dynamic-import noVNC to keep initial bundle small
      const RFBMod = await import('novnc-next');
      const RFB = (RFBMod as { default: new (...a: unknown[]) => unknown }).default;

      // 4. Construct RFB client
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const rfb = new (RFB as any)(canvasRef.current, wsUrl, {
        credentials: { password: ticketResp.ticket },
        wsProtocols: ['binary'],
        repeaterID: `stackwatch-${hostId}-${vmid}`,
      });
      rfbRef.current = rfb;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (rfb as any).addEventListener('connect', () => setState('connected'));
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (rfb as any).addEventListener('disconnect', () => setState('disconnected'));
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (rfb as any).addEventListener('securityfailure', (e: { detail: { reason: string } }) => {
        setState('error');
        setError(`Security failure: ${e.detail.reason}`);
      });
    } catch (cause) {
      setState('error');
      setError(cause instanceof Error ? cause.message : 'Unable to start VNC session.');
    }
  }, [hostId, node, vmid]);

  // Auto-connect on mount
  useEffect(() => {
    void connect();
    return () => {
      const rfb = rfbRef.current as { disconnect?: () => void } | null;
      try {
        rfb?.disconnect?.();
      } catch {
        /* ignore */
      }
      rfbRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const sendCtrlAltDel = useCallback(() => {
    const rfb = rfbRef.current as { sendCtrlAltDel?: () => void } | null;
    try {
      rfb?.sendCtrlAltDel?.();
    } catch {
      /* ignore */
    }
  }, []);

  const fullscreen = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    if (document.fullscreenElement) {
      void document.exitFullscreen();
    } else {
      void el.requestFullscreen?.();
    }
  }, []);

  const stateLabel: Record<ConnState, { label: string; color: string }> = {
    idle: { label: 'idle', color: 'slate' },
    connecting: { label: 'connecting…', color: 'amber' },
    connected: { label: 'connected', color: 'green' },
    disconnected: { label: 'disconnected', color: 'slate' },
    error: { label: 'error', color: 'red' },
  };

  return (
    <motion.div
      ref={containerRef}
      className="px-console"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
    >
      <div className="px-console-toolbar">
        <span className={`px-status px-status-${stateLabel[state].color}`}>
          ● {stateLabel[state].label}
        </span>
        <div className="px-console-actions">
          <button type="button" className="px-btn px-btn-quiet" onClick={connect}>
            ⟳ Reconnect
          </button>
          <button type="button" className="px-btn px-btn-quiet" onClick={sendCtrlAltDel} disabled={state !== 'connected'}>
            ⌃⌥⌫ Send Ctrl+Alt+Del
          </button>
          <button type="button" className="px-btn px-btn-quiet" onClick={fullscreen}>
            ⛶ Fullscreen
          </button>
        </div>
      </div>

      {error && <div className="px-error">⚠ {error}</div>}

      <div className="px-console-canvas-wrap">
        <canvas
          ref={canvasRef}
          className="px-console-canvas"
          width={1024}
          height={768}
        />
      </div>

      {state !== 'connected' && state !== 'connecting' && (
        <div className="px-console-overlay">
          <p>{state === 'error' ? error || 'Connection failed' : 'VNC disconnected'}</p>
          <button type="button" className="px-btn" onClick={connect}>Reconnect</button>
        </div>
      )}
    </motion.div>
  );
}