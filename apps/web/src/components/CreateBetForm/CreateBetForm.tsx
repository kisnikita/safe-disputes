import React, { useState, useRef, FormEvent, useEffect, useCallback } from 'react';
import './CreateBetForm.css';
import { apiFetch } from '../../utils/apiFetch';
import { useBetMasterContract } from '../../hooks/useBetMasterContract';
import { useBetContract } from '../../hooks/useBetContract';
import { useTonConnect } from '../../hooks/useTonConnect';
import { FileInput } from '../FileInput/FileInput';
import { TimePicker } from '../TimePicker/TimePicker';
import { backButton, hideKeyboard, popup } from '@tma.js/sdk-react';

interface Props {
  onClose: () => void;
  onCreated: () => void;
}

export interface CreateBetDraft {
  title: string;
  description: string;
  opponent: string;
  amount: number;
  endsAtISO: string;
}

const errorMessages: Record<string, string> = {
  'opponent not found': 'Пользователь с таким username не зарегистрирован в приложении',
  'creator and opponent must be different': 'Нельзя создать пари с самим собой',
  'amount too less': 'Ваш оппонент не готов на такую ставку, попробуйте её увеличить',
  'opponent not ready': 'Ваш оппонент сейчас не готов участвовать в пари',
  'invalid transaction boc': 'Не удалось обработать подписанную транзакцию',
  'transaction monitor unavailable': 'Сервис проверки блокчейна временно недоступен',
  'transaction not finalized in time': 'Транзакция пока не подтверждена, попробуйте ещё раз',
  'transaction failed': 'Транзакция завершилась с ошибкой',
  'internal server error': 'На сервере произошла непредвиденная ошибка, попробуйте чуть позже',
};

const CREATE_BET_FILE_INPUT_MAX_FILES = 1;
const DESCRIPTION_MIN_HEIGHT_PX = 80;
const CREATE_BET_DRAFT_KEY = 'create-bet-draft-v1';

const roundToMinute = (value: Date): Date => {
  const next = new Date(value);
  next.setSeconds(0, 0);
  return next;
};

const getDefaultEndsAt = (): Date => {
  const next = new Date();
  next.setHours(next.getHours() + 24);
  return roundToMinute(next);
};

const getMinAllowedEndsAt = (): Date => {
  const now = new Date();
  const next = new Date(now);
  if (now.getSeconds() > 0 || now.getMilliseconds() > 0) {
    next.setMinutes(next.getMinutes() + 1);
  }
  next.setSeconds(0, 0);
  return next;
};

const isSameDay = (left: Date, right: Date): boolean => left.getFullYear() === right.getFullYear()
  && left.getMonth() === right.getMonth()
  && left.getDate() === right.getDate();

const formatTimeInputValue = (value: Date): string => {
  const hours = `${value.getHours()}`.padStart(2, '0');
  const minutes = `${value.getMinutes()}`.padStart(2, '0');
  return `${hours}:${minutes}`;
};

const formatShortDuration = (durationMs: number): string => {
  const totalMinutes = Math.max(0, Math.floor(durationMs / 60000));
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (minutes === 0) return `${hours} ч`;
  return `${hours} ч ${minutes} мин`;
};

const formatDateInputValue = (value: Date): string => {
  const year = value.getFullYear();
  const month = `${value.getMonth() + 1}`.padStart(2, '0');
  const day = `${value.getDate()}`.padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const isFromDescriptionTextarea = (target: EventTarget | null): boolean => {
  if (!(target instanceof HTMLElement)) return false;
  return target.closest('.create-bet-textarea') !== null;
};

const parseCreateBetDraft = (raw: string | null): CreateBetDraft | null => {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<CreateBetDraft>;
    if (typeof parsed.title !== 'string') return null;
    if (typeof parsed.description !== 'string') return null;
    if (typeof parsed.opponent !== 'string') return null;
    if (typeof parsed.amount !== 'number' || Number.isNaN(parsed.amount)) return null;
    if (typeof parsed.endsAtISO !== 'string') return null;
    const parsedDate = new Date(parsed.endsAtISO);
    if (Number.isNaN(parsedDate.getTime())) return null;
    return {
      title: parsed.title,
      description: parsed.description,
      opponent: parsed.opponent,
      amount: parsed.amount,
      endsAtISO: parsed.endsAtISO,
    };
  } catch {
    return null;
  }
};

const getSessionStorage = (): Storage | null => {
  if (typeof window === 'undefined') return null;
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
};

