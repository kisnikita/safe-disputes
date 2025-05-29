// src/components/Layout/CreateBetButton.tsx
import React, { useState, useEffect } from 'react';
import { useTonWallet } from '@tonconnect/ui-react';
import './CreateBetButton.css';

interface Props {
  onOpenForm: () => void;
}

export const CreateBetButton: React.FC<Props> = ({ onOpenForm }) => {
  const wallet = useTonWallet();
  const disabled = !wallet;
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
      className={`create-button${visible ? '' : ' hidden'}`}
      onClick={() => !disabled && onOpenForm()}
      disabled={disabled}
      title={disabled ? 'Сначала подключите TON-кошелёк' : wallet?.account.address}
    >
      Создать пари
    </button>
  );
};
