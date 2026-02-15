import { useEffect, useState } from 'react';

export function useScrollVisibility(threshold: number): boolean {
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    const onSubtabChange = (event: Event) => {
      const custom = event as CustomEvent<{ scrollTop: number }>;
      if (typeof custom.detail?.scrollTop === 'number') {
        setVisible(custom.detail.scrollTop < threshold);
      }
    };

    const onScroll = (event?: Event) => {
      const target = event?.target as HTMLElement | null;
      if (target?.classList?.contains('subcontent-panel')) {
        setVisible(target.scrollTop < threshold);
        return;
      }
      if (target?.classList?.contains('content')) {
        setVisible(target.scrollTop < threshold);
      }
    };

    const container = document.querySelector<HTMLElement>('.content');
    if (container) {
      setVisible(container.scrollTop < threshold);
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
