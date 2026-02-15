import React from 'react';
import './CreateBetButton.css';
import { useTonConnect } from '../../hooks/useTonConnect';
import { HIDE_THRESHOLD } from '../../utils/constants';
import { useScrollVisibility } from '../../hooks/useScrollVisibility';

interface Props {
  onOpenForm: () => void;
  forceHidden?: boolean;
}

export const CreateBetButton: React.FC<Props> = ({ onOpenForm, forceHidden = false }) => {
  const { address, connected } = useTonConnect();
  const visible = useScrollVisibility(HIDE_THRESHOLD);

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
