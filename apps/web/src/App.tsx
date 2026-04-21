import { useState, useRef, useEffect } from 'react';
import { AppRoot } from './components/Layout/AppRoot';
import { TabBar } from './components/Layout/TabBar';
import { CreateBetButton } from './components/Layout/CreateBetButton';
import { CreateBetForm } from './components/CreateBetForm/CreateBetForm';
import { BetsSection, BetsSectionHandle } from './components/Bets/BetsSection';
import { InvestigationsSection, InvestigationsSectionHandle } from './components/Investigations/InvestigationsSection';
import { SettingsSection } from './components/Settings/SettingsSection';
import { SearchSection } from './components/Search/SearchSection';
import { useTelegramAuth } from './hooks/useTelegramAuth';
import { Spinner } from '@telegram-apps/telegram-ui';
import { init, backButton } from '@tma.js/sdk-react';
import './App.css';

export function App() {
  const { status, error } = useTelegramAuth();
  const [activeTab, setActiveTab] = useState<'bets' | 'investigations' | 'search' | 'settings'>('bets');
  const [showForm, setShowForm] = useState(false);
  const betsSectionRef = useRef<BetsSectionHandle>(null);
  const investigationsSectionRef = useRef<InvestigationsSectionHandle>(null);
  const [_, setModalOpen] = useState(false);

  useEffect(() => {
    init();
  }, []);

  useEffect(() => {
    if (status !== 'ready') return;
    if (!backButton.isSupported()) return;

    if (!backButton.isMounted()) {
      backButton.mount();
    }

    if (showForm) {
      backButton.show();
    } else {
      backButton.hide();
    }
  }, [status, showForm]);

  return (
    <AppRoot hideTonButton={showForm || status !== 'ready'}>
      {status === 'loading' && (
        <div className="center">
          <Spinner size="l" className="spinner" />
        </div>
      )}
      {status === 'error' && (
        <div className="center error">
          <p>{error}</p>
        </div>
      )}
      {status === 'ready' && (
        <>
          {activeTab === 'bets' && (
            <CreateBetButton
              onOpenForm={() => setShowForm(true)}
              forceHidden={showForm}
            />
          )}

          <div className="app">
            <div
              className={`content${
                !showForm && (activeTab === 'bets' || activeTab === 'investigations') ? ' no-scroll' : ''
              }${showForm ? ' create-mode' : ''}`}
            >
              {showForm && (
                <CreateBetForm
                  onClose={() => {
                    setShowForm(false);
                    setModalOpen(false);
                  }}
                  onCreated={() => betsSectionRef.current?.refresh()}
                />
              )}
              {!showForm && activeTab === 'bets' && (
                <BetsSection
                  ref={betsSectionRef}
                  onModalChange={open => setModalOpen(open)}
                />
              )}
              {!showForm && activeTab === 'investigations' && (
                <InvestigationsSection
                  ref={investigationsSectionRef}
                  onModalChange={open => setModalOpen(open)}
                />
              )}
              {!showForm && activeTab === 'search' && <SearchSection />}
              {!showForm && activeTab === 'settings' && <SettingsSection />}
            </div>
            {!showForm && (
              <TabBar
                active={activeTab}
                onChange={id => {
                  const nextTab = id as typeof activeTab;
                  if (nextTab === activeTab) {
                    if (nextTab === 'bets') {
                      betsSectionRef.current?.scrollToTop();
                      return;
                    }
                    if (nextTab === 'investigations') {
                      investigationsSectionRef.current?.scrollToTop();
                    }
                    return;
                  }
                  setActiveTab(nextTab);
                  if (nextTab === 'search' || nextTab === 'settings') {
                      window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: 0 } }));
                  }
                }}
              />
            )}
          </div>
        </>
      )}
    </AppRoot>
  );
}
