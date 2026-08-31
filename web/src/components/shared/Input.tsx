/**
 * Input.tsx — shared Input component + BaseInputWrapper.
 *
 * Tier 20: addresses 150 medium form_input drifts from audit.
 *
 * Visual contract (from spec §3.4):
 *   - Container: display: flex; flex-direction: column; gap: var(--space-1)
 *   - Label: 14px, 500 weight, var(--color-text), 4px below container top
 *   - Required: * after label, var(--color-error)
 *   - Input: 40px (sm=32px), var(--color-surface) bg, 1px border-strong, focus → 2px border-focus
 *   - Padding: var(--space-2) var(--space-3)
 *   - Font: 14px, var(--color-text)
 *   - Placeholder: var(--color-text-disabled)
 *   - Description: 12px, var(--color-text-muted), 4px above input
 *   - Error: border var(--color-error), description → error text in error color,
 *     aria-invalid="true", aria-describedby points to error id
 *   - Disabled: opacity 0.5, cursor not-allowed, aria-disabled="true"
 *   - Focus ring: 2px var(--color-border-focus) with 2px offset, via :focus-visible
 *
 * Accessibility (from spec §3.4):
 *   - <label htmlFor> properly associated with input id
 *   - Error messages linked via aria-describedby
 *   - Required: aria-required="true" + visible asterisk
 */
import {
  forwardRef,
  ReactNode,
  InputHTMLAttributes,
  TextareaHTMLAttributes,
  SelectHTMLAttributes,
  useId,
} from 'react';

export interface BaseInputProps {
  label?: string;
  description?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  size?: 'sm' | 'md';
  fullWidth?: boolean;
  className?: string;
}

/**
 * BaseInputWrapper — shared label + description + error layout.
 * Used by Input, Select, Textarea. Exported so the others can reuse.
 */
export function BaseInputWrapper({
  label,
  description,
  error,
  required,
  htmlFor,
  describedById,
  children,
}: {
  label?: string;
  description?: string;
  error?: string;
  required?: boolean;
  htmlFor?: string;
  describedById?: string;
  children: ReactNode;
}) {
  return (
    <div className="input-wrapper">
      {label && (
        <label htmlFor={htmlFor} className="input-label">
          {label}
          {required && (
            <span aria-hidden="true" className="input-required">
              *
            </span>
          )}
        </label>
      )}
      {children}
      {(error || description) && (
        <div
          id={describedById}
          className={error ? 'input-error-text' : 'input-description-text'}
        >
          {error || description}
        </div>
      )}
    </div>
  );
}

export interface InputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'>,
    BaseInputProps {
  // Inherits all native <input> attrs via Omit+extends
}

/**
 * Input — text/email/password/etc input with label + description + error.
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  {
    label,
    description,
    error,
    required,
    disabled,
    size = 'md',
    fullWidth,
    className = '',
    id: providedId,
    ...inputProps
  },
  ref,
) {
  const reactId = useId();
  const id = providedId ?? `input-${reactId}`;
  const descId = `${id}-desc`;
  const inputClasses = [
    'input',
    `input-${size}`,
    error ? 'input-error' : '',
    fullWidth ? 'input-full' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <BaseInputWrapper
      label={label}
      description={description}
      error={error}
      required={required}
      htmlFor={id}
      describedById={descId}
    >
      <input
        ref={ref}
        id={id}
        className={inputClasses}
        disabled={disabled}
        required={required}
        aria-required={required || undefined}
        aria-invalid={error ? 'true' : undefined}
        aria-describedby={description || error ? descId : undefined}
        aria-disabled={disabled || undefined}
        {...inputProps}
      />
    </BaseInputWrapper>
  );
});

export default Input;