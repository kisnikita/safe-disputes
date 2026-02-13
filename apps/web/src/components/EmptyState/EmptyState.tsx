import Lottie from 'lottie-react';
import emptyDuck from '../../../assets/empty-duck.json';
import notFoundDuck from '../../../assets/not-found-duck.json';
import willAddedDuck from '../../../assets/will-added-duck.json';
import './EmptyState.css';

type EmptyStateVariant = 'empty' | 'notFound' | 'comingSoon';

type EmptyStateProps = {
  message?: string;
  variant: EmptyStateVariant;
  className?: string;
  hint?: string;
  onHintClick?: () => void;
  hintIconDirection?: 'up' | 'right' | 'left' | 'both' | 'none';
};

const animationByVariant: Record<EmptyStateVariant, object> = {
  empty: emptyDuck,
  notFound: notFoundDuck,
  comingSoon: willAddedDuck,
};

export const EmptyState = ({
  message,
  variant,
  className,
  hint,
  onHintClick,
  hintIconDirection = 'up',
}: EmptyStateProps) => {
  const animationData = animationByVariant[variant];
  const rootClassName = ['empty-state', 'has-animation', className].filter(Boolean).join(' ');

  const isRight = hintIconDirection === 'right';
  const isBoth = hintIconDirection === 'both';
  const isNone = hintIconDirection === 'none';
  const leadingIconDirection = hintIconDirection === 'both' ? 'left' : hintIconDirection;

  return (
    <div className={rootClassName}>
      <Lottie
        className="empty-state-animation"
        animationData={animationData}
        loop
        autoplay
        aria-hidden="true"
      />
      {message && <div className="empty-state-message">{message}</div>}
      {hint &&
        (onHintClick ? (
          <button type="button" className="empty-state-hint empty-state-hint-button" onClick={onHintClick}>
            {!isRight && !isNone && (
              <span
                className={`empty-state-hint-icon empty-state-hint-icon-${leadingIconDirection}`}
                aria-hidden="true"
              />
            )}
            <span>{hint}</span>
            {!isNone && (isRight || isBoth) && (
              <span className="empty-state-hint-icon empty-state-hint-icon-right" aria-hidden="true" />
            )}
          </button>
        ) : (
          <div className="empty-state-hint">
            {!isRight && !isNone && (
              <span
                className={`empty-state-hint-icon empty-state-hint-icon-${leadingIconDirection}`}
                aria-hidden="true"
              />
            )}
            <span>{hint}</span>
            {!isNone && (isRight || isBoth) && (
              <span className="empty-state-hint-icon empty-state-hint-icon-right" aria-hidden="true" />
            )}
          </div>
        ))}
    </div>
  );
};
