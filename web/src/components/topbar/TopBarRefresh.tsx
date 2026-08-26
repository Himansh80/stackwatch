/**
 * TopBarRefresh.tsx — refresh button with spin animation.
 * Click triggers onRefresh callback. Spins while in flight.
 */
import { useState } from 'react';
import { motion } from 'framer-motion';
import { RefreshIcon } from '../icons';

interface TopBarRefreshProps {
  onRefresh: () => Promise<void> | void;
}

export default function TopBarRefresh({ onRefresh }: TopBarRefreshProps) {
  const [spinning, setSpinning] = useState(false);

  async function handleClick() {
    setSpinning(true);
    try {
      await onRefresh();
    } finally {
      // Spin for at least 500ms so it feels responsive even if data loads fast
      setTimeout(() => setSpinning(false), 500);
    }
  }

  return (
    <button
      type="button"
      className="tb-refresh"
      onClick={handleClick}
      disabled={spinning}
      aria-label="Refresh page"
      title="Refresh"
    >
      <motion.span
        animate={{ rotate: spinning ? 360 : 0 }}
        transition={{ duration: 0.5, ease: 'linear' }}
        style={{ display: 'inline-flex' }}
      >
        <RefreshIcon size={14} />
      </motion.span>
    </button>
  );
}