// src/components/Layout/AppRoot.tsx
import React, { useState, useEffect } from 'react';
import { AppRoot as TelegramAppRoot } from '@telegram-apps/telegram-ui';
import { TonConnectUIProvider, TonConnectButton } from '@tonconnect/ui-react';
import { HIDE_THRESHOLD } from '../../utils/constants';
import { useScrollVisibility } from '../../hooks/useScrollVisibility';
import './AppRoot.css';

interface AppRootProps {
  children: React.ReactNode;
  hideTonButton?: boolean;
}

const getWebApp = () => (typeof window === 'undefined' ? undefined : (window as any)?.Telegram?.WebApp);

export const AppRoot: React.FC<AppRootProps> = ({ children, hideTonButton = false }) => {
  const scrollVisible = useScrollVisibility(HIDE_THRESHOLD);
  const [showRotateHint, setShowRotateHint] = useState(false);

  const syncAppHeight = () => {
      const root = document.documentElement;
      const stableHeight = Number(getWebApp()?.viewportStableHeight);
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

  // static configuration
  useEffect(() => {
    const webApp = getWebApp();
    webApp?.setHeaderColor?.('#0F172A');
    webApp?.setBackgroundColor?.('#0F172A'); // just for bottom bar color on desktop.
    if (webApp?.enableClosingConfirmation) webApp.enableClosingConfirmation();
    if (webApp?.disableVerticalSwipes) webApp.disableVerticalSwipes();
    syncAppHeight(); // for desktop
    return () => {
      webApp.disableClosingConfirmation?.();
      webApp.enableVerticalSwipes?.();
    };
  }, []);

  // dynamic configuration
  useEffect(() => {
    const webApp = getWebApp();
    if (!webApp?.requestFullscreen) return;
    const platform = String(webApp.platform || '').toLowerCase();
    const isMobile = platform === 'android' || platform === 'ios';
    if (!isMobile) return;
    let rafId = 0;
    let syncHeightTimeout = 0;
    let activateTimeout = 0;
    let isOrientationListening = false;

    const syncAfterRotate = () => {
      if (rafId) cancelAnimationFrame(rafId);
      if (syncHeightTimeout) window.clearTimeout(syncHeightTimeout);
      syncAppHeight();
      rafId = requestAnimationFrame(syncAppHeight);
      syncHeightTimeout = window.setTimeout(syncAppHeight, 120);
    };

    const isLandscape = () => {
      const width = window.visualViewport?.width ?? window.innerWidth;
      const height = window.visualViewport?.height ?? window.innerHeight;
      return width > height;
    };
    const updateOrientationState = () => {
      syncAfterRotate();
      const landscape = isLandscape();
      setShowRotateHint(landscape);
      if (!landscape) {
        webApp.lockOrientation?.();
        window.removeEventListener('orientationchange', updateOrientationState);
        isOrientationListening = false;
        return;
      }
      webApp.unlockOrientation?.();
      if (!isOrientationListening) {
        window.addEventListener('orientationchange', updateOrientationState);
        isOrientationListening = true;
      }
    };
    const onActivated = () => {
      if (activateTimeout) window.clearTimeout(activateTimeout);
      activateTimeout = window.setTimeout(updateOrientationState, 120);
    };

    // start flow
    webApp.requestFullscreen();
    updateOrientationState();
    webApp.onEvent?.('activated', onActivated);

    return () => {
      cancelAnimationFrame(rafId);
      window.clearTimeout(syncHeightTimeout);
      window.clearTimeout(activateTimeout);
      window.removeEventListener('orientationchange', updateOrientationState);
      webApp.offEvent?.('activated', onActivated);
      webApp.unlockOrientation?.();
    };
  }, []);

  const isVisible = !hideTonButton && scrollVisible && !showRotateHint;
  return (
    <TonConnectUIProvider manifestUrl="https://tomato-adjacent-badger-155.mypinata.cloud/ipfs/bafkreihvbraicmhxbbsgzkt2ochdjuptqg37cdybjxz7dd2joa2avohwii">
      <TelegramAppRoot>
        <div className="app-root">
          {showRotateHint ? (
            <div className="orientation-lock">
              <div className="orientation-lock-illustration" aria-hidden="true">
                <svg className="orientation-lock-svg" viewBox="0 0 120 120" xmlns="http://www.w3.org/2000/svg">
                  <defs>
                    <marker
                      id="rotate-arrow-head"
                      markerWidth="8"
                      markerHeight="8"
                      refX="1.8"
                      refY="4"
                      orient="auto"
                      markerUnits="userSpaceOnUse"
                    >
                      <path d="M7,1 L0.8,4 L7,7" className="rotate-arrow-head-stroke" />
                    </marker>
                  </defs>
                  <g className="orientation-lock-phone">
                    <rect x="42" y="26" width="36" height="68" rx="8" className="phone-body" />
                    <rect x="46" y="34" width="28" height="50" rx="4" className="phone-screen" />
                    <circle cx="60" cy="89" r="2.5" className="phone-dot" />
                  </g>
                  <path d="M27 49a34 34 0 0 1 24-24" className="rotate-arrow" markerStart="url(#rotate-arrow-head)" />
                  <path d="M93 71a34 34 0 0 1-24 24" className="rotate-arrow" markerStart="url(#rotate-arrow-head)" />
                </svg>
              </div>
              <div className="orientation-lock-title">Переверните экран</div>
              <div className="orientation-lock-text">Использование в альбомном режиме недоступно</div>
            </div>
          ) : (
            <>
              <div className={`ton-button${isVisible ? '' : ' hidden'}`}>
                <TonConnectButton />
              </div>
              {children}
            </>
          )}
        </div>
      </TelegramAppRoot>
    </TonConnectUIProvider>
  );
};
