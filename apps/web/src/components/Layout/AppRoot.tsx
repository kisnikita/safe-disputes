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
    <TonConnectUIProvider manifestUrl="https://YOUR_MANIFEST_URL">
      <div className="app-root">
        <div className={`ton-button${isVisible ? '' : ' hidden'}`}>
          <TonConnectButton />
        </div>
        {children}
      </div>
    </TonConnectUIProvider>
  );
};
