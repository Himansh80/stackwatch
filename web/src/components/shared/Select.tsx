/**
 * Select.tsx — shared Select (dropdown) component.
 *
 * Tier 20: addresses ~50 medium form_input drifts (specifically <select> elements).
 *
 * Visual contract: same as Input (per spec §3.4).
 * Wraps native <select> with BaseInputWrapper from Input.tsx.
 */
import { forwardRef, SelectHTMLAttributes, useId } from 'react';
import { BaseInputWrapper, type BaseInputProps } from './Input';

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface SelectProps
  extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'size'>,
    BaseInputProps {
  options: SelectOption[];
  placeholder?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  {
    label,
    description,
    error,
    required,
    disabled,
    size = 'md',
    fullWidth,
    className = '',
    options,
    placeholder,
    id: providedId,
    ...selectProps
  },
  ref,
) {
  const reactId = useId();
  const id = providedId ?? `select-${reactId}`;
  const descId = `${id}-desc`;
  const selectClasses = [
    'input',
    `input-${size}`,
    'select',
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
      <select
        ref={ref}
        id={id}
        className={selectClasses}
        disabled={disabled}
        required={required}
        aria-required={required || undefined}
        aria-invalid={error ? 'true' : undefined}
        aria-describedby={description || error ? descId : undefined}
        aria-disabled={disabled || undefined}
        {...selectProps}
      >
        {placeholder && (
          <option value="" disabled>
            {placeholder}
          </option>
        )}
        {options.map((opt) => (
          <option key={opt.value} value={opt.value} disabled={opt.disabled}>
            {opt.label}
          </option>
        ))}
      </select>
    </BaseInputWrapper>
  );
});

export default Select;