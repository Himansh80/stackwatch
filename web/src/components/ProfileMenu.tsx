import { useEffect, useRef, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useLogout } from '../lib/useLogout';
import Button from './shared/Button';

type MenuItem = {
  label: string;
  description: string;
  to?: string;
  icon: string;
  danger?: boolean;
  onClick?: () => void;
};

type MenuGroup = {
  heading?: string;
  items: MenuItem[];
};

type MenuProps = {
  firstName: string;
  fullName: string;
  tenantName: string;
  initials: string;
  avatarUrl?: string;
};

export default function ProfileMenu({ firstName, fullName, tenantName, initials, avatarUrl }: MenuProps) {
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement | null>(null);
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const location = useLocation();
  const logout = useLogout();

  function close() {
    setOpen(false);
    buttonRef.current?.focus();
  }

  useEffect(() => {
    if (!open) return;
    function onDoc(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        setOpen(false);
        buttonRef.current?.focus();
      }
    }
    document.addEventListener('mousedown', onDoc);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDoc);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  // Close menu when route changes (covers the case where the user clicks a
  // menu item that navigates — we still want it closed even if the listener
  // hasn't fired yet).
  useEffect(() => {
    setOpen(false);
  }, [location.pathname]);

  const groups: MenuGroup[] = [
    {
      items: [
        { icon: '◉', label: 'Profile', description: 'Your name, email, workspace', to: '/profile' },
        { icon: '$', label: 'Billing', description: 'Plan, invoices, payment method', to: '/billing' },
        { icon: '⚙', label: 'Settings', description: 'Security, password, workspace', to: '/settings' },
      ],
    },
    {
      items: [
        { icon: '↪', label: 'Sign out', description: `Sign out of ${firstName}'s session`, danger: true, onClick: logout },
      ],
    },
  ];

  return (
    <div className="dash-menu-wrap" ref={wrapRef}>
      <Button
        ref={buttonRef}
        type="button"
        variant="ghost"
        size="md"
        className="dash-user-link"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Open profile menu"
        onClick={() => setOpen((o) => !o)}
      >
        <span className="dash-avatar">
          {avatarUrl ? <img src={avatarUrl} alt="" /> : initials}
        </span>
        <span className={`dash-menu-caret ${open ? 'open' : ''}`} aria-hidden>▾</span>
      </Button>
      {open && (
        <div className="dash-menu" role="menu" aria-label="Profile">
          <div className="dash-menu-header">
            <span className="dash-avatar dash-avatar-lg">
              {avatarUrl ? <img src={avatarUrl} alt="" /> : initials}
            </span>
            <div>
              <strong>{fullName || '—'}</strong>
              <small>{tenantName || 'Workspace'}</small>
            </div>
          </div>
          {groups.map((group, gIdx) => (
            <div className="dash-menu-group" key={gIdx}>
              {group.heading && <span className="dash-menu-heading">{group.heading}</span>}
              <ul className="dash-menu-items">
                {group.items.map((item) => {
                  const className = `dash-menu-item ${item.danger ? 'dash-menu-item-danger' : ''}`;
                  if (item.onClick) {
                    return (
                      <li key={item.label}>
                        <Button type="button" variant={item.danger ? 'danger' : 'ghost'} size="md" role="menuitem" className={className} onClick={() => { close(); item.onClick!(); }}>
                          <span className="dash-menu-icon" aria-hidden>{item.icon}</span>
                          <span className="dash-menu-text">
                            <strong>{item.label}</strong>
                            <small>{item.description}</small>
                          </span>
                        </Button>
                      </li>
                    );
                  }
                  return (
                    <li key={item.label}>
                      <Link role="menuitem" to={item.to!} className={className} onClick={close}>
                        <span className="dash-menu-icon" aria-hidden>{item.icon}</span>
                        <span className="dash-menu-text">
                          <strong>{item.label}</strong>
                          <small>{item.description}</small>
                        </span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
