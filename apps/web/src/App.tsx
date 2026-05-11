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

type ChangesResponse = {
  data: {
    disputes: Array<{ disputeID: string; status: 'current' | 'new' | 'passed' }>;
    investigations: Array<{ investigationID: string; status: 'current' | 'passed' }>;
  };
  unreadCounts: {
    disputes: { current: number; new: number; passed: number };
    investigations: { current: number; passed: number };
  };
  nextSince: string;
};

const CHANGES_POLL_INTERVAL_MS = 10_000;

export function App() {
  const { status, error } = useTelegramAuth();
  const [activeTab, setActiveTab] = useState<'bets' | 'investigations' | 'search' | 'settings'>('bets');
  const [activeScreen, setActiveScreen] = useState<'tabs' | 'createBet' | 'evidence'>('tabs');
  const [evidenceDisputeId, setEvidenceDisputeId] = useState<string | null>(null);
  const [pendingBetModalId, setPendingBetModalId] = useState<string | null>(null);
  const [betsInitialSubtab, setBetsInitialSubtab] = useState<'current' | 'new' | 'passed'>('current');
  const [userPhotoUrl, setUserPhotoUrl] = useState<string | null>(null);
  const [username, setUsername] = useState<string>('');
  const [hasBetsUnread, setHasBetsUnread] = useState(false);
  const [hasInvestigationsUnread, setHasInvestigationsUnread] = useState(false);
  const [disputesUnreadCounts, setDisputesUnreadCounts] = useState<{ current: number; new: number; passed: number }>({
    current: 0,
    new: 0,
    passed: 0,
  });
  const [investigationsUnreadCounts, setInvestigationsUnreadCounts] = useState<{ current: number; passed: number }>({
    current: 0,
    passed: 0,
  });
  const [changesSnapshot, setChangesSnapshot] = useState<ChangesResponse | null>(null);
  const betsSectionRef = useRef<BetsSectionHandle>(null);
  const investigationsSectionRef = useRef<InvestigationsSectionHandle>(null);
  const evidenceSubmittedRef = useRef(false);
  const sinceRef = useRef<string | null>(null);
  const pollTimerRef = useRef<number | null>(null);
  const [_, setModalOpen] = useState(false);

  const applyOptimisticUnreadDelta = (entity: 'disputes' | 'investigations', tab: 'current' | 'new' | 'passed', delta: number) => {
    if (entity === 'disputes') {
      setDisputesUnreadCounts(prev => {
        const next = { ...prev, [tab]: Math.max(0, (prev as Record<string, number>)[tab] + delta) } as typeof prev;
        setHasBetsUnread(next.current + next.new + next.passed > 0);
        return next;
      });
      return;
    }
    if (tab === 'new') return;
    setInvestigationsUnreadCounts(prev => {
      const next = { ...prev, [tab]: Math.max(0, (prev as Record<string, number>)[tab] + delta) } as typeof prev;
      setHasInvestigationsUnread(next.current + next.passed > 0);
      return next;
    });
  };

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

  useEffect(() => {
    if (status !== 'ready') return;
    if (!sinceRef.current) {
      sinceRef.current = new Date().toISOString();
    }

    const pollNow = async () => {
      const since = sinceRef.current;
      if (!since) return;
      try {
        const params = new URLSearchParams({ since });
        const res = await apiFetch(`/api/v1/changes?${params}`);
        if (!res.ok) throw new Error('changes failed');
        const payload = (await res.json()) as ChangesResponse;
        sinceRef.current = payload.nextSince || sinceRef.current;
        const disputesUnread = payload.unreadCounts.disputes;
        const investigationsUnread = payload.unreadCounts.investigations;
        setDisputesUnreadCounts(disputesUnread);
        setInvestigationsUnreadCounts(investigationsUnread);
        setChangesSnapshot(payload);
        setHasBetsUnread(disputesUnread.current + disputesUnread.new + disputesUnread.passed > 0);
        setHasInvestigationsUnread(investigationsUnread.current + investigationsUnread.passed > 0);
        window.dispatchEvent(new CustomEvent('app-changes', { detail: payload }));
      } catch {
        // ignore polling errors
      }
    };

    const schedule = () => {
      if (pollTimerRef.current !== null) {
        window.clearInterval(pollTimerRef.current);
        pollTimerRef.current = null;
      }
      if (document.hidden) return;
      const intervalMs = CHANGES_POLL_INTERVAL_MS;
      pollTimerRef.current = window.setInterval(() => {
        void pollNow();
      }, intervalMs);
    };

    void pollNow();
    schedule();
    const onPollNow = () => {
      void pollNow();
      schedule();
    };
    const onVisibilityChange = () => {
      schedule();
      if (!document.hidden) {
        void pollNow();
      }
    };
    window.addEventListener('pollNow', onPollNow);
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => {
      window.removeEventListener('pollNow', onPollNow);
      document.removeEventListener('visibilitychange', onVisibilityChange);
      if (pollTimerRef.current !== null) {
        window.clearInterval(pollTimerRef.current);
        pollTimerRef.current = null;
      }
    };
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
                  onCreated={() => {
                    setBetsInitialSubtab('new');
                    setActiveTab('bets');
                    window.dispatchEvent(new Event('pollNow'));
                    betsSectionRef.current?.refresh();
                  }}
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
                    if (evidenceSubmittedRef.current) {
                      setBetsInitialSubtab('current');
                      setPendingBetModalId(null);
                    } else {
                      setPendingBetModalId(targetBetId);
                    }
                    evidenceSubmittedRef.current = false;
                    setModalOpen(false);
                  }}
                  onSubmitted={() => {
                    evidenceSubmittedRef.current = true;
                    setBetsInitialSubtab('current');
                    window.dispatchEvent(new Event('pollNow'));
                    betsSectionRef.current?.refresh();
                  }}
                />
              )}
              {activeScreen === 'tabs' && activeTab === 'bets' && (
                <BetsSection
                  ref={betsSectionRef}
                  onModalChange={open => setModalOpen(open)}
                  initialSubtab={betsInitialSubtab}
                  unreadCountsByTabExternal={disputesUnreadCounts}
                  changesSnapshot={changesSnapshot}
                  onOptimisticUnreadDelta={(tab, delta) => applyOptimisticUnreadDelta('disputes', tab, delta)}
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
                  unreadCountsByTabExternal={investigationsUnreadCounts}
                  changesSnapshot={changesSnapshot}
                  onOptimisticUnreadDelta={(tab, delta) => applyOptimisticUnreadDelta('investigations', tab, delta)}
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
                hasBetsUnread={hasBetsUnread}
                hasInvestigationsUnread={hasInvestigationsUnread}
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
                  if (activeTab === 'bets') {
                    setBetsInitialSubtab('current');
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
