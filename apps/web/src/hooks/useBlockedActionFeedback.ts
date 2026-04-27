import { useCallback, useState } from 'react';

type FocusTarget = HTMLElement | null | (() => HTMLElement | null);

const getFocusTargetElement = (target: FocusTarget): HTMLElement | null => (
  typeof target === 'function' ? target() : target
);

const focusTarget = (target: FocusTarget): void => {
  const element = getFocusTargetElement(target);
  if (!element) return;

  try {
    element.focus({ preventScroll: true });
  } catch {
    element.focus();
  }
  element.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'smooth' });

  const isInputElement = typeof HTMLInputElement !== 'undefined' && element instanceof HTMLInputElement;
  const isTextareaElement = typeof HTMLTextAreaElement !== 'undefined' && element instanceof HTMLTextAreaElement;
  if (isInputElement || isTextareaElement) {
    element.select();
  }
};

export const useBlockedActionFeedback = () => {
  const [isShaking, setIsShaking] = useState(false);

  const triggerShake = useCallback(() => {
    setIsShaking(false);
    requestAnimationFrame(() => setIsShaking(true));
  }, []);

  const handleShakeAnimationEnd = useCallback(() => {
    setIsShaking(false);
  }, []);

  const triggerBlockedActionFeedback = useCallback((target: FocusTarget) => {
    triggerShake();
    requestAnimationFrame(() => focusTarget(target));
  }, [triggerShake]);

  return {
    isShaking,
    triggerShake,
    handleShakeAnimationEnd,
    triggerBlockedActionFeedback,
  };
};
