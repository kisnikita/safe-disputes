// src/components/Bets/BetsSection.tsx
import React, {
  useState,
  useEffect,
  useRef,
  useCallback,
  forwardRef,
  useImperativeHandle
} from 'react';
import { apiFetch } from '../../utils/apiFetch';
import { BetDetailsModal } from './BetDetailsModal';
import './BetsSection.css';

interface Bet {
  id: string;
  title: string;
  amount: number;
  opponent: string;
  status: string;
  result:
    | 'evidence'
    | 'sent'
    | 'answered'
    | 'inspected'
    | 'win'
    | 'lose'
    | 'draw'
    | 'rejected'
    | 'processed'
    | 'evidence_answered';
  claim: boolean;
}

export interface BetsSectionHandle {
  refresh: () => void;
}

interface Props {
  onModalChange: (open: boolean) => void;
}

export const BetsSection = forwardRef<BetsSectionHandle, Props>(({onModalChange}, ref) => {
  const [subtab, setSubtab] = useState<'current' | 'new' | 'passed'>('current');
  const [bets, setBets] = useState<Bet[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const cursorRef = useRef<string | null>(null);
  const observer = useRef<IntersectionObserver | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const fetchFirstPage = useCallback(async () => {
    cursorRef.current = null;
    setHasMore(true);
    setLoading(true);
    const params = new URLSearchParams({ status: subtab, limit: '10' });
    try {
      const res = await apiFetch(`/api/v1/disputes?${params}`);
      if (!res.ok) throw new Error();
      const { data, nextCursor } = (await res.json()) as {
        data: Bet[];
        nextCursor: string | null;
      };
      setBets(data);
      cursorRef.current = nextCursor;
      setHasMore(!!nextCursor);
    } catch {
      setBets([]);
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
      const res = await apiFetch(`/api/v1/disputes?${params}`);
      if (!res.ok) throw new Error();
      const { data, nextCursor } = (await res.json()) as {
        data: Bet[];
        nextCursor: string | null;
      };
      setBets(prev => [...prev, ...data]);
      cursorRef.current = nextCursor;
      setHasMore(!!nextCursor);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [subtab, loading, hasMore]);

  useEffect(() => {
    fetchFirstPage();
  }, [fetchFirstPage]);

  useImperativeHandle(ref, () => ({ refresh: fetchFirstPage }));

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

  // бейджи для вкладки «Текущие»
  const getCurrentBadge = (result: Bet['result']) => {
    switch (result) {
      case 'evidence':
        return { color: 'red', text: 'Нужны доказательства' };
      case 'sent':
        return { color: 'gray', text: 'Ожидание ответа' };
      case 'answered':
        return { color: 'green', text: 'Результат выбран' };
      case 'inspected':
        return { color: 'yellow', text: 'Расследование' };
      case 'processed':
        return { color: 'yellow', text: 'В процессе' };
      case 'evidence_answered':
        return { color: 'gray', text: 'Ожидание доказательств оппонента' };
      default:
        return null;
    }
  };

  // бейджи для вкладки «Прошедшие»
  const getPassedBadge = (result: Bet['result']) => {
    switch (result) {
      case 'win':
        return { color: 'green', text: 'Победа' };
      case 'lose':
        return { color: 'red', text: 'Поражение' };
      case 'draw':
        return { color: 'yellow', text: 'Ничья' };
      case 'rejected':
        return { color: 'gray', text: 'Отменено' };
      default:
        return null;
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

    return (
      <>
        <div className="bets-section">
          <div className="subtabs">
            {(['current', 'new', 'passed'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => setSubtab(tab)}
                className={subtab === tab ? 'active' : ''}
              >
                {tab === 'current'
                  ? 'Текущие'
                  : tab === 'new'
                  ? 'Новые'
                  : 'Прошедшие'}
              </button>
            ))}
          </div>

          <div className="subcontent">
            {bets.map((bet, idx) => {
              const isLast = idx === bets.length - 1;
              const badge =
                subtab === 'current'
                  ? getCurrentBadge(bet.result)
                  : subtab === 'passed'
                  ? getPassedBadge(bet.result)
                  : null;

              return (
                <div
                  key={bet.id}
                  ref={isLast ? lastRef : null}
                  className="bet-card"
                  onClick={() => openDetails(bet.id)}
                >
                  <h4>{bet.title}</h4>
                  <p>Ставка: {bet.amount} TON</p>
                  <p>Оппонент: {bet.opponent}</p>

                  {badge && (
                    <div
                      className="result-badge"
                      style={{ borderColor: badge.color }}
                    >
                      <span className={`dot ${badge.color}`} />
                      {badge.text}
                    </div>
                  )}

                  {subtab === 'passed' && bet.claim && (
                    <div className="claim-label">Награда доступна</div>
                  )}
                </div>
              );
            })}

            {loading && <div className="loading">Загрузка…</div>}
            {!loading && bets.length === 0 && (
              <div className="empty-message">Тут пока пусто</div>
            )}
          </div>
        </div>

        {!!selectedId && bets && (
          <BetDetailsModal
            id={selectedId}
            onClose={closeDetails}
            onCompleted={() => {
              closeDetails();
              fetchFirstPage();
            }}
            showActions={subtab === 'new'}
            showResultActions={subtab === 'current'}
            showClaimActions={subtab === 'passed'}
          />
        )}
      </>
    );
  }
);
