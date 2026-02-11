import React, { useCallback, useEffect, useRef } from 'react';
import './ScrollTopHitArea.css';

interface ScrollTopHitAreaProps {
  enabled: boolean;
  onHit?: () => void;
  className?: string;
}

export function ScrollTopHitArea({
  enabled,
  onHit = () => {},
  className = '',
}: ScrollTopHitAreaProps) {
  return (
    <button
      type="button"
      className={`scroll-top-hit-area${enabled ? ' is-enabled' : ''}${className ? ` ${className}` : ''}`}
      onClick={onHit}
    />
  );
}

export function useDefaultScrollTopHit(
  subcontentRef: React.RefObject<HTMLDivElement | null>,
  currentIndex: number,
  onReset: () => void
) {
  const animationFrameRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (animationFrameRef.current !== null) {
        window.cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, []);

  return useCallback(() => {
    const node = subcontentRef.current;
    if (!node) return;
    const panel = node.querySelectorAll<HTMLElement>('.subcontent-panel')[currentIndex];
    if (!panel) return;

    const startTop = panel.scrollTop;
    if (animationFrameRef.current !== null) {
      window.cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }

    if (startTop <= 1) {
      panel.scrollTop = 0;
      onReset();
      window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: 0 } }));
      return;
    }

    const viewportHeight = Math.max(1, panel.clientHeight);
    const scrolledScreens = startTop / viewportHeight;
    const durationMs = Math.max(160, Math.min(700, 160 + scrolledScreens * 140));
    const startedAt = performance.now();
    const animate = (now: number) => {
      const progress = Math.min(1, (now - startedAt) / durationMs);
      const eased = 1 - Math.pow(1 - progress, 3);
      panel.scrollTop = startTop * (1 - eased);
      if (progress < 1) {
        animationFrameRef.current = window.requestAnimationFrame(animate);
        return;
      }
      panel.scrollTop = 0;
      animationFrameRef.current = null;
    };

    animationFrameRef.current = window.requestAnimationFrame(animate);
    onReset();
    window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: 0 } }));
  }, [subcontentRef, currentIndex, onReset]);
}
