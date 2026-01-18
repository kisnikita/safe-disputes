import React, { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import './RatingButton.css';

interface Props {
  onClick: () => void;
}

export const RatingButton: React.FC<Props> = ({ onClick }) => {
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    const HIDE_THRESHOLD = 30;
    const onSubtabChange = (event: Event) => {
      const custom = event as CustomEvent<{ scrollTop: number }>;
      if (typeof custom.detail?.scrollTop === 'number') {
        setVisible(custom.detail.scrollTop < HIDE_THRESHOLD);
      }
    };
    const onScroll = (event?: Event) => {
      const target = event?.target as HTMLElement | null;
      if (target?.classList?.contains('subcontent-panel')) {
        setVisible(target.scrollTop < HIDE_THRESHOLD);
        return;
      }
      if (target?.classList?.contains('content')) {
        setVisible(target.scrollTop < HIDE_THRESHOLD);
      }
    };

    const container = document.querySelector<HTMLElement>('.content');
    if (container) {
      setVisible(container.scrollTop < HIDE_THRESHOLD);
    }

    document.addEventListener('scroll', onScroll, { passive: true, capture: true });
    window.addEventListener('subtab-scroll-sync', onSubtabChange as EventListener);
    return () => {
      document.removeEventListener('scroll', onScroll, true);
      window.removeEventListener('subtab-scroll-sync', onSubtabChange as EventListener);
    };
  }, []);

  const button = (
    <button className={`rating-button${visible ? '' : ' hidden'}`} onClick={onClick}>
      Рейтинг
    </button>
  );

  return createPortal(button, document.body);
};
