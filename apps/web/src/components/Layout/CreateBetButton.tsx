// src/components/Layout/CreateBetButton.tsx
import React, { useState, useEffect } from 'react';
import './CreateBetButton.css';
import { useTonConnect } from '../../hooks/useTonConnect';
import { HIDE_THRESHOLD } from '../../utils/constants';


interface Props {
  onOpenForm: () => void;
  forceHidden?: boolean;
}

export const CreateBetButton: React.FC<Props> = ({ onOpenForm, forceHidden = false }) => {
  const { address, connected } = useTonConnect();
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    // Порог в пикселях, после которого кнопку скрываем
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

  return (
    <button
      className={`create-button${visible && !forceHidden ? '' : ' hidden'}`}
      onClick={() => connected && onOpenForm()}
      disabled={!connected}
      title={connected ? undefined : 'Сначала подключите TON-кошелёк'}
    >
      Создать пари
    </button>
  );
};
