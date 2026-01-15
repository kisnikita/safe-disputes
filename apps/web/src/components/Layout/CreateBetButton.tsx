// src/components/Layout/CreateBetButton.tsx
import React, { useState, useEffect } from 'react';
import './CreateBetButton.css';
import { useTonConnect } from '../../hooks/useTonConnect';

interface Props {
  onOpenForm: () => void;
  forceHidden?: boolean;
}

export const CreateBetButton: React.FC<Props> = ({ onOpenForm, forceHidden = false }) => {
  const { address, connected } = useTonConnect();
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    // Ищем ваш скроллируемый контейнер
    const container = document.querySelector<HTMLElement>('.content');
    if (!container) return;

    // Порог в пикселях, после которого кнопку скрываем
    const HIDE_THRESHOLD = 30;
    const onScroll = () => {
      // visible = true, если прокрутили меньше чем на порог
      setVisible(container.scrollTop < HIDE_THRESHOLD);
    };

    container.addEventListener('scroll', onScroll, { passive: true });
    // сразу проверим начальное состояние
    onScroll();

    return () => {
      container.removeEventListener('scroll', onScroll);
    };
  }, []);

  return (
    <button
      className={`create-button${visible && !forceHidden ? '' : ' hidden'}`}
      onClick={() => connected && onOpenForm()}
      disabled={!connected}
      title={connected ? address!! : 'Сначала подключите TON-кошелёк'}
    >
      Создать пари
    </button>
  );
};
