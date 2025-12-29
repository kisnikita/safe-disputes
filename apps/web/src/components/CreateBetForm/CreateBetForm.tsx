// src/components/CreateBetForm/CreateBetForm.tsx
import React, { useState, useRef, ChangeEvent, FormEvent, useEffect } from 'react';
import './CreateBetForm.css';
import { apiFetch } from '../../utils/apiFetch';
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

  const { deposit } = useBetContract();

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

    try {
      await deposit(amount.toString());
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (/rejected|declined|cancel/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else {
        setError('Не удалось отправить транзакцию');
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
    if (file) form.append('image', file);

    try {
      const res = await apiFetch('/api/v1/disputes', {
        method: 'POST',
        body: form,
      });

      if (!res.ok) {
        let serverError: string;
        try {
          const errPayload = await res.json();
          serverError = typeof errPayload.error === 'string'
            ? errPayload.error
            : `Ошибка ${res.status}`;
        } catch {
          serverError = `Ошибка ${res.status}`;
        }
        throw new Error(serverError);
      }

      // Успех: показываем сообщение, оставляем форму открытой
      setSuccess(true);
    } catch (err: any) {
      const key = Object.keys(errorMessages)
        .find(k => err.message.includes(k))
        ?? err.message;
      const userMsg = errorMessages[key] || `Не удалось создать пари: ${key}`;
      setError(userMsg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="overlay">
      <form className="form-card" onSubmit={handleSubmit}>
        <h3>Новое пари</h3>

        {error && <div className="error-message">{error}</div>}

        {success ? (
          <div className="success-message">
            🎉 Пари успешно создано!
            <button
              type="button"
              className="btn-close-success"
              onClick={() => {
                onClose();
                onCreated();
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
                onClick={onClose}
                disabled={submitting}
              >
                Отмена
              </button>
              <button
                type="submit"
                className="btn-submit"
                disabled={submitting}
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
