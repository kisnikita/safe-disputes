import React, { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import betIcon from '../../../assets/bet-icon.svg';
import investigationIcon from '../../../assets/investigation-icon.svg';
import settingsIcon from '../../../assets/settings-icon.svg';
import searchIcon from '../../../assets/search-icon.svg';
import './TabBar.css';

const tabs: { id: string; label: string; icon?: string }[] = [
  { id: 'bets', label: 'Пари', icon: betIcon },
  { id: 'investigations', label: 'Суд', icon: investigationIcon },
  { id: 'search', label: 'Поиск', icon: searchIcon },
  { id: 'settings', label: 'Профиль', icon: settingsIcon },
];

export const TabBar: React.FC<{
  active: string;
  onChange: (id: string) => void;
}> = ({ active, onChange }) => {
  const navRef = useRef<HTMLElement | null>(null);
  const trackRef = useRef<HTMLDivElement | null>(null);
  const buttonRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  const [indicatorStyle, setIndicatorStyle] = useState<React.CSSProperties>({});
  const activeIndex = Math.max(
    0,
    tabs.findIndex(t => t.id === active)
  );
  const activeId = useMemo(() => tabs[activeIndex]?.id, [activeIndex]);

  useEffect(() => {
    const node = navRef.current;
    if (!node) return;
    const root = document.documentElement;
    const update = () => {
      root.style.setProperty('--app-tabbar-height', `${node.offsetHeight}px`);
    };
    update();
    if (typeof ResizeObserver !== 'undefined') {
      const ro = new ResizeObserver(update);
      ro.observe(node);
      return () => ro.disconnect();
    }
    window.addEventListener('resize', update);
    return () => window.removeEventListener('resize', update);
  }, []);

  useLayoutEffect(() => {
    const track = trackRef.current;
    if (!track || !activeId) return;
    const button = buttonRefs.current[activeId];
    if (!button) return;
    const trackRect = track.getBoundingClientRect();
    const buttonRect = button.getBoundingClientRect();
    const translateX = buttonRect.left - trackRect.left;
    setIndicatorStyle({
      width: `${buttonRect.width}px`,
      transform: `translateX(${translateX}px) scaleX(1)`,
    });
  }, [activeId]);

  useEffect(() => {
    const track = trackRef.current;
    if (!track) return;
    const ro = new ResizeObserver(() => {
      const button = activeId ? buttonRefs.current[activeId] : null;
      if (!button) return;
      const trackRect = track.getBoundingClientRect();
      const buttonRect = button.getBoundingClientRect();
      const translateX = buttonRect.left - trackRect.left;
      setIndicatorStyle({
        width: `${buttonRect.width}px`,
        transform: `translateX(${translateX}px) scaleX(1)`,
      });
    });
    ro.observe(track);
    Object.values(buttonRefs.current).forEach(node => node && ro.observe(node));
    return () => ro.disconnect();
  }, [activeId]);

  return (
    <nav
      ref={navRef}
      className="tabbar"
      style={
        {
          '--tab-count': tabs.length,
          '--tab-index': activeIndex,
        } as React.CSSProperties
      }
    >
      <div className="tabbar-track" ref={trackRef}>
        {tabs.map(t => (
          <button
            key={t.id}
            className={active === t.id ? 'active' : ''}
            onClick={() => onChange(t.id)}
            aria-label={t.label}
            ref={node => {
              buttonRefs.current[t.id] = node;
            }}
          >
            {t.icon ? (
              <span className="tabbar-icon" aria-hidden="true">
                <img
                  className="tab-icon"
                  src={t.icon}
                  alt=""
                  draggable={false}
                />
              </span>
            ) : null}
            <span className="tabbar-label">{t.label}</span>
          </button>
        ))}
        <span className="tabbar-indicator" aria-hidden="true" style={indicatorStyle} />
      </div>
    </nav>
  );
};
