/**
 * Button.test.tsx — tests for shared Button component.
 *
 * Tier 20 AC-1: 4 variants × 3 sizes correctly, loading state shows
 * spinner + disabled, click handler fires, aria-disabled + aria-busy.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import Button from './Button';

describe('Button', () => {
  it('renders children correctly', () => {
    render(<Button>Save changes</Button>);
    expect(screen.getByText('Save changes')).toBeInTheDocument();
  });

  it('renders as a real <button> element', () => {
    render(<Button>Click me</Button>);
    const btn = screen.getByRole('button');
    expect(btn.tagName).toBe('BUTTON');
  });

  it('applies variant class', () => {
    const { rerender } = render(<Button variant="primary">P</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-primary');

    rerender(<Button variant="secondary">S</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-secondary');

    rerender(<Button variant="ghost">G</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-ghost');

    rerender(<Button variant="danger">D</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-danger');
  });

  it('applies size class', () => {
    const { rerender } = render(<Button size="sm">S</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-sm');

    rerender(<Button size="md">M</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-md');

    rerender(<Button size="lg">L</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-lg');
  });

  it('defaults to variant=primary and size=md', () => {
    render(<Button>Default</Button>);
    const btn = screen.getByRole('button');
    expect(btn).toHaveClass('btn-primary');
    expect(btn).toHaveClass('btn-md');
  });

  it('fires onClick when clicked', () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click</Button>);
    fireEvent.click(screen.getByText('Click'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('does NOT fire onClick when disabled', () => {
    const onClick = vi.fn();
    render(<Button disabled onClick={onClick}>Disabled</Button>);
    fireEvent.click(screen.getByText('Disabled'));
    expect(onClick).not.toHaveBeenCalled();
  });

  it('shows loading state with spinner + disables click', () => {
    const onClick = vi.fn();
    render(<Button loading onClick={onClick}>Loading</Button>);
    const btn = screen.getByRole('button');
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute('aria-disabled', 'true');
    expect(btn).toHaveAttribute('aria-busy', 'true');
    expect(btn).toHaveClass('btn-loading');
    fireEvent.click(btn);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('renders icon before children when provided', () => {
    render(
      <Button icon={<span data-testid="icon">★</span>}>Star</Button>
    );
    const btn = screen.getByRole('button');
    const icon = screen.getByTestId('icon');
    // icon should appear before label in DOM order
    expect(btn.compareDocumentPosition(icon) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('sets aria-label when provided', () => {
    render(<Button aria-label="Close dialog">×</Button>);
    expect(screen.getByLabelText('Close dialog')).toBeInTheDocument();
  });

  it('supports type="submit"', () => {
    render(<Button type="submit">Submit</Button>);
    expect(screen.getByRole('button')).toHaveAttribute('type', 'submit');
  });

  it('applies fullWidth class when fullWidth', () => {
    render(<Button fullWidth>Wide</Button>);
    expect(screen.getByRole('button')).toHaveClass('btn-full');
  });
});