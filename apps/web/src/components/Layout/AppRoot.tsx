// src/components/Layout/AppRoot.tsx
import React, { useState, useEffect } from 'react';
import { TonConnectUIProvider, TonConnectButton } from '@tonconnect/ui-react';
import './AppRoot.css';

interface AppRootProps {
  children: React.ReactNode;
  hideTonButton?: boolean;     // новый проп
}

export const AppRoot: React.FC<AppRootProps> = ({ children, hideTonButton = false }) => {
  const [scrollVisible, setScrollVisible] = useState(true);

  useEffect(() => {
    const webApp = (window as any)?.Telegram?.WebApp;
    if (!webApp) return;
    webApp.setBackgroundColor?.('#ffffff');
    webApp.setHeaderColor?.('#ffffff');
    webApp.setBottomBarColor?.('#ffffff');
  }, []);

  useEffect(() => {
    const container = document.querySelector<HTMLElement>('.content');
    if (!container) return;
    const HIDE_THRESHOLD = 30;
    const onScroll = () => setScrollVisible(container.scrollTop < HIDE_THRESHOLD);
    container.addEventListener('scroll', onScroll, { passive: true });
    onScroll();
    return () => container.removeEventListener('scroll', onScroll);
  }, []);

  // TonConnectButton видим только если оба флага false
  const isVisible = !hideTonButton && scrollVisible;

  return (
    <TonConnectUIProvider manifestUrl="https://tomato-adjacent-badger-155.mypinata.cloud/ipfs/bafkreidcmhwlwlsjqfuw23jphfvjdu2vyc2zwzmz2gyoxksjdjt4mgz5ru">
      <div className="app-root">
        <div className={`ton-button${isVisible ? '' : ' hidden'}`}>
          <TonConnectButton />
        </div>
        {children}
      </div>
    </TonConnectUIProvider>
  );
};
