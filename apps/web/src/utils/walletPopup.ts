import { popup } from '@tma.js/sdk-react';
import { useTonConnectUI } from '@tonconnect/ui-react';

interface WalletConnectPopupParams {
  message?: string;
}

const DEFAULT_MESSAGE = 'Подключите TON-кошелёк, чтобы выполнить действие';

export const useWalletConnectPopup = () => {
  const [tonConnectUI] = useTonConnectUI();

  return async ({ message = DEFAULT_MESSAGE }: WalletConnectPopupParams = {}): Promise<void> => {
    if (popup.isSupported()) {
      const buttonId = await popup.show({
        message,
        buttons: [
          { id: 'connect', type: 'default', text: 'Подключить кошелёк' },
          { id: 'ok', type: 'ok' },
        ],
      });

      if (buttonId === 'connect') {
        await tonConnectUI.openModal();
      }
      return;
    }

    if (window.confirm(`${message}\n\nОткрыть подключение кошелька?`)) {
      await tonConnectUI.openModal();
    }
  };
};
