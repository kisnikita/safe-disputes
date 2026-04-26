import * as React from 'react';
import './Alert.css';

const ALERT_EXIT_MS = 260;

export type AlertStatus = 'default' | 'info' | 'success' | 'error' | 'warning' | 'loading';
export type AlertPlacement = 'top' | 'bottom' | 'center';
export type AlertActionVariant = 'default' | 'destructive' | 'positive';

type AlertProps = Omit<React.HTMLAttributes<HTMLDivElement>, 'title'> & {
  variant?: 'default' | 'destructive';
  status?: AlertStatus;
  placement?: AlertPlacement;
  floating?: boolean;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  durationMs?: number | null;
  title?: React.ReactNode;
  description?: React.ReactNode;
  actionLabel?: string;
  actionAriaLabel?: string;
  actionVariant?: AlertActionVariant;
  onAction?: () => void;
  closeOnAction?: boolean;
};

const renderStatusIcon = (status: AlertStatus): React.ReactNode => {
  if (status === 'loading') {
    return <span className={`ui-alert-icon ui-alert-icon-${status}`} aria-hidden="true" />;
  }

  const iconMap: Record<Exclude<AlertStatus, 'loading'>, React.ReactNode> = {
    default: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" strokeWidth="2" />
        <circle cx="12" cy="8" r="1.2" fill="currentColor" />
        <path d="M12 11v5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      </svg>
    ),
    info: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" strokeWidth="2" />
        <circle cx="12" cy="8" r="1.2" fill="currentColor" />
        <path d="M12 11v5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      </svg>
    ),
    success: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" strokeWidth="2" />
        <path d="M8.5 12.5l2.4 2.4 4.8-5.1" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
    error: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" strokeWidth="2" />
        <path d="M12 7.6v6.2" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
        <circle cx="12" cy="16.9" r="1.1" fill="currentColor" />
      </svg>
    ),
    warning: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 4l8 14H4L12 4z" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" />
        <path d="M12 9v4" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
        <circle cx="12" cy="15.8" r="1" fill="currentColor" />
      </svg>
    ),
  };

  return <span className={`ui-alert-icon ui-alert-icon-${status}`} aria-hidden="true">{iconMap[status]}</span>;
};

export const Alert = React.forwardRef<HTMLDivElement, AlertProps>(
  (
    {
      className,
      variant = 'default',
      status,
      placement = 'bottom',
      floating = false,
      open,
      defaultOpen = true,
      onOpenChange,
      durationMs = 4000,
      title,
      description,
      actionLabel,
      actionAriaLabel,
      actionVariant = 'default',
      onAction,
      closeOnAction = true,
      children,
      ...props
    },
    ref,
  ) => {
    const isControlled = typeof open === 'boolean';
    const [internalOpen, setInternalOpen] = React.useState(defaultOpen);
    const alertOpen = isControlled ? Boolean(open) : internalOpen;
    const [mounted, setMounted] = React.useState(alertOpen || !floating);
    const [visible, setVisible] = React.useState(alertOpen || !floating);

    const effectiveStatus: AlertStatus = status ?? (variant === 'destructive' ? 'error' : 'default');

    const setAlertOpen = React.useCallback((next: boolean) => {
      if (!isControlled) setInternalOpen(next);
      onOpenChange?.(next);
    }, [isControlled, onOpenChange]);

    React.useEffect(() => {
      if (!floating) return;

      if (alertOpen) {
        setMounted(true);
        const raf = window.requestAnimationFrame(() => setVisible(true));
        return () => window.cancelAnimationFrame(raf);
      }

      setVisible(false);
      const timer = window.setTimeout(() => setMounted(false), ALERT_EXIT_MS);
      return () => window.clearTimeout(timer);
    }, [alertOpen, floating]);

    React.useEffect(() => {
      if (!floating) return;
      if (!alertOpen || durationMs === null || durationMs <= 0) return;

      const timer = window.setTimeout(() => setAlertOpen(false), durationMs);
      return () => window.clearTimeout(timer);
    }, [alertOpen, durationMs, floating, setAlertOpen]);

    if (floating && !mounted) return null;

    const handleActionClick = () => {
      onAction?.();
      if (closeOnAction) setAlertOpen(false);
    };

    const content = (
      <div
        ref={ref}
        role="alert"
        className={`ui-alert ui-alert-status-${effectiveStatus}${floating ? ' ui-alert-floating' : ''}${visible ? ' ui-alert-visible' : ' ui-alert-hidden'}${className ? ` ${className}` : ''}`}
        {...props}
      >
        {title || description || actionLabel ? (
          <div className="ui-alert-content">
            {renderStatusIcon(effectiveStatus)}
            <div className="ui-alert-main">
              {title ? <h5 className="ui-alert-title">{title}</h5> : null}
              {description ? <div className="ui-alert-description">{description}</div> : null}
            </div>
            {actionLabel ? (
              <button
                type="button"
                className={`ui-alert-action ui-alert-action-${actionVariant}`}
                aria-label={actionAriaLabel ?? actionLabel}
                onClick={handleActionClick}
              >
                {actionLabel}
              </button>
            ) : null}
          </div>
        ) : children}
      </div>
    );

    if (!floating) return content;

    return (
      <div className={`ui-alert-portal ui-alert-placement-${placement}`}>
        {content}
      </div>
    );
  },
);
Alert.displayName = 'Alert';

export const AlertTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h5 ref={ref} className={`ui-alert-title${className ? ` ${className}` : ''}`} {...props} />
  ),
);
AlertTitle.displayName = 'AlertTitle';

export const AlertDescription = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={`ui-alert-description${className ? ` ${className}` : ''}`} {...props} />
  ),
);
AlertDescription.displayName = 'AlertDescription';
