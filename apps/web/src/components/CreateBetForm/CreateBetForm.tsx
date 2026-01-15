// src/components/CreateBetForm/CreateBetForm.tsx
import React, { useState, useRef, ChangeEvent, FormEvent, useEffect } from 'react';
import './CreateBetForm.css';
import { apiFetch } from '../../utils/apiFetch';
import { useBetMasterContract } from '../../hooks/useBetMasterContract';
import { useBetContract } from '../../hooks/useBetContract';

interface Props {
  onClose: () => void;
  onCreated: () => void;
  onOpen: () => void;
}

const errorMessages: Record<string, string> = {
  'opponent not found': 'Пользователь с таким username не зарегистрирован в приложении',
  'amount too less': 'Ваш оппонент не готов на такую ставку, попробуйте её увеличить',
  'opponent not ready': 'Ваш оппонент сейчас не готов участвовать в пари',
  'internal server error': 'На сервере произошла непредвиденная ошибка, попробуйте чуть позже',
};

export const CreateBetForm: React.FC<Props> = ({ onClose, onCreated, onOpen }) => {
  useEffect(() => {
    onOpen();
  }, [onOpen]);

  const [isOpen, setIsOpen] = useState(false);
  const [isClosing, setIsClosing] = useState(false);
  const closeTimeoutRef = useRef<number | null>(null);
  const hasClosedRef = useRef(false);

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

  const { getAddress } = useBetContract();
  const { createBetWithDeposit } = useBetMasterContract();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [opponent, setOpponent] = useState('');
  const [amount, setAmount] = useState<number>(0);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const generateBetId = (): bigint => {
    const uuid = crypto?.randomUUID?.();
    if (!uuid) throw new Error('crypto.randomUUID not available');

    return BigInt(`0x${uuid.replace(/-/g, '')}`);
  };

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null;
    setFile(f);
    if (f) {
      setPreview(URL.createObjectURL(f));
    } else {
      setPreview(null);
    }
  };

  const removePhoto = () => {
    setFile(null);
    if (preview) {
      URL.revokeObjectURL(preview);
      setPreview(null);
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    setSuccess(false);

    let betID: bigint;
    let betAddress: string;
    try {
      betID = generateBetId();
    } catch {
      setError('Не удалось сгенерировать ID пари');
      setSubmitting(false);
      return;
    }
    try {
      betAddress = (await getAddress(betID)).toString();
    } catch {
      setError('Не удалось вычислить адрес контракта пари');
      setSubmitting(false);
      return;
    }

    try {
      await createBetWithDeposit(betID, amount.toString());
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (/rejected|declined|cancel/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else {
        setError('Не удалось развернуть контракт и внести депозит');
      }
      console.log('failed to deposit %s', msg);
      setSubmitting(false);
      return;
    }

    const form = new FormData();
    form.append('title', title);
    form.append('description', description);
    form.append('opponent', opponent);
    form.append('amount', amount.toString());
    form.append('contractAddress', betAddress);
    if (file) form.append('image', file);

    try {
      const res = await apiFetch('/api/v1/disputes', {
        method: 'POST',
        body: form,
      });

      if (!res.ok) {
        let serverError = `Ошибка ${res.status}`;
        try {
          const errPayload = await res.json();
          if (typeof errPayload.error === 'string') serverError = errPayload.error;
        } catch { } // ignore JSON parse errors
        throw new Error(serverError);
      }
    } catch (err: any) {
      const key = Object.keys(errorMessages).find(k => err.message.includes(k)) ?? err.message;
      setError(errorMessages[key] || `Не удалось создать пари: ${key}`);
      return;
    } finally {
      setSubmitting(false);
    }

    // Успех: показываем сообщение, оставляем форму открытой
    setSuccess(true);
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

  return (
    <div className={`overlay${isOpen ? ' open' : ''}`}>
      <form
        className={`form-card${isOpen ? ' open' : ''}`}
        onSubmit={handleSubmit}
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
        <h3>Новое пари</h3>

        {error && <div className="error-message">{error}</div>}

        {success ? (
          <div className="success-message">
            🎉 Пари успешно создано!
            <button
              type="button"
              className="btn-close-success"
              onClick={() => {
                onCreated();
                requestClose();
              }}
            >
              Закрыть
            </button>
          </div>
        ) : (
          <>
            <label>
              Название:
              <input
                type="text"
                value={title}
                onChange={e => setTitle(e.target.value)}
                required
              />
            </label>

            <label>
              Описание:
              <textarea
                value={description}
                onChange={e => setDescription(e.target.value)}
                required
              />
            </label>

            <label>
              Оппонент (username):
              <input
                type="text"
                value={opponent}
                onChange={e => setOpponent(e.target.value)}
                placeholder="username"
                required
              />
            </label>

            <label>
              Ставка (TON):
              <input
                type="number"
                step="0.01"
                value={amount || ''}
                onChange={e => setAmount(parseFloat(e.target.value))}
                required
              />
            </label>

            <label>
              Фото (необязательно):
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                onChange={handleFileChange}
              />
            </label>

            {preview && (
              <div className="image-preview">
                <button
                  type="button"
                  className="remove-photo-btn"
                  onClick={removePhoto}
                  title="Удалить фото"
                >
                  ×
                </button>
                <img src={preview} alt="Превью" />
              </div>
            )}

            <div className="form-actions">
              <button
                type="button"
                className="btn-cancel"
                onClick={requestClose}
                disabled={submitting || isClosing}
              >
                Отмена
              </button>
              <button
                type="submit"
                className="btn-submit"
                disabled={submitting || isClosing}
              >
                {submitting ? 'Отправка…' : 'Вызвать'}
              </button>
            </div>
          </>
        )}
      </form>
    </div>
  );
};
