// src/App.tsx
import React, { useEffect, useState } from 'react';
import './App.css';
import { retrieveRawInitData } from '@telegram-apps/sdk';

type Tab = 'bets' | 'investigations' | 'settings';
type InitDataResponse = { accessToken: string };

export function App() {
  const [status, setStatus] = useState<'loading' | 'error' | 'ready'>('loading');
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('bets');

  useEffect(() => {
    async function authenticate() {
      try {
        const initDataRaw = retrieveRawInitData()
        if (!initDataRaw) {
          setError('initDataRaw отсутствует');
          setStatus('error');
          return;
        }


        // Отправляем запрос на бэкенд с initDataRaw в заголовке
        const res = await fetch(`/api/auth/telegram`, {
          method: 'POST',
          headers: {
            Authorization: `tma ${initDataRaw}`,
            'Content-Type': 'application/json',
          },
        });
        if (!res.ok) {
          const errorBody = await res.text();
          throw new Error(`Ошибка авторизации: ${res.status}. Детали: ${errorBody}`);
        }

        const data: InitDataResponse = await res.json();
        localStorage.setItem('accessToken', data.accessToken);
        setStatus('ready');
      } catch (e: any) {
        console.error('[Auth] Error:', e.message);
        setError(e.message);
        setStatus('error');
      }
    }
    authenticate();
  }, []);

  if (status === 'loading') {
    return <div className="center"><p>Проверка авторизации...</p></div>;
  }
  if (status === 'error') {
    return <div className="center error"><p>{error}</p></div>;
  }
  

  // Основной UI после авторизации
  return (
    <div className="app">
      <div className="content">
        {activeTab === 'bets' && <BetsSection />}
        {activeTab === 'investigations' && <InvestigationsSection />}
        {activeTab === 'settings' && <SettingsSection />}
      </div>
      <nav className="tabbar">
        <button onClick={() => setActiveTab('bets')} className={activeTab === 'bets' ? 'active' : ''}>Пари</button>
        <button onClick={() => setActiveTab('investigations')} className={activeTab === 'investigations' ? 'active' : ''}>Расследования</button>
        <button onClick={() => setActiveTab('settings')} className={activeTab === 'settings' ? 'active' : ''}>Настройки</button>
      </nav>
    </div>
  );
}

function BetsSection() {
  const [subtab, setSubtab] = useState<'current' | 'new' | 'past'>('current');
  return (
    <div>
      <div className="subtabs">
        <button onClick={() => setSubtab('current')} className={subtab === 'current' ? 'active' : ''}>Текущие</button>
        <button onClick={() => setSubtab('new')} className={subtab === 'new' ? 'active' : ''}>Новые</button>
        <button onClick={() => setSubtab('past')} className={subtab === 'past' ? 'active' : ''}>Прошедшие</button>
      </div>
      <div className="subcontent">
        {subtab === 'current' && <p>Список текущих пари...</p>}
        {subtab === 'new' && <p>Список новых вызовов...</p>}
        {subtab === 'past' && <p>Список прошедших пари с фильтрами...</p>}
      </div>
    </div>
  );
}

function InvestigationsSection() {
  return <div className="center"><p>Скоро будет добавлено</p></div>;
}

function SettingsSection() {
  const [username, setUsername] = useState('');
  const [notifications, setNotifications] = useState(true);
  return (
    <div className="settings">
      <label>
        Username:
        <input value={username} onChange={e => setUsername(e.target.value)} placeholder="Введите username" />
      </label>
      <label>
        <input type="checkbox" checked={notifications} onChange={e => setNotifications(e.target.checked)} />
        Уведомления
      </label>
    </div>
  );
}
