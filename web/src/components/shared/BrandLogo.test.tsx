/**
 * SkeletonCard.test.tsx + BrandLogo.test.tsx (combined).
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import SkeletonCard from './SkeletonCard';
import BrandLogo from './BrandLogo';

describe('SkeletonCard', () => {
  it('renders with default panel variant', () => {
    const { container } = render(<SkeletonCard />);
    expect(container.querySelector('.skeleton-card')).toBeInTheDocument();
    expect(container.querySelector('.skeleton-card-panel')).toBeInTheDocument();
  });

  it('renders kpi variant with eyebrow + value blocks', () => {
    const { container } = render(<SkeletonCard variant="kpi" />);
    expect(container.querySelector('.skeleton-card-eyebrow')).toBeInTheDocument();
    expect(container.querySelector('.skeleton-card-value')).toBeInTheDocument();
  });

  it('renders list variant with 3 blocks', () => {
    const { container } = render(<SkeletonCard variant="list" />);
    const blocks = container.querySelectorAll('.skeleton-card-list-block');
    expect(blocks.length).toBe(3);
  });

  it('has role=status + aria-live=polite for screen readers', () => {
    const { container } = render(<SkeletonCard />);
    const card = container.querySelector('.skeleton-card');
    expect(card).toHaveAttribute('role', 'status');
    expect(card).toHaveAttribute('aria-live', 'polite');
  });

  it('honors width and height props', () => {
    const { container } = render(<SkeletonCard width={400} height={200} />);
    const card = container.querySelector('.skeleton-card') as HTMLElement;
    expect(card.style.width).toBe('400px');
    expect(card.style.height).toBe('200px');
  });
});

describe('BrandLogo', () => {
  it('renders mark variant by default', () => {
    const { container } = render(<BrandLogo />);
    expect(container.querySelector('.brand-logo-mark')).toBeInTheDocument();
  });

  it('renders full variant with wordmark', () => {
    const { container } = render(<BrandLogo variant="full" />);
    expect(container.querySelector('.brand-logo-full')).toBeInTheDocument();
    expect(screen.getByText('StackWatch')).toBeInTheDocument();
    expect(screen.getByText('Infrastructure control plane')).toBeInTheDocument();
  });

  it('uses default size 32', () => {
    const { container } = render(<BrandLogo />);
    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('width', '32');
    expect(svg).toHaveAttribute('height', '32');
  });

  it('honors custom size', () => {
    const { container } = render(<BrandLogo size={48} />);
    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('width', '48');
    expect(svg).toHaveAttribute('height', '48');
  });

  it('uses currentColor for theming', () => {
    const { container } = render(<BrandLogo />);
    const svg = container.querySelector('svg');
    const rect = svg?.querySelector('rect');
    expect(rect).toHaveAttribute('fill', 'currentColor');
  });
});