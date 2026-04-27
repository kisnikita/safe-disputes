import { useState, useRef, useEffect } from 'react';
import { AppRoot } from './components/Layout/AppRoot';
import { TabBar } from './components/Layout/TabBar';
import { CreateBetButton } from './components/Layout/CreateBetButton';
import { CreateBetForm } from './components/CreateBetForm/CreateBetForm';
import { EvidenceForm } from './components/EvidenceForm/EvidenceForm';
import { BetsSection, BetsSectionHandle } from './components/Bets/BetsSection';
import { InvestigationsSection, InvestigationsSectionHandle } from './components/Investigations/InvestigationsSection';
import { SettingsSection } from './components/Settings/SettingsSection';
import { SearchSection } from './components/Search/SearchSection';
import { useTelegramAuth } from './hooks/useTelegramAuth';
import { apiFetch } from './utils/apiFetch';
import { Spinner } from '@telegram-apps/telegram-ui';
import { init, backButton } from '@tma.js/sdk-react';
import './App.css';

export function App() {
  const { status, error } = useTelegramAuth();
  const [activeTab, setActiveTab] = useState<'bets' | 'investigations' | 'search' | 'settings'>('bets');
  const [activeScreen, setActiveScreen] = useState<'tabs' | 'createBet' | 'evidence'>('tabs');
  const [evidenceDisputeId, setEvidenceDisputeId] = useState<string | null>(null);
  const [pendingBetModalId, setPendingBetModalId] = useState<string | null>(null);
  const [userPhotoUrl, setUserPhotoUrl] = useState<string | null>(null);
  const [username, setUsername] = useState<string>('');
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

    if (activeScreen !== 'tabs') {
      backButton.show();
    } else {
      backButton.hide();
    }
  }, [status, activeScreen]);

  useEffect(() => {
    if (status !== 'ready') return;

    (async () => {
      try {
        const res = await apiFetch('/api/v1/users/me');
        const { data } = (await res.json()) as { data?: { username?: string; photoUrl?: string | null } };
        setUserPhotoUrl(data?.photoUrl ?? null);
        setUsername(data?.username ?? '');
      } catch (fetchError) {
        console.error(fetchError);
        setUserPhotoUrl(null);
        setUsername('');
      }
    })();
  }, [status]);

  return (
    <AppRoot hideTonButton={activeScreen !== 'tabs' || status !== 'ready'}>
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
          {activeScreen === 'tabs' && activeTab === 'bets' && (
            <CreateBetButton
              onOpenForm={() => setActiveScreen('createBet')}
              forceHidden={activeScreen !== 'tabs'}
            />
          )}

          <div className="app">
            <div
              className={`content${
                activeScreen === 'tabs' && (activeTab === 'bets' || activeTab === 'investigations') ? ' no-scroll' : ''
              }${activeScreen !== 'tabs' ? ' create-mode' : ''}`}
            >
              {activeScreen === 'createBet' && (
                <CreateBetForm
                  onClose={() => {
                    setActiveScreen('tabs');
                    setModalOpen(false);
                  }}
                  onCreated={() => betsSectionRef.current?.refresh()}
                />
              )}
              {activeScreen === 'evidence' && evidenceDisputeId && (
                <EvidenceForm
                  disputeId={evidenceDisputeId}
                  onClose={() => {
                    const targetBetId = evidenceDisputeId;
                    setActiveScreen('tabs');
                    setActiveTab('bets');
                    setEvidenceDisputeId(null);
                    setPendingBetModalId(targetBetId);
                    setModalOpen(false);
                  }}
                  onSubmitted={() => {
                    betsSectionRef.current?.refresh();
                  }}
                />
              )}
              {activeScreen === 'tabs' && activeTab === 'bets' && (
                <BetsSection
                  ref={betsSectionRef}
                  onModalChange={open => setModalOpen(open)}
                  initialOpenBetId={pendingBetModalId}
                  onInitialOpenBetHandled={() => setPendingBetModalId(null)}
                  onOpenEvidence={disputeId => {
                    setEvidenceDisputeId(disputeId);
                    setActiveScreen('evidence');
                    setModalOpen(false);
                  }}
                />
              )}
              {activeScreen === 'tabs' && activeTab === 'investigations' && (
                <InvestigationsSection
                  ref={investigationsSectionRef}
                  onModalChange={open => setModalOpen(open)}
                />
              )}
              {activeScreen === 'tabs' && activeTab === 'search' && <SearchSection />}
              {activeScreen === 'tabs' && activeTab === 'settings' && (
                <SettingsSection username={username} userPhotoUrl={userPhotoUrl} />
              )}
            </div>
            {activeScreen === 'tabs' && (
              <TabBar
                active={activeTab}
                userPhotoUrl={userPhotoUrl}
                username={username}
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