export const CreateBetForm: React.FC<Props> = ({ onClose, onCreated }) => {
  const screenRef = useRef<HTMLDivElement | null>(null);
  const descriptionRef = useRef<HTMLTextAreaElement | null>(null);
  const draftLoadedRef = useRef(false);
  const hasUserInputRef = useRef(false);
  const createdRef = useRef(false);
  const createdNotifiedRef = useRef(false);
  const attemptCloseRef = useRef<() => Promise<void>>(async () => {});
  const closeInFlightRef = useRef(false);
  const touchStartYRef = useRef<number | null>(null);
  const touchStartedAtTopRef = useRef(false);
  const touchHideTriggeredRef = useRef(false);
  const initialEndsAtRef = useRef<Date>(getDefaultEndsAt());

  const { getAddress } = useBetContract();
  const { createBetWithDeposit } = useBetMasterContract();
  const { connected } = useTonConnect();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [opponent, setOpponent] = useState('');
  const [amount, setAmount] = useState<number>(0);
  const [endsAt, setEndsAt] = useState<Date>(() => getDefaultEndsAt());
  const [file, setFile] = useState<File | null>(null);
  const [fileInputHasError, setFileInputHasError] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitShake, setSubmitShake] = useState(false);

  const notifyCreatedIfNeeded = useCallback(() => {
    if (!createdRef.current || createdNotifiedRef.current) return;
    createdNotifiedRef.current = true;
    onCreated();
  }, [onCreated]);

  const showSaveDraftConfirm = useCallback(async (): Promise<boolean> => {
    if (popup.isSupported()) {
      const buttonId = await popup.show({
        message: 'Сохранить введённые данные перед выходом?',
        buttons: [
          { id: 'yes', type: 'default', text: 'Да' },
          { id: 'no', type: 'default', text: 'Нет' },
        ],
      });
      return buttonId === 'yes';
    }

    return window.confirm('Сохранить введённые данные перед выходом?');
  }, []);

  const getCurrentDraft = useCallback((): CreateBetDraft => ({
    title,
    description,
    opponent,
    amount,
    endsAtISO: endsAt.toISOString(),
  }), [title, description, opponent, amount, endsAt]);

  const saveDraftToStorage = useCallback((draft: CreateBetDraft) => {
    const storage = getSessionStorage();
    if (!storage) return;
    storage.setItem(CREATE_BET_DRAFT_KEY, JSON.stringify(draft));
  }, []);

  const clearDraftFromStorage = useCallback(() => {
    const storage = getSessionStorage();
    if (!storage) return;
    storage.removeItem(CREATE_BET_DRAFT_KEY);
  }, []);

  const hideKeyboardSafe = useCallback(() => {
    if (hideKeyboard.isSupported()) {
      hideKeyboard();
      return;
    }
  }, []);

  const isScreenAtTop = useCallback((screen: HTMLDivElement): boolean => {
    const localAtTop = screen.scrollTop <= 1;
    const root = document.scrollingElement;
    const docAtTop = root ? root.scrollTop <= 1 : true;
    const windowAtTop = window.scrollY <= 1;
    return localAtTop && docAtTop && windowAtTop;
  }, []);

  const hasSavableDraftData = title.trim().length > 0
    || description.trim().length > 0
    || opponent.trim().length > 0
    || amount > 0
    || endsAt.getTime() !== initialEndsAtRef.current.getTime();
  const minAllowedEndsAt = getMinAllowedEndsAt();
  const endsAtMs = endsAt.getTime();
  const minAllowedEndsAtMs = minAllowedEndsAt.getTime();
  const isEndsAtInvalid = endsAtMs < minAllowedEndsAtMs;
  const acceptanceWindowMs = endsAtMs - Date.now();
  const isShortAcceptanceWindow = acceptanceWindowMs > 0 && acceptanceWindowMs < 24 * 60 * 60 * 1000;
  const acceptanceHintText = isEndsAtInvalid
    ? 'Дата и время окончания пари указаны в прошлом'
    : isShortAcceptanceWindow
      ? `Оппоненту на принятие пари останется ${formatShortDuration(acceptanceWindowMs)}`
      : 'У оппонента будет 24 часа на принятие пари';
  const isRequiredFieldsFilled = title.trim().length > 0
    && description.trim().length > 0
    && opponent.trim().length > 0
    && Number.isFinite(amount)
    && amount > 0
    && !isEndsAtInvalid;

  attemptCloseRef.current = async () => {
    if (closeInFlightRef.current || submitting) return;
    closeInFlightRef.current = true;
    try {
      if (success) {
        clearDraftFromStorage();
        notifyCreatedIfNeeded();
        onClose();
        return;
      }
      if (!hasSavableDraftData) {
        clearDraftFromStorage();
        onClose();
        return;
      }
      const shouldSaveDraft = await showSaveDraftConfirm();
      if (shouldSaveDraft) {
        saveDraftToStorage(getCurrentDraft());
      } else {
        clearDraftFromStorage();
      }
      onClose();
    } finally {
      closeInFlightRef.current = false;
    }
  };

  useEffect(() => {
    if (!backButton.isSupported()) return;

    const handleBackClick = () => {
      void attemptCloseRef.current();
    };

    const offClick = backButton.onClick(handleBackClick);

    return () => {
      offClick();
    };
  }, []);

  const handleScreenWheel = useCallback((event: React.WheelEvent<HTMLDivElement>) => {
    if (isFromDescriptionTextarea(event.target)) return;
    if (!isScreenAtTop(event.currentTarget)) return;
    if (event.deltaY < 0) {
      hideKeyboardSafe();
    }
  }, [hideKeyboardSafe, isScreenAtTop]);

  const handleScreenTouchStart = useCallback((event: React.TouchEvent<HTMLDivElement>) => {
    if (isFromDescriptionTextarea(event.target)) return;
    if (event.touches.length !== 1) return;
    touchStartYRef.current = event.touches[0].clientY;
    touchStartedAtTopRef.current = isScreenAtTop(event.currentTarget);
    touchHideTriggeredRef.current = false;
  }, [isScreenAtTop]);

  const handleScreenTouchMove = useCallback((event: React.TouchEvent<HTMLDivElement>) => {
    if (isFromDescriptionTextarea(event.target)) return;
    if (touchHideTriggeredRef.current) return;
    if (!touchStartedAtTopRef.current) return;
    if (event.touches.length !== 1 || touchStartYRef.current === null) return;
    if (!isScreenAtTop(event.currentTarget)) return;

    const deltaY = event.touches[0].clientY - touchStartYRef.current;
    if (deltaY > 50) {
      hideKeyboardSafe();
      touchHideTriggeredRef.current = true;
    }
  }, [hideKeyboardSafe, isScreenAtTop]);

  const resetTouchTracking = useCallback(() => {
    touchStartYRef.current = null;
    touchStartedAtTopRef.current = false;
    touchHideTriggeredRef.current = false;
  }, []);

  const resizeDescription = useCallback(() => {
    const textarea = descriptionRef.current;
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = `${Math.max(textarea.scrollHeight, DESCRIPTION_MIN_HEIGHT_PX)}px`;
  }, []);

  useEffect(() => {
    resizeDescription();
  }, [description, resizeDescription]);

  useEffect(() => {
    document.documentElement.classList.add('create-bet-native-scroll');
    return () => {
      document.documentElement.classList.remove('create-bet-native-scroll');
    };
  }, []);

  useEffect(() => {
    if (draftLoadedRef.current) return;
    draftLoadedRef.current = true;
    const storage = getSessionStorage();
    if (!storage) return;

    const raw = storage.getItem(CREATE_BET_DRAFT_KEY);
    if (hasUserInputRef.current) return;
    const draft = parseCreateBetDraft(raw);
    if (!draft) return;
    setTitle(draft.title);
    setDescription(draft.description);
    setOpponent(draft.opponent);
    setAmount(draft.amount);
    setEndsAt(roundToMinute(new Date(draft.endsAtISO)));
  }, []);

  const extractApiError = async (res: Response): Promise<string> => {
    let serverError = `Ошибка ${res.status}`;
    try {
      const errPayload = await res.json();
      if (typeof errPayload.error === 'string') serverError = errPayload.error;
    } catch {
      // ignore JSON parse errors
    }
    return serverError;
  };

  const generateBetId = (): bigint => {
    const uuid = crypto?.randomUUID?.();
    if (!uuid) throw new Error('crypto.randomUUID not available');

    // Contract expects int128, so keep the value in [1, 2^127 - 1]
    // to avoid intermittent overflow when UUID's highest bit is set.
    const value = BigInt(`0x${uuid.replace(/-/g, '')}`) & ((1n << 127n) - 1n);
    if (value === 0n) {
      return 1n;
    }
    return value;
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    setSuccess(false);

    if (!connected) {
      setError('Подключите TON-кошелёк, чтобы выполнить транзакцию');
      setSubmitShake(false);
      requestAnimationFrame(() => setSubmitShake(true));
      setSubmitting(false);
      return;
    }

    if (!isRequiredFieldsFilled) {
      setError('Заполните все обязательные поля');
      setSubmitting(false);
      return;
    }
    if (isEndsAtInvalid) {
      setError('Дата и время окончания должны быть в будущем');
      setSubmitting(false);
      return;
    }

    try {
      const precheckRes = await apiFetch('/api/v1/disputes/precheck', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          opponent,
          amount,
        }),
      });

      if (!precheckRes.ok) {
        throw new Error(await extractApiError(precheckRes));
      }
    } catch (err: any) {
      const key = Object.keys(errorMessages).find(k => err.message.includes(k)) ?? err.message;
      setError(errorMessages[key] || `Не удалось проверить оппонента: ${key}`);
      setSubmitting(false);
      return;
    }

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
    } catch (err) {
      setError('Не удалось вычислить адрес контракта пари');
      setSubmitting(false);
      return;
    }

    let signedBoc: string;
    try {
      signedBoc = await createBetWithDeposit(betID, amount.toString());
    } catch (err: any) {
      const msg = typeof err?.message === 'string' ? err.message : '';
      if (/rejected|declined|cancel|not sent/i.test(msg)) {
        setError('Транзакция отменена пользователем');
      } else if (/No enough funds/i.test(msg)) {
        setError('Недостаточно средств для внесения депозита');
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
    form.append('endsAt', endsAt.toISOString());
    form.append('contractAddress', betAddress);
    form.append('boc', signedBoc);
    if (file) form.append('image', file);

    try {
      const res = await apiFetch('/api/v1/disputes', {
        method: 'POST',
        body: form,
      });

      if (!res.ok) {
        throw new Error(await extractApiError(res));
      }
    } catch (err: any) {
      const key = Object.keys(errorMessages).find(k => err.message.includes(k)) ?? err.message;
      setError(errorMessages[key] || `Не удалось создать пари: ${key}`);
      return;
    } finally {
      setSubmitting(false);
    }

    // Успех: показываем сообщение, форма закрывается вручную через кнопку/BackButton
    clearDraftFromStorage();
    createdRef.current = true;
    setSuccess(true);
  };

  return (
    <div
      ref={screenRef}
      className="create-bet-screen"
      onWheel={handleScreenWheel}
      onTouchStart={handleScreenTouchStart}
      onTouchMove={handleScreenTouchMove}
      onTouchEnd={resetTouchTracking}
      onTouchCancel={resetTouchTracking}
    >
      <form
        className="create-bet-page"
        onSubmit={handleSubmit}
      >
        <header className="create-bet-header">
          <h3>Новое пари</h3>
        </header>

        {error && <div className="create-bet-error-message">{error}</div>}

        {success ? (
          <div className="create-bet-success-message">
            🎉 Пари успешно создано!
            <button
              type="button"
              className="create-bet-close-success-btn"
              onClick={() => {
                clearDraftFromStorage();
                notifyCreatedIfNeeded();
                onClose();
              }}
            >
              К списку пари
            </button>
          </div>
        ) : (
          <>
            <section className="create-bet-section">
              <div className="create-bet-field-label">
                Название<span className="create-bet-required-mark" aria-hidden="true">*</span>
              </div>
              <div className="create-bet-input-wrap">
                <input
                  className="create-bet-input"
                  type="text"
                  value={title}
                  onChange={e => {
                    hasUserInputRef.current = true;
                    setTitle(e.target.value);
                  }}
                  placeholder="О чём пари?"
                  required
                />
                <button
                  type="button"
                  className={`create-bet-input-clear${title.length > 0 ? ' visible' : ''}`}
                  onMouseDown={event => event.preventDefault()}
                  onClick={() => setTitle('')}
                  aria-label="Очистить название"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <path
                      d="M7 7l10 10M17 7L7 17"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                  </svg>
                </button>
              </div>
              <div className="create-bet-field-label">
                Описание<span className="create-bet-required-mark" aria-hidden="true">*</span>
              </div>
              <div className="create-bet-input-wrap create-bet-textarea-wrap">
                <textarea
                  ref={descriptionRef}
                  className="create-bet-input create-bet-textarea"
                  style={{ minHeight: `${DESCRIPTION_MIN_HEIGHT_PX}px` }}
                  value={description}
                  onChange={event => {
                    hasUserInputRef.current = true;
                    setDescription(event.target.value);
                  }}
                  onInput={resizeDescription}
                  placeholder="Добавьте детали и условия"
                  required
                />
                <button
                  type="button"
                  className={`create-bet-input-clear${description.length > 0 ? ' visible' : ''}`}
                  onMouseDown={event => event.preventDefault()}
                  onClick={() => setDescription('')}
                  aria-label="Очистить описание"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <path
                      d="M7 7l10 10M17 7L7 17"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                  </svg>
                </button>
              </div>
            </section>

            <section className="create-bet-section">
              <div className="create-bet-field-label">
                Оппонент<span className="create-bet-required-mark" aria-hidden="true">*</span>
              </div>
              <div className="create-bet-input-wrap">
                <input
                  className="create-bet-input"
                  type="text"
                  value={opponent}
                  onChange={e => {
                    hasUserInputRef.current = true;
                    setOpponent(e.target.value);
                  }}
                  placeholder="username"
                  required
                />
                <button
                  type="button"
                  className={`create-bet-input-clear${opponent.length > 0 ? ' visible' : ''}`}
                  onMouseDown={event => event.preventDefault()}
                  onClick={() => setOpponent('')}
                  aria-label="Очистить оппонента"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <path
                      d="M7 7l10 10M17 7L7 17"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                  </svg>
                </button>
              </div>
              <div className="create-bet-field-label">
                Ставка (TON)<span className="create-bet-required-mark" aria-hidden="true">*</span>
              </div>
              <div className="create-bet-input-wrap">
                <input
                  className="create-bet-input"
                  type="number"
                  step="0.01"
                  value={amount || ''}
                  onChange={e => {
                    hasUserInputRef.current = true;
                    setAmount(e.target.value === '' ? 0 : parseFloat(e.target.value));
                  }}
                  required
                />
                <button
                  type="button"
                  className={`create-bet-input-clear${amount ? ' visible' : ''}`}
                  onMouseDown={event => event.preventDefault()}
                  onClick={() => setAmount(0)}
                  aria-label="Очистить ставку"
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <path
                      d="M7 7l10 10M17 7L7 17"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                  </svg>
                </button>
              </div>
            </section>

            <section className="create-bet-section">
              <div className="create-bet-field-label">
                Окончание пари<span className="create-bet-required-mark" aria-hidden="true">*</span>
              </div>
              <div className="create-bet-datetime-row">
                <div className="create-bet-input-wrap create-bet-date-wrap">
                  <input
                    className={`create-bet-input create-bet-date-input${isEndsAtInvalid ? ' create-bet-input-invalid' : ''}`}
                    type="date"
                    value={formatDateInputValue(endsAt)}
                    min={formatDateInputValue(minAllowedEndsAt)}
                    onChange={event => {
                      hasUserInputRef.current = true;
                      const value = event.target.value;
                      if (!value) return;
                      const [yearRaw, monthRaw, dayRaw] = value.split('-');
                      const year = Number(yearRaw);
                      const month = Number(monthRaw);
                      const day = Number(dayRaw);
                      if (!year || !month || !day) return;
                      const next = new Date(endsAt);
                      next.setFullYear(year, month - 1, day);
                      const rounded = roundToMinute(next);
                      setEndsAt(rounded.getTime() < minAllowedEndsAtMs ? minAllowedEndsAt : rounded);
                    }}
                    required
                  />
                </div>
                <TimePicker
                  value={endsAt}
                  onChange={next => {
                    hasUserInputRef.current = true;
                    const rounded = roundToMinute(next);
                    setEndsAt(rounded.getTime() < minAllowedEndsAtMs ? minAllowedEndsAt : rounded);
                  }}
                  minuteStep={1}
                  min={isSameDay(endsAt, minAllowedEndsAt) ? formatTimeInputValue(minAllowedEndsAt) : undefined}
                  className={`create-bet-time-picker${isEndsAtInvalid ? ' create-bet-input-invalid' : ''}`}
                />
              </div>
              <p className={`create-bet-field-hint${isEndsAtInvalid ? ' create-bet-field-hint-error' : isShortAcceptanceWindow ? ' create-bet-field-hint-warning' : ''}`}>
                {acceptanceHintText}
              </p>
            </section>

            <section className="create-bet-section">
              <div className="create-bet-file-section">
                <div className="create-bet-file-section-label">Файлы</div>
                <FileInput
                  className="create-bet-file-input"
                  label="Добавить"
                  maxFiles={CREATE_BET_FILE_INPUT_MAX_FILES}
                  onHasErrorChange={setFileInputHasError}
                  onPrimaryFileChange={setFile}
                />
              </div>
            </section>

            <div className="create-bet-form-actions">
              <button
                type="submit"
                className={`create-bet-submit-btn${!connected ? ' create-bet-submit-btn-wallet-disconnected' : ''}${submitShake ? ' create-bet-submit-btn-shake' : ''}`}
                disabled={submitting || fileInputHasError || !isRequiredFieldsFilled}
                onAnimationEnd={() => setSubmitShake(false)}
                title={connected ? undefined : 'Подключите TON-кошелёк'}
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
