/**
 * AppShell.tsx — shared layout wrapper for every authenticated page.
 *
 * Structure:
 *   <a className="skip-link">Skip to main content</a>
 *   <Sidebar />                          (desktop, hidden on mobile)
 *   <TopBar />                           (sticky, blurred background)
 *   <MobileDrawer isOpen={...} />        (slides in from left on mobile)
 *   <main>
 *     <AnimatePresence mode="wait">
 *       <motion.div key={location.pathname} ...>
 *         {children}                       ← page renderer
 *       </motion.div>
 *     </AnimatePresence>
 *   </main>
 *
 * State owned here:
 *   - mobileDrawerOpen: boolean (controlled by hamburger in TopBar)
 *   - paletteOpen:      boolean (controlled by TopBarSearch + Cmd+K)
 *
 * Refs / handlers provided to children:
 *   - refresh(): Promise — triggers a page reload via location.reload()
 *     (simple; pages with internal polling can ignore)
 */
import { ReactNode, useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { AnimatePresence, motion } from 'framer-motion';
import Sidebar from './sidebar/Sidebar';
import TopBar from './topbar/TopBar';
import CommandPalette from './CommandPalette';
import { CloseIcon } from './icons';

interface AppShellProps {
  children: ReactNode;
}

export default function AppShell({ children }: AppShellProps) {
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const location = useLocation();

  // Close mobile drawer on Escape
  useEffect(() => {
    if (!mobileDrawerOpen) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setMobileDrawerOpen(false);
    }
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [mobileDrawerOpen]);

  // Cmd+K opens palette
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setPaletteOpen(true);
      } else if (e.key === 'Escape' && paletteOpen) {
        setPaletteOpen(false);
      }
    }
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [paletteOpen]);

  // Refresh = full page reload (simple + reliable)
  const refresh = useCallback(async () => {
    window.location.reload();
  }, []);

  const closeDrawer = useCallback(() => setMobileDrawerOpen(false), []);

  return (
    <>
      <a href="#main" className="skip-link">Skip to main content</a>

      {/* Desktop sidebar (hidden on mobile via CSS) */}
      <Sidebar />

      {/* Mobile drawer (slides in from left) */}
      <AnimatePresence>
        {mobileDrawerOpen && (
          <>
            <motion.div
              className="sb-backdrop"
              onClick={closeDrawer}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.15 }}
            />
            <motion.div
              className="sb-drawer-wrap"
              initial={{ x: '-100%' }}
              animate={{ x: 0 }}
              exit={{ x: '-100%' }}
              transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
            >
              <button
                type="button"
                className="sb-drawer-close"
                onClick={closeDrawer}
                aria-label="Close navigation"
              >
                <CloseIcon size={18} />
              </button>
              <Sidebar inDrawer onItemClick={closeDrawer} />
            </motion.div>
          </>
        )}
      </AnimatePresence>

      <TopBar
        onMobileMenu={() => setMobileDrawerOpen(true)}
        onOpenPalette={() => setPaletteOpen(true)}
        onRefresh={refresh}
      />

      <main id="main" className="app-main">
        <AnimatePresence mode="wait">
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
          >
            {children}
          </motion.div>
        </AnimatePresence>
      </main>

      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
    </>
  );
}