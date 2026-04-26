import React, { useState } from 'react';
import './CreateBetButton.css';
import { useTonConnect } from '../../hooks/useTonConnect';
import { HIDE_THRESHOLD } from '../../utils/constants';
import { useScrollVisibility } from '../../hooks/useScrollVisibility';
import { useWalletConnectPopup } from '../../utils/walletPopup';

interface Props {
  onOpenForm: () => void;
  forceHidden?: boolean;
}

export const CreateBetButton: React.FC<Props> = ({ onOpenForm, forceHidden = false }) => {
  const { connected } = useTonConnect();
  const showWalletConnectPopup = useWalletConnectPopup();
  const visible = useScrollVisibility(HIDE_THRESHOLD);
  const [shake, setShake] = useState(false);

  const handleClick = async () => {
    if (connected) {
      onOpenForm();
      return;
    }

    setShake(false);
    requestAnimationFrame(() => setShake(true));
    await showWalletConnectPopup();
  };

  return (
    <button
      className={`create-button${visible && !forceHidden ? '' : ' hidden'}${connected ? '' : ' create-button-wallet-disconnected'}${shake ? ' create-button-shake' : ''}`}
      onClick={() => void handleClick()}
      onAnimationEnd={() => setShake(false)}
      aria-disabled={!connected}
      title={connected ? undefined : 'Подключите TON-кошелёк'}
    >
      Создать пари
    </button>
  );
};
