import { useEffect, useState } from 'react';

// Persist the last known docked state across mount/unmount of buttons.
let wasDockedGlobal = false;

export function useScrollVisibility(threshold: number): boolean {
  const [visible, setVisible] = useState(() => !wasDockedGlobal);

  useEffect(() => {
    const setByScrollTop = (scrollTop: number) => {
      const nextVisible = scrollTop < threshold;
      wasDockedGlobal = !nextVisible;
      setVisible(nextVisible);
    };

    const onSubtabChange = (event: Event) => {
      const custom = event as CustomEvent<{ scrollTop: number }>;
      if (typeof custom.detail?.scrollTop === 'number') {
        setByScrollTop(custom.detail.scrollTop);
      }
    };

    const onScroll = (event?: Event) => {
      const target = event?.target as HTMLElement | null;
      if (target?.classList?.contains('subcontent-panel')) {
        setByScrollTop(target.scrollTop);
        return;
      }
      if (target?.classList?.contains('content')) {
        setByScrollTop(target.scrollTop);
      }
    };

    const container = document.querySelector<HTMLElement>('.content');
    if (container) {
      setByScrollTop(container.scrollTop);
    }

    document.addEventListener('scroll', onScroll, { passive: true, capture: true });
    window.addEventListener('subtab-scroll-sync', onSubtabChange as EventListener);

    return () => {
      document.removeEventListener('scroll', onScroll, true);
      window.removeEventListener('subtab-scroll-sync', onSubtabChange as EventListener);
    };
  }, [threshold]);

  return visible;
}
