// src/components/Bets/BetDetailsModal.tsx
import React, { useState, useEffect, ChangeEvent, FormEvent, useRef } from 'react';
import { createPortal } from 'react-dom';
import { apiFetch } from '../../utils/apiFetch';
import './BetDetailsModal.css';
import { useBetContract } from '../../hooks/useBetContract';
import { Spinner } from '@telegram-apps/telegram-ui';

interface Props {
  id: string;
  onClose: () => void;
  onCompleted: () => void;
  showActions?: boolean;
  showResultActions?: boolean;
  showClaimActions?: boolean;
}

interface BetDetail {
  id: string;
  title: string;
  description: string;
  amount: number;
  opponent: string;
  cryptocurrency: string;
  contractAddress: string;
  createdAt: string;
  updatedAt: string;
  imageData?: string;
  imageType?: string;
  result:
  | 'evidence'
  | 'sent'
  | 'answered'
  | 'inspected'
  | 'win'
  | 'lose'
  | 'draw'
  | 'rejected'
  | 'processed';
  claim: boolean;
  vote?: boolean; // true = win, false = lose
}

export const BetDetailsModal: React.FC<Props> = ({
  id,
  onClose,
  onCompleted,
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
  const [evidenceForm, setEvidenceForm] = useState(false);
  const [evidenceText, setEvidenceText] = useState('');
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { accept, refund, win, draw } = useBetContract();

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
    setActionLoading(true);
    setError(null);

    try {
      if (action === 'accept') {
        await accept(bet.contractAddress, bet.amount.toString());
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

  // Отправка доказательств
  const handleEvidenceSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!bet) return;
    setActionLoading(true);
    try {
      const form = new FormData();
      form.append('description', evidenceText);
      if (evidenceFile) form.append('evidence', evidenceFile);
      const res = await apiFetch(`/api/v1/disputes/${id}/evidence`, {
        method: 'POST',
        body: form,
      });
      if (!res.ok) throw new Error();
      setSuccess('Доказательства отправлены!');
    } catch {
      // ignore
    } finally {
      setActionLoading(false);
    }
  };

  // Работа с превью файла доказательств
  const handleEvidenceFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null;
    setEvidenceFile(f);
    if (f) setPreview(URL.createObjectURL(f));
    else setPreview(null);
  };
  const removeEvidencePhoto = () => {
    if (preview) URL.revokeObjectURL(preview);
    setEvidenceFile(null);
    setPreview(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
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

  const formatDateUtcPlus3 = (iso: string) => {
    // Парсим ISO-время (UTC)
    const ms = Date.parse(iso);
    // Переводим в UTC: убираем локальное смещение
    const utcMs = ms + new Date(ms).getTimezoneOffset() * 60_000;
    // Добавляем 3 часа
    const targetMs = utcMs + 60_000 / 3;
    const d = new Date(targetMs);
    // Форматируем, например, «дд.мм.гггг, чч:мм:сс»
    return d.toLocaleString('ru-RU', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  // UI состояния успеха
  const modal = success ? (
    <div
      className={`bet-details-overlay${isOpen ? ' open' : ''}`}
      onClick={requestClose}
    >
      <div
        className={`detail-card bet-details-card${isOpen ? ' open' : ''}`}
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
        <p className="success-message">{success}</p>
        <button
          className="btn-close-success"
          onClick={() => {
            onCompleted();
            requestClose();
          }}
        >
          Закрыть
        </button>
      </div>
    </div>
  ) : (
    <div
      className={`bet-details-overlay${isOpen ? ' open' : ''}`}
      onClick={requestClose}
    >
      <div
        className={`detail-card bet-details-card${isOpen ? ' open' : ''}`}
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
           <Spinner size="m" className="spinner"/>
          </div>
        )}
        {error && <p className="error-message">{error}</p>}

        {/* Просмотр деталей */}
        {!loading && bet && !evidenceForm && (
          <>
            <button className="close-btn" onClick={requestClose}>×</button>
            <h3>{bet.title}</h3>
            <p><strong>Оппонент:</strong> {bet.opponent}</p>
            <p><strong>Ставка:</strong> {bet.amount} {bet.cryptocurrency}</p>
            <p><strong>Создано:</strong> {formatDateUtcPlus3(bet.createdAt)}</p>
            <p><strong>Описание:</strong></p>
            <p>{bet.description}</p>
            {bet.imageData && (
              <img
                src={`data:${bet.imageType};base64,${bet.imageData}`}
                alt="Фото пари"
                className="detail-image"
              />
            )}

            {/* Кнопки для вкладки «Новые» */}
            {showActions && bet.result === 'new' && (
              <div className="action-buttons">
                <button
                  className="btn-reject"
                  disabled={actionLoading}
                  onClick={() => handleAction('reject')}
                >
                  {actionLoading ? '…' : 'Отклонить'}
                </button>
                <button
                  className="btn-accept"
                  disabled={actionLoading}
                  onClick={() => handleAction('accept')}
                >
                  {actionLoading ? '…' : 'Принять'}
                </button>
              </div>
            )}

            {/* Голосование за результат (processed) */}
            {showResultActions && bet.result === 'processed' && (
              <div className="action-buttons">
                <button
                  className="btn-reject"
                  disabled={actionLoading}
                  onClick={() => handleResultVote('lose')}
                >
                  {actionLoading ? '…' : 'Поражение'}
                </button>
                <button
                  className="btn-accept"
                  disabled={actionLoading}
                  onClick={() => handleResultVote('win')}
                >
                  {actionLoading ? '…' : 'Победа'}
                </button>
              </div>
            )}

            {/* Индикация вашего выбора (answered) */}
            {bet.result === 'answered' && bet.vote !== undefined && (
              <p className="vote-info">
                Вы выбрали результат{' '}
                {bet.vote
                  ? <span style={{ color: 'green' }}>Победа</span>
                  : <span style={{ color: 'red' }}>Поражение</span>}
              </p>
            )}

            {/* Кнопка «Внести доказательства» (evidence) */}
            {bet.result === 'evidence' && (
              <div className="action-buttons">
                <button
                  className="btn-submit"
                  onClick={() => setEvidenceForm(true)}
                >
                  Внести доказательства
                </button>
              </div>
            )}

            {/* Кнопка «Забрать награду/Вернуть средства» (passed) */}
            {showClaimActions && bet.claim && (
              <div className="action-buttons">
                <button
                  className="btn-accept"
                  disabled={actionLoading}
                  onClick={handleClaim}
                >
                  {actionLoading ? '…' : (bet.result === 'win' ? 'Забрать награду' : 'Вернуть средства')}
                </button>
              </div>
            )}
          </>
        )}

        {/* Форма доказательств */}
        {!loading && bet && evidenceForm && (
          <>
            <div className="header-with-back">
              <button
                type="button"
                className="back-btn"
                onClick={() => setEvidenceForm(false)}
              >
                ←
              </button>
              <h3>Внесите доказательства</h3>
            </div>

            <form onSubmit={handleEvidenceSubmit}>
              <label>
                Описание:
                <textarea
                  value={evidenceText}
                  onChange={e => setEvidenceText(e.target.value)}
                  required
                />
              </label>

              <label>
                Фото (необязательно):
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  onChange={handleEvidenceFileChange}
                />
              </label>

              {preview && (
                <div className="image-preview">
                  <button
                    type="button"
                    className="remove-photo-btn"
                    onClick={removeEvidencePhoto}
                  >
                    ×
                  </button>
                  <img src={preview} alt="Превью" />
                </div>
              )}

              <div className="form-actions">
                <button
                  type="submit"
                  className="btn-submit"
                  disabled={actionLoading}
                >
                  {actionLoading ? 'Отправка…' : 'Отправить'}
                </button>
              </div>
            </form>
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
