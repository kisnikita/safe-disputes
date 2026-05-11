// src/components/Bets/BetDetailsModal.tsx
import React, { useState, useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { apiFetch } from '../../utils/apiFetch';
import './BetDetailsModal.css';
import { useBetContract } from '../../hooks/useBetContract';
import { useTonConnect } from '../../hooks/useTonConnect';
import { Spinner } from '@telegram-apps/telegram-ui';
import { ImageViewerModal } from '../ImageViewer/ImageViewerModal';
import { formatNanoToTon } from '../../utils/tonAmount';
import { useWalletConnectPopup } from '../../utils/walletPopup';

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
  depositNano: string | number;
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
  isClaimable: boolean;
  isWin?: boolean;
}

const claimButtonTitle: Partial<Record<BetDetail['result'], string>> = {
  rejected: 'Вернуть ставку и депозит',
  win:      'Забрать награду',
  lose:     'Вернуть депозит',
  draw:     'Вернуть ставку и депозит',
};

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
  const hasCompletedRef = useRef(false);
  const [bet, setBet] = useState<BetDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [acceptShake, setAcceptShake] = useState(false);
  const [claimShake, setClaimShake] = useState(false);
  const [resultShake, setResultShake] = useState<'win' | 'lose' | null>(null);
  const [previewImageSrc, setPreviewImageSrc] = useState<string | null>(null);
  const { accept, vote, claim, cancel } = useBetContract();
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
        const signedBoc = await accept(bet.contractAddress);
        const params = new URLSearchParams({ boc: signedBoc });
        const res = await apiFetch(`/api/v1/disputes/${id}/${action}?${params.toString()}`, { method: 'POST' });
        if (!res.ok) throw new Error();
        setSuccess('Пари успешно принято!');
      } else {
        const res = await apiFetch(`/api/v1/disputes/${id}/${action}`, { method: 'POST' });
        if (!res.ok) throw new Error();
        setSuccess('Пари отменено');
      }
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (action === 'accept' && /rejected|declined|cancel/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else {
        setError(action === 'accept' ? 'Не удалось отправить транзакцию' : 'Не удалось отменить пари');
      }
      console.log('failed to handle bet action %s, msg = %s, betContract = %s', action, msg, bet.contractAddress);
      setActionLoading(false);
      return;
    } finally {
      setActionLoading(false);
    }
  };

  const handleResultVote = async (v: 'win' | 'lose') => {
    if (!bet) return;
    if (!connected) {
      setResultShake(null);
      requestAnimationFrame(() => setResultShake(v));
      await showWalletConnectPopup();
      return;
    }
    setActionLoading(true);
    try {
      const signedBoc = await vote(bet.contractAddress, v === 'win');
      const res = await apiFetch(`/api/v1/disputes/${id}/vote`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ vote: v === 'win', boc: signedBoc }),
      });
      if (!res.ok) throw new Error();
      setSuccess(v === 'win' ? 'Вы проголосовали за победу!' : 'Вы проголосовали за поражение!');
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (/rejected|declined|cancel/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else {
        setError('Не удалось отправить транзакцию');
      }
      console.log('failed to vote bet with vote: %s, msg: %s, betContract = %s', v, msg, bet.contractAddress);
    } finally {
      setActionLoading(false);
    }
  };

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
      const signedBoc = bet.result === 'rejected'
        ? await cancel(bet.contractAddress)
        : await claim(bet.contractAddress);
      const params = new URLSearchParams({ boc: signedBoc });
      const res = await apiFetch(`/api/v1/disputes/${id}/claim?${params.toString()}`, { method: 'POST' });
      if (!res.ok) throw new Error();
      setSuccess('Средства успешно возвращены!');
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (/rejected|declined|cancel/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else {
        setError('Не удалось отправить транзакцию');
      }
      console.log('failed to claim with result: %s, msg: %s, betContract = %s', bet.result, msg, bet.contractAddress);
      setActionLoading(false);
      return;
    } finally {
      setActionLoading(false);
    }
  };

  const requestClose = () => {
    if (isClosing) return;
    if (success && !hasCompletedRef.current) {
      hasCompletedRef.current = true;
      onCompleted();
    }
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

  const formatAmountWithDeposit = (amountNano: string | number, depositNano: string | number, currency: string): string => {
    const amountTon = formatNanoToTon(amountNano, 2, { keepTrailingZeros: true });
    const depositTon = formatNanoToTon(depositNano, 2);
    return `${amountTon} ${currency} (+${depositTon} ${currency} депозит)`;
  };

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
            <p><strong>Ставка:</strong> {formatAmountWithDeposit(bet.amountNano, bet.depositNano, bet.cryptocurrency)}</p>
            <p><strong>Создано:</strong> {formatDate(bet.createdAt)}</p>
            {(bet.result === 'new' || bet.result === 'sent') && (
              <p><strong>Окончание пари:</strong> {formatDate(bet.endsAt)}</p>
            )}
            <p className="bet-details-description-label"><strong>Условия:</strong></p>
            <p className="bet-details-description-text">{bet.description}</p>
            {bet.imageData && (
              <button
                type="button"
                className="bet-details-image-trigger"
                onClick={() => setPreviewImageSrc(`data:${bet.imageType};base64,${bet.imageData}`)}
                aria-label="Открыть фото пари"
              >
                <img
                  src={`data:${bet.imageType};base64,${bet.imageData}`}
                  alt="Фото пари"
                  className="bet-details-image"
                />
              </button>
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
            {showActions && bet.result === 'sent' && (
              <div className="bet-details-action-buttons">
                <button
                  className="bet-details-reject-btn"
                  disabled={actionLoading}
                  onClick={() => handleAction('reject')}
                >
                  {actionLoading ? '…' : 'Отменить пари'}
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
            {bet.result === 'answered' && bet.isWin !== undefined && (
              <p className="bet-details-vote-info">
                Вы выбрали результат{' '}
                {bet.isWin
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

            {/* Кнопка «Вернуть депозит (passed) */}
            {showClaimActions && bet.isClaimable && (
              <div className="bet-details-action-buttons">
                <button
                  className={`bet-details-accept-btn${!connected ? ' bet-details-wallet-disconnected' : ''}${claimShake ? ' bet-details-wallet-shake' : ''}`}
                  disabled={actionLoading}
                  onClick={handleClaim}
                  onAnimationEnd={() => setClaimShake(false)}
                  title={connected ? undefined : 'Подключите TON-кошелёк'}
                >
                  {actionLoading ? '…' : claimButtonTitle[bet.result]}
                </button>
              </div>
            )}
          </>
        )}
      </div>
      <ImageViewerModal
        isOpen={previewImageSrc !== null}
        src={previewImageSrc}
        alt="Фото пари"
        onClose={() => setPreviewImageSrc(null)}
      />
    </div>
  );

  if (typeof document === 'undefined') {
    return modal;
  }

  // Основной рендер
  return createPortal(modal, document.body);
};
