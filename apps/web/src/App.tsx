import React, { useState, useRef } from 'react';
import { AppRoot } from './components/Layout/AppRoot';
import { TabBar } from './components/Layout/TabBar';
import { CreateBetButton } from './components/Layout/CreateBetButton';
import { CreateBetForm } from './components/CreateBetForm/CreateBetForm';
import { BetsSection, BetsSectionHandle } from './components/Bets/BetsSection';
import { InvestigationsSection, InvestigationsSectionHandle } from './components/Investigations/InvestigationsSection';
import { SettingsSection } from './components/Settings/SettingsSection';
import { useTelegramAuth } from './hooks/useTelegramAuth';
import './App.css';

export function App() {
  const { status, error } = useTelegramAuth();
  const [activeTab, setActiveTab] = useState<'bets' | 'investigations' | 'settings'>('bets');
  const [showForm, setShowForm] = useState(false);
  const betsSectionRef = useRef<BetsSectionHandle>(null);
  const investigationsSectionRef = useRef<InvestigationsSectionHandle>(null);
  const [modalOpen, setModalOpen] = useState(false);

  if (status === 'loading') return <div className="center"><p>Проверка...</p></div>;
  if (status === 'error')   return <div className="center error"><p>{error}</p></div>;

  return (
    <AppRoot hideTonButton={showForm || modalOpen}>
      {activeTab === 'bets' && !showForm && !modalOpen && (
        <CreateBetButton onOpenForm={() => setShowForm(true)} />
      )}

      {showForm && (
        <CreateBetForm
          onClose={() => setShowForm(false)}
          onCreated={() => betsSectionRef.current?.refresh()}
          onOpen={() => setModalOpen(true)}
        />
      )}

      <div className="app">
        <div className="content">
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
          {activeTab === 'settings' && <SettingsSection />}
        </div>
        <TabBar
          active={activeTab}
          onChange={id => setActiveTab(id as any)}
        />
      </div>
    </AppRoot>
  );
}
