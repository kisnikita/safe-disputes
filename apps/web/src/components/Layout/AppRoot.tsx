// src/components/Layout/AppRoot.tsx
import React, { useState, useEffect } from 'react';
import { AppRoot as TelegramAppRoot } from '@telegram-apps/telegram-ui';
import { TonConnectUIProvider, TonConnectButton } from '@tonconnect/ui-react';
import { HIDE_THRESHOLD } from '../../utils/constants';
import './AppRoot.css';

interface AppRootProps {
  children: React.ReactNode;
  hideTonButton?: boolean;     // новый проп
}

const syncAppHeight = () => {
  const root = document.documentElement;
  const webApp = (window as any)?.Telegram?.WebApp;
  const stableHeight = Number(webApp?.viewportStableHeight);
  if (Number.isFinite(stableHeight) && stableHeight > 0) {
    root.style.setProperty('--app-height', `${stableHeight}px`);
    return;
  }
  const candidates = [document.documentElement.clientHeight, window.innerHeight];
  if (window.visualViewport) {
    candidates.push(Math.round(window.visualViewport.height));
  }
  const valid = candidates.filter(value => Number.isFinite(value) && value > 0);
  if (valid.length > 0) {
    root.style.setProperty('--app-height', `${Math.min(...valid)}px`);
  }
};

export const AppRoot: React.FC<AppRootProps> = ({ children, hideTonButton = false }) => {
  const [scrollVisible, setScrollVisible] = useState(true);

  useEffect(() => {
    const webApp = (window as any)?.Telegram?.WebApp;
    if (!webApp) return;
    webApp.setHeaderColor?.('#0F172A');
    webApp.setBackgroundColor?.('#0F172A'); // just for bottom bar color on desktop.
  }, []);

  useEffect(() => {
    const webApp = (window as any)?.Telegram?.WebApp;
    if (!webApp?.requestFullscreen) return;
    const platform = String(webApp.platform || '').toLowerCase();
    const isMobile = platform === 'android' || platform === 'ios';
    if (!isMobile) return;

    webApp.requestFullscreen();
    syncAppHeight();
    const rafId = requestAnimationFrame(syncAppHeight);
    const timeoutA = window.setTimeout(syncAppHeight, 120);
    const timeoutB = window.setTimeout(syncAppHeight, 320);

    return () => {
      cancelAnimationFrame(rafId);
      window.clearTimeout(timeoutA);
      window.clearTimeout(timeoutB);
    };
  }, []);

  useEffect(() => {
    const webApp = (window as any)?.Telegram?.WebApp;
    if (!webApp?.enableClosingConfirmation) return;
    webApp.enableClosingConfirmation();
    return () => webApp.disableClosingConfirmation?.();
  }, []);

  useEffect(() => {
    const webApp = (window as any)?.Telegram?.WebApp;
    if (!webApp?.disableVerticalSwipes) return;
    webApp.disableVerticalSwipes();
    return () => webApp.enableVerticalSwipes?.();
  }, []);

  useEffect(() => {
    const onSubtabChange = (event: Event) => {
      const custom = event as CustomEvent<{ scrollTop: number }>;
      if (typeof custom.detail?.scrollTop === 'number') {
        setScrollVisible(custom.detail.scrollTop < HIDE_THRESHOLD);
      }
    };
    const onScroll = (event?: Event) => {
      const target = event?.target as HTMLElement | null;
      if (target?.classList?.contains('subcontent-panel')) {
        setScrollVisible(target.scrollTop < HIDE_THRESHOLD);
        return;
      }
      if (target?.classList?.contains('content')) {
        setScrollVisible(target.scrollTop < HIDE_THRESHOLD);
        return;
      }
    };

    const container = document.querySelector<HTMLElement>('.content');
    if (container) {
      setScrollVisible(container.scrollTop < HIDE_THRESHOLD);
    }

    document.addEventListener('scroll', onScroll, { passive: true, capture: true });
    window.addEventListener('subtab-scroll-sync', onSubtabChange as EventListener);
    return () => {
      document.removeEventListener('scroll', onScroll, true);
      window.removeEventListener('subtab-scroll-sync', onSubtabChange as EventListener);
    };
  }, []);

  useEffect(() => {
    syncAppHeight();
  }, []);

  // TonConnectButton видим только если оба флага false
  const isVisible = !hideTonButton && scrollVisible;

  return (
    <TonConnectUIProvider manifestUrl="https://tomato-adjacent-badger-155.mypinata.cloud/ipfs/bafkreihvbraicmhxbbsgzkt2ochdjuptqg37cdybjxz7dd2joa2avohwii">
      <TelegramAppRoot>
        <div className="app-root">
          <div className={`ton-button${isVisible ? '' : ' hidden'}`}>
            <TonConnectButton />
          </div>
          {children}
        </div>
      </TelegramAppRoot>
    </TonConnectUIProvider>
  );
};
