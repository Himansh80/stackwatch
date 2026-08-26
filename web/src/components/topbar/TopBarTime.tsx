/**
 * TopBarTime.tsx — wraps the existing TimeWidget.
 * Just re-exports TimeWidget so it lives in the topbar/ folder alongside
 * the rest of the topbar pieces. No new logic.
 */
import TimeWidget from '../TimeWidget';

export default function TopBarTime() {
  return <TimeWidget />;
}