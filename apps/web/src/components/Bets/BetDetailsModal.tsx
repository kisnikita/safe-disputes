// src/components/Bets/BetDetailsModal.tsx
import React, { useState, useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { apiFetch } from '../../utils/apiFetch';
import './BetDetailsModal.css';
import { useBetContract } from '../../hooks/useBetContract';
import { useTonConnect } from '../../hooks/useTonConnect';
import { Spinner } from '@telegram-apps/telegram-ui';
import { useWalletConnectPopup } from '../../utils/walletPopup';
import { formatNanoToTon } from '../../utils/tonAmount';

interface Props {
  id: string;
  onClose: () => void;
  onCompleted: () => void;
  onOpenEvidence: (disputeId: string) => void;
  showActions?: boolean;
  showResultActions?: boolean;
  showClaimActions?: boolean;
}

interface BetDetail {
  id: string;
  title: string;
  description: string;
  amountNano: string | number;
  opponent: string;
  endsAt: string;
  cryptocurrency: string;
  contractAddress: string;
  createdAt: string;
  updatedAt: string;
  imageData?: string;
  imageType?: string;
  result:
  | 'new'
  | 'sent'
  | 'processed'
  | 'answered'
  | 'evidence'
  | 'evidence_answered'
  | 'inspected'
  | 'rejected'
  | 'win'
  | 'lose'
  | 'draw'
  claim: boolean;
  vote?: boolean; // true = win, false = lose
}

export const BetDetailsModal: React.FC<Props> = ({
  id,
  onClose,
  onCompleted,
  onOpenEvidence,
  showActions,
  showResultActions,
  showClaimActions,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [isClosing, setIsClosing] = useState(false);
  const closeTimeoutRef = useRef<number | null>(null);
  const hasClosedRef = useRef(false);
  const [bet, setBet] = useState<BetDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [acceptShake, setAcceptShake] = useState(false);
  const [claimShake, setClaimShake] = useState(false);
  const [resultShake, setResultShake] = useState<'win' | 'lose' | null>(null);
  const { accept, refund, win, draw } = useBetContract();
  const { connected } = useTonConnect();
  const showWalletConnectPopup = useWalletConnectPopup();

  useEffect(() => {
    const id = requestAnimationFrame(() => setIsOpen(true));
    return () => cancelAnimationFrame(id);
  }, []);
  useEffect(() => {
    return () => {
      if (closeTimeoutRef.current !== null) {
        clearTimeout(closeTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    (async () => {
      try {
        const res = await apiFetch(`/api/v1/disputes/${id}`);
        if (!res.ok) throw new Error();
        const { data } = (await res.json()) as { data: BetDetail };
        setBet(data);
      } catch {
        // ignore
      } finally {
        setLoading(false);
      }
    })();
  }, [id]);

  // Принять/Отклонить
  const handleAction = async (action: 'accept' | 'reject') => {
    if (!bet) return;
    if (action === 'accept' && !connected) {
      setAcceptShake(false);
      requestAnimationFrame(() => setAcceptShake(true));
      await showWalletConnectPopup();
      return;
    }
    setActionLoading(true);
    setError(null);

    try {
      if (action === 'accept') {
        await accept(bet.contractAddress, String(bet.amountNano));
      }
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (/rejected|declined|cancel/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else {
        setError('Не удалось отправить транзакцию');
      }
      console.log('failed to accept %s', msg);
      setActionLoading(false);
      return;
    }
    try {
      const res = await apiFetch(`/api/v1/disputes/${id}/${action}`, { method: 'POST' });
      if (!res.ok) throw new Error();
      setSuccess(action === 'accept' ? 'Пари успешно принято!' : 'Пари отклонено');
    } catch {
      // ignore
    } finally {
      setActionLoading(false);
    }
  };

  // Голосование за результат
  const handleResultVote = async (vote: 'win' | 'lose') => {
    if (!bet) return;
    if (vote === 'win' && !connected) {
      setResultShake(null);
      requestAnimationFrame(() => setResultShake(vote));
      await showWalletConnectPopup();
      return;
    }
    setActionLoading(true);
    try {
      const res = await apiFetch(`/api/v1/disputes/${id}/vote`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ vote: vote === 'win' }),
      });
      if (!res.ok) throw new Error();
      setSuccess(vote === 'win' ? 'Вы проголосовали за победу!' : 'Вы проголосовали за поражение!');
    } catch {
      // ignore
    } finally {
      setActionLoading(false);
    }
  };

  // Забрать награду/Вернуть средства
  const handleClaim = async () => {
    if (!bet) return;
    if (!connected) {
      setClaimShake(false);
      requestAnimationFrame(() => setClaimShake(true));
      await showWalletConnectPopup();
      return;
    }
    setActionLoading(true);
    setError(null);
    try {
      if (bet.result === 'draw') {
        await draw(bet.contractAddress);
      } else if (bet.result === 'win') {
        await win(bet.contractAddress);
      } else if (bet.result === 'rejected') {
        await refund(bet.contractAddress);
      } else {
        throw new Error('Unsupported bet result for claim');
      }
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (/rejected|declined|cancel/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else {
        setError('Не удалось отправить транзакцию');
      }
      console.log('failed to claim bet with result %s %s', bet.result, msg);
      setActionLoading(false);
      return;
    }
    try {
      const res = await apiFetch(`/api/v1/disputes/${id}/claim`, { method: 'POST' });
      if (!res.ok) throw new Error();
      setSuccess((bet.result === 'win' ? 'Награда успешно получена!' : 'Средства успешно возвращены!'));
    } catch {
      // ignore
    } finally {
      setActionLoading(false);
    }
  };

  const requestClose = () => {
    if (isClosing) return;
    setIsClosing(true);
    setIsOpen(false);
    hasClosedRef.current = false;
    closeTimeoutRef.current = window.setTimeout(() => {
      if (hasClosedRef.current) return;
      hasClosedRef.current = true;
      onClose();
    }, 100);
  };

  const formatDate = (iso: string) =>
    new Date(iso).toLocaleString('ru-RU', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
  });

  // UI состояния успеха
  const modal = success ? (
    <div
      className={`bet-details-overlay${isOpen ? ' open' : ''}`}
      onClick={requestClose}
    >
      <div
        className={`bet-details-detail-card bet-details-card bet-details-success-card${isOpen ? ' open' : ''}`}
        onClick={e => e.stopPropagation()}
        onTransitionEnd={e => {
          if (!isClosing) return;
          if (e.propertyName !== 'opacity') return;
          if (e.currentTarget !== e.target) return;
          if (closeTimeoutRef.current !== null) {
            clearTimeout(closeTimeoutRef.current);
          }
          if (hasClosedRef.current) return;
          hasClosedRef.current = true;
          onClose();
        }}
      >
        <div className="bet-details-success-layout">
          <p className="bet-details-success-message">{success}</p>
          <div className="bet-details-success-actions">
            <button
              className="bet-details-close-success-btn"
              onClick={() => {
                onCompleted();
                requestClose();
              }}
            >
              Закрыть
            </button>
          </div>
        </div>
      </div>
    </div>
  ) : (
    <div
      className={`bet-details-overlay${isOpen ? ' open' : ''}`}
      onClick={requestClose}
    >
      <div
        className={`bet-details-detail-card bet-details-card${isOpen ? ' open' : ''}`}
        onClick={e => e.stopPropagation()}
        onTransitionEnd={e => {
          if (!isClosing) return;
          if (e.propertyName !== 'opacity') return;
          if (e.currentTarget !== e.target) return;
          if (closeTimeoutRef.current !== null) {
            clearTimeout(closeTimeoutRef.current);
          }
          if (hasClosedRef.current) return;
          hasClosedRef.current = true;
          onClose();
        }}
      >
        {loading && (
          <div className="loading">
            <Spinner size="m" className="spinner" />
          </div>
        )}
        {error && <p className="bet-details-error-message">{error}</p>}

        {/* Просмотр деталей */}
        {!loading && bet && (
          <>
            <button className="bet-details-close-btn" onClick={requestClose}>×</button>
            <h3>{bet.title}</h3>
            <p><strong>Оппонент:</strong> {bet.opponent}</p>
            <p><strong>Ставка:</strong> {formatNanoToTon(bet.amountNano, 2, { keepTrailingZeros: true })} {bet.cryptocurrency}</p>
            <p><strong>Создано:</strong> {formatDate(bet.createdAt)}</p>
            {(bet.result === 'new' || bet.result === 'sent') && (
              <p><strong>Окончание пари:</strong> {formatDate(bet.endsAt)}</p>
            )}
            <p className="bet-details-description-label"><strong>Условия:</strong></p>
            <p className="bet-details-description-text">{bet.description}</p>
            {bet.imageData && (
              <img
                src={`data:${bet.imageType};base64,${bet.imageData}`}
                alt="Фото пари"
                className="bet-details-image"
              />
            )}

            {/* Кнопки для вкладки «Новые» */}
            {showActions && bet.result === 'new' && (
              <div className="bet-details-action-buttons">
                <button
                  className="bet-details-reject-btn"
                  disabled={actionLoading}
                  onClick={() => handleAction('reject')}
                >
                  {actionLoading ? '…' : 'Отклонить'}
                </button>
                <button
                  className={`bet-details-accept-btn${!connected ? ' bet-details-wallet-disconnected' : ''}${acceptShake ? ' bet-details-wallet-shake' : ''}`}
                  disabled={actionLoading}
                  onClick={() => handleAction('accept')}
                  onAnimationEnd={() => setAcceptShake(false)}
                  title={connected ? undefined : 'Подключите TON-кошелёк'}
                >
                  {actionLoading ? '…' : 'Принять'}
                </button>
              </div>
            )}

            {/* Голосование за результат (processed) */}
            {showResultActions && bet.result === 'processed' && (
              <div className="bet-details-action-buttons">
                <button
                  className={`bet-details-reject-btn${resultShake === 'lose' ? ' bet-details-wallet-shake' : ''}`}
                  disabled={actionLoading}
                  onClick={() => handleResultVote('lose')}
                  onAnimationEnd={() => setResultShake(null)}
                >
                  {actionLoading ? '…' : 'Поражение'}
                </button>
                <button
                  className={`bet-details-accept-btn${!connected ? ' bet-details-wallet-disconnected' : ''}${resultShake === 'win' ? ' bet-details-wallet-shake' : ''}`}
                  disabled={actionLoading}
                  onClick={() => handleResultVote('win')}
                  onAnimationEnd={() => setResultShake(null)}
                  title={connected ? undefined : 'Подключите TON-кошелёк'}
                >
                  {actionLoading ? '…' : 'Победа'}
                </button>
              </div>
            )}

            {/* Индикация вашего выбора (answered) */}
            {bet.result === 'answered' && bet.vote !== undefined && (
              <p className="bet-details-vote-info">
                Вы выбрали результат{' '}
                {bet.vote
                  ? <span style={{ color: 'green' }}>Победа</span>
                  : <span style={{ color: 'red' }}>Поражение</span>}
              </p>
            )}

            {/* Кнопка «Внести доказательства» (evidence) */}
            {bet.result === 'evidence' && (
              <div className="bet-details-action-buttons">
                <button
                  className="bet-details-submit-btn"
                  onClick={() => onOpenEvidence(id)}
                >
                  Внести доказательства
                </button>
              </div>
            )}

            {/* Кнопка «Забрать награду/Вернуть средства» (passed) */}
            {showClaimActions && bet.claim && (
              <div className="bet-details-action-buttons">
                <button
                  className={`bet-details-accept-btn${!connected ? ' bet-details-wallet-disconnected' : ''}${claimShake ? ' bet-details-wallet-shake' : ''}`}
                  disabled={actionLoading}
                  onClick={handleClaim}
                  onAnimationEnd={() => setClaimShake(false)}
                  title={connected ? undefined : 'Подключите TON-кошелёк'}
                >
                  {actionLoading ? '…' : (bet.result === 'win' ? 'Забрать награду' : 'Вернуть средства')}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );

  if (typeof document === 'undefined') {
    return modal;
  }

  // Основной рендер
  return createPortal(modal, document.body);
};
