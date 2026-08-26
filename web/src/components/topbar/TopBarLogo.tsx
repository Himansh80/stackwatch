/**
 * TopBarLogo.tsx — left-most topbar element.
 * Shows the brand mark (S gradient square) + word "StackWatch".
 * Hidden on mobile (the sidebar drawer is the mobile nav entry point).
 */
import { Link } from 'react-router-dom';

export default function TopBarLogo() {
  return (
    <Link to="/dashboard" className="tb-logo" aria-label="Go to Overview">
      <span className="tb-logo-mark" aria-hidden="true">S</span>
      <span className="tb-logo-text">StackWatch</span>
    </Link>
  );
}