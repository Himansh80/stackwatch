/**
 * TopBarProfile.tsx — fetches user data, then renders existing ProfileMenu.
 * Just a wrapper that wires the data plumbing into ProfileMenu.
 */
import { useEffect, useState } from 'react';
import ProfileMenu from '../ProfileMenu';
import { me } from '../../lib/api';

interface MeResponse {
  user?: {
    full_name?: string;
    email?: string;
    avatar_url?: string;
  };
  tenant?: { name?: string };
}

export default function TopBarProfile() {
  const [meData, setMeData] = useState<MeResponse | null>(null);

  useEffect(() => {
    let cancelled = false;
    me()
      .then((data: MeResponse) => {
        if (!cancelled) setMeData(data);
      })
      .catch(() => { /* silent */ });
    return () => { cancelled = true; };
  }, []);

  // me() returns { user, tenant } — read from the nested fields
  const fullName = meData?.user?.full_name || meData?.user?.email || 'Operator';
  const firstName = fullName.split(' ')[0] || 'Operator';
  const initials = (fullName.charAt(0) || 'O').toUpperCase();
  const tenantName = meData?.tenant?.name || '—';

  return (
    <ProfileMenu
      firstName={firstName}
      fullName={fullName}
      tenantName={tenantName}
      initials={initials}
      avatarUrl={meData?.user?.avatar_url}
    />
  );
}