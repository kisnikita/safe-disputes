// src/components/Investigations/InvestigationsSection.tsx
import React, {
  useState,
  useEffect,
  useRef,
  useCallback,
  forwardRef,
  useImperativeHandle
} from 'react';
import { apiFetch } from '../../utils/apiFetch';
import './InvestigationsSection.css';
import { InvestigationDetailsModal } from './InvestigationDetailsModal';

interface Investigation {
  id: string;
  dispute_id: string;
  title: string;
  status: 'current' | 'passed';
  ends_at: string; // ISO date string
  result: 'new' | 'sent' | 'correct' | 'incorrect';
  vote?: string;
}

interface TopUser {
  username: string;
  rating: number;
}

export interface InvestigationsSectionHandle {
  refresh: () => void;
}

interface Props {
  onModalChange: (open: boolean) => void;
}

export const InvestigationsSection = forwardRef<InvestigationsSectionHandle, Props>(
  ({ onModalChange }, ref) => {
    const [subtab, setSubtab] = useState<'current' | 'passed'>('current');
    const [items, setItems] = useState<Investigation[]>([]);
    const [loading, setLoading] = useState(false);
    const [hasMore, setHasMore] = useState(true);
    const [showRating, setShowRating] = useState(false);
    const [topUsers, setTopUsers] = useState<TopUser[]>([]);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const cursorRef = useRef<string | null>(null);
    const observer = useRef<IntersectionObserver | null>(null);

    const fetchInvestigations = useCallback(async () => {
      cursorRef.current = null;
      setHasMore(true);
      setLoading(true);
      const params = new URLSearchParams({ status: subtab, limit: '10' });
      try {
        const res = await apiFetch(`/api/v1/investigations?${params}`);
        if (!res.ok) throw new Error();
        const { data, nextCursor } = (await res.json()) as {
          data: Investigation[];
          nextCursor: string | null;
        };
        setItems(data);
        cursorRef.current = nextCursor;
        setHasMore(!!nextCursor);
      } catch {
        setItems([]);
        setHasMore(false);
      } finally {
        setLoading(false);
      }
    }, [subtab]);

    const loadMore = useCallback(async () => {
      if (loading || !hasMore) return;
      setLoading(true);
      const params = new URLSearchParams({ status: subtab, limit: '10' });
      if (cursorRef.current) params.set('cursor', cursorRef.current);
      try {
        const res = await apiFetch(`/api/v1/investigations?${params}`);
        if (!res.ok) throw new Error();
        const { data, nextCursor } = (await res.json()) as {
          data: Investigation[];
          nextCursor: string | null;
        };
        setItems(prev => [...prev, ...data]);
        cursorRef.current = nextCursor;
        setHasMore(!!nextCursor);
      } catch {
        // ignore
      } finally {
        setLoading(false);
      }
    }, [subtab, loading, hasMore]);

    const fetchTopUsers = useCallback(async () => {
      try {
        const res = await apiFetch(`/api/v1/users/top`);
        if (!res.ok) throw new Error();
        const { data } = (await res.json()) as { data: TopUser[] };
        setTopUsers(data.slice(0, 100));
      } catch {
        setTopUsers([]);
      }
    }, []);

    useEffect(() => {
      fetchInvestigations();
    }, [fetchInvestigations]);

    useImperativeHandle(ref, () => ({ refresh: fetchInvestigations }));

    const lastRef = useCallback(
      (node: HTMLDivElement | null) => {
        if (loading || !hasMore) return;
        if (observer.current) observer.current.disconnect();
        observer.current = new IntersectionObserver(entries => {
          if (entries[0].isIntersecting) loadMore();
        });
        if (node) observer.current.observe(node);
      },
      [loading, hasMore, loadMore]
    );

    const getBadge = (result: Investigation['result']) => {
      switch (result) {
        case 'new': return { color: 'blue', text: 'Новое' };
        case 'sent': return { color: 'green', text: 'Голос отдан' };
        case 'correct': return { color: 'green', text: 'Верно' };
        case 'incorrect': return { color: 'red', text: 'Неверно' };
        default: return null;
      }
    };

    const openDetails = (id: string) => {
      setSelectedId(id);
      onModalChange(true);
    };
    const closeDetails = () => {
      setSelectedId(null);
      onModalChange(false);
    };

  const getTimeRemaining = (endsAt: string) => {
  // Парсим время окончания как UTC-момент
  const endMs = Date.parse(endsAt);

  // Считаем, сколько минут нужно добавить к локальному времени,
  // чтобы получить UTC+3:
  //   локальный оффсет = new Date().getTimezoneOffset() в минутах (от UTC до локали)
  //   нам нужно получить смещение от локали до UTC+3 → 180 мин - локальный оффсет
  const offsetToUTCPlus3Min = -new Date().getTimezoneOffset();
  // текущее «UTC+3» в миллисекундах
  const nowUtcPlus3 = Date.now() + offsetToUTCPlus3Min * 60_000;

  const diff = endMs - nowUtcPlus3;
  if (diff <= 0) return 'завершено';

  const days = Math.floor(diff / (1000 * 60 * 60 * 24));
  const hours = Math.floor((diff / (1000 * 60 * 60)) % 24);
  const minutes = Math.floor((diff / (1000 * 60)) % 60);
  return `${days}д ${hours}ч ${minutes}м`;
};

    return (
      <>
        <div className="investigations-section">
          <div className="header">
            <button
              className="rating-button"
              onClick={() => {
                setShowRating(prev => {
                  if (!prev) fetchTopUsers();
                  return !prev;
                });
              }}
            >
              Рейтинг
            </button>
          </div>

          {showRating ? (
            <div className="rating-panel">
              <h3>Топ 100 пользователей</h3>
              <ul>
                {topUsers.map((user, idx) => (
                  <li key={user.username}>
                    <span>{idx + 1}. {user.username}</span>
                    <span className="rating-value">{user.rating}</span>
                  </li>
                ))}
                {topUsers.length === 0 && <li>Нет данных</li>}
              </ul>
            </div>
          ) : (
            <div className="main-panel">
              <div className="subtabs">
                {(['current', 'passed'] as const).map(tab => (
                  <button
                    key={tab}
                    onClick={() => setSubtab(tab)}
                    className={subtab === tab ? 'active' : ''}
                  >
                    {tab === 'current' ? 'Текущие' : 'Прошедшие'}
                  </button>
                ))}
              </div>

              <div className="subcontent">
                {items.map((inv, idx) => {
                  const isLast = idx === items.length - 1;
                  const badge = getBadge(inv.result);

                  return (
                    <div
                      key={inv.dispute_id}
                      ref={isLast ? lastRef : null}
                      className="investigation-card"
                      onClick={() => openDetails(inv.id)}
                    >
                      <h4>{inv.title}</h4>
                      <p>Окончание через: {getTimeRemaining(inv.ends_at)}</p>
                      {inv.vote && <p>Ваш голос: {inv.vote == 'p1' ? 'Пользователь 1' : "Пользователь 2"}</p>}

                      {badge && (
                        <div className="result-badge" style={{ borderColor: badge.color }}>
                          <span className={`dot ${badge.color}`} />
                          {badge.text}
                        </div>
                      )}
                    </div>
                  );
                })}

                {loading && <div className="loading">Загрузка…</div>}
                {!loading && items.length === 0 && (
                  <div className="empty-message">Тут пока пусто</div>
                )}
              </div>
            </div>
          )}

          {!!selectedId && (
            <InvestigationDetailsModal
              id={selectedId}
              onClose={closeDetails}
              onCompleted={() => {
                closeDetails();
                fetchInvestigations();
              }}
            />
          )}
        </div>
      </>
    );
  }
);