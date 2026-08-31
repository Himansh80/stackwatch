/**
 * Textarea.tsx — shared multi-line text input component.
 *
 * Tier 20: addresses ~20 medium form_input drifts (textarea elements).
 *
 * Visual contract: same as Input (per spec §3.4).
 */
import { forwardRef, TextareaHTMLAttributes, useId } from 'react';
import { BaseInputWrapper, type BaseInputProps } from './Input';

export interface TextareaProps
  extends TextareaHTMLAttributes<HTMLTextAreaElement>,
    BaseInputProps {
  rows?: number;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  function Textarea(
    {
      label,
      description,
      error,
      required,
      disabled,
      size = 'md',
      fullWidth,
      className = '',
      rows = 4,
      id: providedId,
      ...textareaProps
    },
    ref,
  ) {
    const reactId = useId();
    const id = providedId ?? `textarea-${reactId}`;
    const descId = `${id}-desc`;
    const classes = [
      'input',
      `input-${size}`,
      'textarea',
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
        <textarea
          ref={ref}
          id={id}
          className={classes}
          rows={rows}
          disabled={disabled}
          required={required}
          aria-required={required || undefined}
          aria-invalid={error ? 'true' : undefined}
          aria-describedby={description || error ? descId : undefined}
          aria-disabled={disabled || undefined}
          {...textareaProps}
        />
      </BaseInputWrapper>
    );
  },
);

export default Textarea;