import React from 'react';
import './TabBar.css';

const tabs: { id: string; label: string }[] = [
  { id: 'bets', label: 'Пари' },
  { id: 'investigations', label: 'Расследования' },
  { id: 'settings', label: 'Настройки' },
];

export const TabBar: React.FC<{
  active: string;
  onChange: (id: string) => void;
}> = ({ active, onChange }) => (
  <nav className="tabbar">
    {tabs.map(t => (
      <button
        key={t.id}
        className={active === t.id ? 'active' : ''}
        onClick={() => onChange(t.id)}
      >
        {t.label}
      </button>
    ))}
  </nav>
);
