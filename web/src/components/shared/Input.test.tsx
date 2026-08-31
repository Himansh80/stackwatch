/**
 * Input.test.tsx + Select.test.tsx + Textarea.test.tsx (combined).
 *
 * Tier 20 AC-4: render, error state, disabled, required, focus.
 */
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import Input from './Input';
import Select from './Select';
import Textarea from './Textarea';

describe('Input', () => {
  it('renders with label', () => {
    render(<Input label="Email" />);
    expect(screen.getByText('Email')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(<Input label="Email" description="We will not spam you" />);
    expect(screen.getByText('We will not spam you')).toBeInTheDocument();
  });

  it('shows error in red and marks aria-invalid', () => {
    render(<Input label="Email" error="Invalid email address" />);
    const input = screen.getByLabelText('Email');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByText('Invalid email address')).toBeInTheDocument();
    expect(screen.getByText('Invalid email address')).toHaveClass('input-error-text');
  });

  it('shows required asterisk when required', () => {
    render(<Input label="Email" required />);
    const asterisk = screen.getByText('*');
    expect(asterisk).toHaveClass('input-required');
    expect(asterisk).toHaveAttribute('aria-hidden', 'true');
  });

  it('sets aria-required when required', () => {
    render(<Input label="Email" required />);
    expect(screen.getByLabelText('Email')).toHaveAttribute('aria-required', 'true');
  });

  it('sets aria-disabled when disabled', () => {
    render(<Input label="Email" disabled />);
    const input = screen.getByLabelText('Email');
    expect(input).toBeDisabled();
    expect(input).toHaveAttribute('aria-disabled', 'true');
  });

  it('fires onChange when typed', () => {
    let value = '';
    render(<Input label="Name" onChange={(e) => (value = e.target.value)} />);
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Himan' } });
    expect(value).toBe('Himan');
  });

  it('applies size class', () => {
    render(<Input label="A" size="sm" />);
    expect(screen.getByLabelText('A')).toHaveClass('input-sm');
  });

  it('applies fullWidth class', () => {
    render(<Input label="A" fullWidth />);
    expect(screen.getByLabelText('A')).toHaveClass('input-full');
  });

  it('label htmlFor matches input id', () => {
    render(<Input label="A" id="my-input" />);
    const input = screen.getByLabelText('A');
    expect(input.id).toBe('my-input');
  });
});

describe('Select', () => {
  const options = [
    { value: 'a', label: 'Alpha' },
    { value: 'b', label: 'Beta' },
    { value: 'c', label: 'Gamma' },
  ];

  it('renders with label and options', () => {
    render(<Select label="Tier" options={options} />);
    expect(screen.getByText('Tier')).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Alpha' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Beta' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Gamma' })).toBeInTheDocument();
  });

  it('renders placeholder option', () => {
    render(<Select label="Tier" options={options} placeholder="Choose..." />);
    const placeholder = screen.getByRole('option', { name: 'Choose...' });
    expect(placeholder).toBeDisabled();
  });

  it('error state shows aria-invalid', () => {
    render(<Select label="Tier" options={options} error="Pick one" />);
    expect(screen.getByLabelText('Tier')).toHaveAttribute('aria-invalid', 'true');
  });
});

describe('Textarea', () => {
  it('renders with rows', () => {
    render(<Textarea label="Bio" rows={6} />);
    const ta = screen.getByLabelText('Bio');
    expect(ta.tagName).toBe('TEXTAREA');
    expect(ta).toHaveAttribute('rows', '6');
  });

  it('error state shows aria-invalid', () => {
    render(<Textarea label="Bio" error="Required" />);
    expect(screen.getByLabelText('Bio')).toHaveAttribute('aria-invalid', 'true');
  });

  it('required shows asterisk', () => {
    render(<Textarea label="Bio" required />);
    expect(screen.getByText('*')).toHaveClass('input-required');
  });

  it('applies fullWidth class', () => {
    render(<Textarea label="Bio" fullWidth />);
    expect(screen.getByLabelText('Bio')).toHaveClass('input-full');
  });
});