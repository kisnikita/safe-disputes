import { useState, useRef } from 'react';
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
import './App.css';

export function App() {
  const { status, error } = useTelegramAuth();
  const [activeTab, setActiveTab] = useState<'bets' | 'investigations' | 'search' | 'settings'>('bets');
  const [showForm, setShowForm] = useState(false);
  const betsSectionRef = useRef<BetsSectionHandle>(null);
  const investigationsSectionRef = useRef<InvestigationsSectionHandle>(null);
  const [_, setModalOpen] = useState(false);

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

          {showForm && (
            <CreateBetForm
              onClose={() => {
                setShowForm(false);
                setModalOpen(false);
              }}
              onCreated={() => betsSectionRef.current?.refresh()}
              onOpen={() => setModalOpen(true)}
            />
          )}

          <div className="app">
            <div
              className={`content${
                activeTab === 'bets' || activeTab === 'investigations' ? ' no-scroll' : ''
              }`}
            >
              {activeTab === 'bets' && (
                <BetsSection
                  ref={betsSectionRef}
                  onModalChange={open => setModalOpen(open)}
                />
              )}
              {activeTab === 'investigations' && (
                <InvestigationsSection
                  ref={investigationsSectionRef}
                  onModalChange={open => setModalOpen(open)}
                />
              )}
              {activeTab === 'search' && <SearchSection />}
              {activeTab === 'settings' && <SettingsSection />}
            </div>
            <TabBar
              active={activeTab}
              onChange={id => {
                const nextTab = id as typeof activeTab;
                if (nextTab === activeTab) return;
                setActiveTab(nextTab);
                if (nextTab === 'search' || nextTab === 'settings') {
                    window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: 0 } }));
                }
              }}
            />
          </div>
        </>
      )}
    </AppRoot>
  );
}
