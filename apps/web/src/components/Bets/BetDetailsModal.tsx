// src/components/Bets/BetDetailsModal.tsx
import React, { useState, useEffect, ChangeEvent, FormEvent, useRef } from 'react';
import { apiFetch } from '../../utils/apiFetch';
import './BetDetailsModal.css';

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
  const [bet, setBet] = useState<BetDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const [evidenceForm, setEvidenceForm] = useState(false);
  const [evidenceText, setEvidenceText] = useState('');
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

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

  // Забрать награду
  const handleClaim = async () => {
    if (!bet) return;
    setActionLoading(true);
    try {
      const res = await apiFetch(`/api/v1/disputes/${id}/claim`, { method: 'POST' });
      if (!res.ok) throw new Error();
      setSuccess('Награда успешно получена!');
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

  // UI состояния успеха
  if (success) {
    return (
      <div className="overlay">
        <div className="detail-card">
          <p className="success-message">{success}</p>
          <button
            className="btn-close-success"
            onClick={() => {
              onClose();
              onCompleted();
            }}
          >
            Закрыть
          </button>
        </div>
      </div>
    );
    
  }

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

  

  // Основной рендер
  return (
    <div className="overlay">
      <div className="detail-card">
        {loading && <p>Загрузка…</p>}

        {/* Просмотр деталей */}
        {!loading && bet && !evidenceForm && (
          <>
            <button className="close-btn" onClick={onClose}>×</button>
            <h3>{bet.title}</h3>
            <p><strong>Оппонент:</strong> {bet.opponent}</p>
            <p><strong>Ставка:</strong> {bet.amount} {bet.cryptocurrency}</p>
            <p><strong>Создано (UTC+3):</strong> {formatDateUtcPlus3(bet.createdAt)}</p>
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
            {showActions && (
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
              <button
                className="btn-submit"
                onClick={() => setEvidenceForm(true)}
              >
                Внести доказательства
              </button>
            )}

            {/* Кнопка «Забрать награду» (passed) */}
            {showClaimActions && bet.claim && (
              <div className="action-buttons">
                <button
                  className="btn-accept"
                  disabled={actionLoading}
                  onClick={handleClaim}
                >
                  {actionLoading ? '…' : 'Забрать награду'}
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
};
