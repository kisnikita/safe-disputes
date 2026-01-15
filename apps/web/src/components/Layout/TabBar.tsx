import React from 'react';
import betIcon from '../../../assets/bet-icon.svg';
import investigationIcon from '../../../assets/investigation-icon.svg';
import settingsIcon from '../../../assets/settings-icon.svg';
import searchIcon from '../../../assets/search-icon.svg';
import './TabBar.css';

const tabs: { id: string; label: string; icon?: string }[] = [
  { id: 'bets', label: 'Пари', icon: betIcon },
  { id: 'investigations', label: 'Расследования', icon: investigationIcon },
  { id: 'search', label: 'Поиск', icon: searchIcon },
  { id: 'settings', label: 'Настройки', icon: settingsIcon },
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
        aria-label={t.label}
      >
        {t.icon ? (
          <img className="tab-icon" src={t.icon} alt="" aria-hidden="true" />
        ) : (
          t.label
        )}
      </button>
    ))}
  </nav>
);
