/**
 * Button.tsx — shared Button component.
 *
 * Tier 20: replaces all inline buttons across the dashboard. Single source
 * of truth for button styling, sizing, states (default/hover/focus/active/
 * disabled/loading), and accessibility.
 *
 * Visual contract (from spec §3.1):
 *   - 4 variants: primary (cyan), secondary (elevated surface), ghost (transparent),
 *     danger (red)
 *   - 3 sizes: sm=32px, md=40px, lg=48px
 *   - All colors via var(--*) tokens
 *   - framer-motion scale on tap
 *   - Loading state shows Spinner + disables click
 *
 * Accessibility:
 *   - Real <button> element (not <div>)
 *   - aria-disabled when loading
 *   - aria-busy when loading
 *   - Focus ring on :focus-visible
 */
import { forwardRef, MouseEvent, ReactNode, ButtonHTMLAttributes } from 'react';
import { motion, useReducedMotion } from 'framer-motion';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
export type ButtonSize = 'sm' | 'md' | 'lg';

export interface ButtonProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'type' | 'size'> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: ReactNode;
  iconRight?: ReactNode;
  loading?: boolean;
  disabled?: boolean;
  fullWidth?: boolean;
  type?: 'button' | 'submit' | 'reset';
  children: ReactNode;
  onClick?: (e: MouseEvent<HTMLButtonElement>) => void;
  'aria-label'?: string;
  className?: string;
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = 'primary',
    size = 'md',
    icon,
    iconRight,
    loading = false,
    disabled = false,
    fullWidth = false,
    type = 'button',
    children,
    onClick,
    'aria-label': ariaLabel,
    className = '',
  },
  ref,
) {
  const reduce = useReducedMotion();
  const isDisabled = disabled || loading;

  const classes = [
    'btn',
    `btn-${variant}`,
    `btn-${size}`,
    fullWidth ? 'btn-full' : '',
    loading ? 'btn-loading' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <motion.button
      ref={ref}
      type={type}
      className={classes}
      onClick={onClick}
      disabled={isDisabled}
      aria-disabled={isDisabled || undefined}
      aria-busy={loading || undefined}
      aria-label={ariaLabel}
      whileTap={reduce || isDisabled ? undefined : { scale: 0.97 }}
      transition={{ duration: 0.1, ease: [0.4, 0, 0.2, 1] }}
    >
      {loading ? (
        <span className="btn-spinner" aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
            <circle
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeOpacity="0.25"
              strokeWidth="2"
            />
            <path
              d="M22 12 a10 10 0 0 0 -10 -10"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
            />
          </svg>
        </span>
      ) : icon ? (
        <span className="btn-icon" aria-hidden="true">
          {icon}
        </span>
      ) : null}
      <span className="btn-label">{children}</span>
      {iconRight && !loading ? (
        <span className="btn-icon-right" aria-hidden="true">
          {iconRight}
        </span>
      ) : null}
    </motion.button>
  );
});

export default Button;