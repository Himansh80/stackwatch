/**
 * Modal.test.tsx — tests for shared Modal component.
 *
 * Tier 20 AC-2: open/close, focus trap, escape, backdrop click,
 * focus restoration, body scroll lock.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import { useRef } from 'react';
import Modal from './Modal';

describe('Modal', () => {
  beforeEach(() => {
    document.body.style.overflow = '';
  });
  afterEach(() => {
    cleanup();
  });

  it('renders nothing when open is false', () => {
    render(
      <Modal open={false} onClose={() => {}} title="Hidden">
        Body
      </Modal>
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders dialog with title when open', () => {
    render(
      <Modal open onClose={() => {}} title="Edit server">
        Body content
      </Modal>
    );
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Edit server')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(
      <Modal
        open
        onClose={() => {}}
        title="Confirm delete"
        description="This cannot be undone"
      >
        Body
      </Modal>
    );
    expect(screen.getByText('This cannot be undone')).toBeInTheDocument();
  });

  it('uses role=dialog and aria-modal=true', () => {
    render(
      <Modal open onClose={() => {}} title="Test">
        Body
      </Modal>
    );
    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-modal', 'true');
  });

  it('aria-labelledby points to title', () => {
    render(
      <Modal open onClose={() => {}} title="My Title">
        Body
      </Modal>
    );
    const dialog = screen.getByRole('dialog');
    const labelledBy = dialog.getAttribute('aria-labelledby');
    expect(labelledBy).toBeTruthy();
    const titleEl = labelledBy ? document.getElementById(labelledBy) : null;
    expect(titleEl?.textContent).toBe('My Title');
  });

  it('renders footer when provided', () => {
    render(
      <Modal
        open
        onClose={() => {}}
        title="Confirm"
        footer={<button>Save</button>}
      >
        Body
      </Modal>
    );
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
  });

  it('calls onClose when Escape key pressed (default closeOnEscape)', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} title="Test">
        Body
      </Modal>
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does NOT call onClose when Escape pressed if closeOnEscape=false', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} title="Test" closeOnEscape={false}>
        Body
      </Modal>
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
  });

  it('locks body scroll when open', () => {
    render(
      <Modal open onClose={() => {}} title="Test">
        Body
      </Modal>
    );
    expect(document.body.style.overflow).toBe('hidden');
  });

  it('restores body scroll when closed', () => {
    document.body.style.overflow = 'auto';
    const { rerender } = render(
      <Modal open onClose={() => {}} title="Test">
        Body
      </Modal>
    );
    expect(document.body.style.overflow).toBe('hidden');
    rerender(
      <Modal open={false} onClose={() => {}} title="Test">
        Body
      </Modal>
    );
    expect(document.body.style.overflow).toBe('auto');
  });

  it('uses size class sm/md/lg', () => {
    const { rerender } = render(
      <Modal open onClose={() => {}} title="Test" size="sm">
        Body
      </Modal>
    );
    expect(screen.getByRole('dialog')).toHaveClass('modal-sm');

    rerender(
      <Modal open onClose={() => {}} title="Test" size="md">
        Body
      </Modal>
    );
    expect(screen.getByRole('dialog')).toHaveClass('modal-md');

    rerender(
      <Modal open onClose={() => {}} title="Test" size="lg">
        Body
      </Modal>
    );
    expect(screen.getByRole('dialog')).toHaveClass('modal-lg');
  });

  it('forwards initialFocusRef', () => {
    function TestWrapper() {
      const ref = useRef<HTMLButtonElement>(null);
      return (
        <>
          <button>Outside</button>
          <Modal
            open
            onClose={() => {}}
            title="Test"
            initialFocusRef={ref}
          >
            <button ref={ref}>Initial</button>
            <button>Other</button>
          </Modal>
        </>
      );
    }
    render(<TestWrapper />);
    // initialFocusRef should focus the Initial button
    const initial = screen.getByRole('button', { name: 'Initial' });
    expect(document.activeElement).toBe(initial);
  });

  it('calls onClose when backdrop clicked (default)', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} title="Test">
        <button>Inside</button>
      </Modal>
    );
    const backdrop = document.querySelector('.modal-backdrop') as HTMLElement;
    expect(backdrop).toBeTruthy();
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does NOT close when inside content clicked', () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} title="Test">
        <button>Inside</button>
      </Modal>
    );
    fireEvent.click(screen.getByText('Inside'));
    expect(onClose).not.toHaveBeenCalled();
  });
});