import React, { FormEvent, useCallback, useEffect, useRef, useState } from 'react';
import { backButton, hideKeyboard } from '@tma.js/sdk-react';
import { apiFetch } from '../../utils/apiFetch';
import { FileInput } from '../FileInput/FileInput';
import { Alert } from '../ui/alert/Alert';
import { AutoGrowTextarea } from '../ui/auto-grow-textarea/AutoGrowTextarea';
import '../CreateBetForm/CreateBetForm.css';
import './EvidenceForm.css';
import { useBlockedActionFeedback } from '../../hooks/useBlockedActionFeedback';

interface Props {
  disputeId: string;
  onClose: () => void;
  onSubmitted: () => void;
}

interface EvidenceDraft {
  statement: string;
}

const EVIDENCE_FILE_INPUT_MAX_FILES = 1;
const EVIDENCE_TEXT_MIN_HEIGHT_PX = 96;
const EVIDENCE_TEXT_MAX_LENGTH = 4096;
const EVIDENCE_TEXT_WARNING_LENGTH = 3600;
const EVIDENCE_DRAFT_KEY_PREFIX = 'evidence-draft-v1:';

const getSessionStorage = (): Storage | null => {
  if (typeof window === 'undefined') return null;
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
};

const getEvidenceDraftKey = (disputeId: string): string => `${EVIDENCE_DRAFT_KEY_PREFIX}${disputeId}`;

const parseEvidenceDraft = (raw: string | null): EvidenceDraft | null => {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<EvidenceDraft>;
    if (typeof parsed.statement !== 'string') return null;
    return { statement: parsed.statement };
  } catch {
    return null;
  }
};

const isFromEvidenceTextarea = (target: EventTarget | null): boolean => {
  if (!(target instanceof HTMLElement)) return false;
  return target.closest('.evidence-form-textarea') !== null;
};

export const EvidenceForm: React.FC<Props> = ({ disputeId, onClose, onSubmitted }) => {
  const draftLoadedRef = useRef(false);
  const statementInputRef = useRef<HTMLTextAreaElement | null>(null);
  const closeInFlightRef = useRef(false);
  const submittedNotifiedRef = useRef(false);
  const attemptCloseRef = useRef<() => Promise<void>>(async () => {});
  const touchStartYRef = useRef<number | null>(null);
  const touchHideTriggeredRef = useRef(false);

  const [statement, setStatement] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [fileInputHasError, setFileInputHasError] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showValidationErrors, setShowValidationErrors] = useState(false);
  const [showDraftRestoredAlert, setShowDraftRestoredAlert] = useState(false);
  const {
    isShaking: submitShake,
    handleShakeAnimationEnd: handleSubmitShakeAnimationEnd,
    triggerBlockedActionFeedback,
  } = useBlockedActionFeedback();

  const draftKey = getEvidenceDraftKey(disputeId);
  const isStatementEmpty = statement.trim().length === 0;
  const isStatementTooLong = statement.length > EVIDENCE_TEXT_MAX_LENGTH;
  const isStatementNearLimit = statement.length >= EVIDENCE_TEXT_WARNING_LENGTH && !isStatementTooLong;
  const shouldShowStatementValidation = (showValidationErrors && isStatementEmpty) || isStatementTooLong;
  const statementValidationText = isStatementTooLong
    ? `Текст должен быть не длиннее ${EVIDENCE_TEXT_MAX_LENGTH} символов`
    : 'Заполните обязательное поле';
  const isRequiredFieldsFilled = !isStatementEmpty && !isStatementTooLong;
  const isSubmitBlockedByValidation = !isRequiredFieldsFilled;
  const hasSavableDraftData = !isStatementEmpty;

  const notifySubmittedIfNeeded = useCallback(() => {
    if (submittedNotifiedRef.current) return;
    submittedNotifiedRef.current = true;
    onSubmitted();
  }, [onSubmitted]);

  const saveDraftToStorage = useCallback((draft: EvidenceDraft) => {
    const storage = getSessionStorage();
    if (!storage) return;
    storage.setItem(draftKey, JSON.stringify(draft));
  }, [draftKey]);

  const clearDraftFromStorage = useCallback(() => {
    const storage = getSessionStorage();
    if (!storage) return;
    storage.removeItem(draftKey);
  }, [draftKey]);

  const hideKeyboardSafe = useCallback(() => {
    if (!hideKeyboard.isSupported()) return;
    hideKeyboard();
  }, []);

  attemptCloseRef.current = async () => {
    if (closeInFlightRef.current || submitting) return;
    closeInFlightRef.current = true;
    try {
      if (success) {
        clearDraftFromStorage();
        notifySubmittedIfNeeded();
      }
      onClose();
    } finally {
      closeInFlightRef.current = false;
    }
  };

  useEffect(() => {
    if (!backButton.isSupported()) return;
    const offClick = backButton.onClick(() => {
      void attemptCloseRef.current();
    });
    return () => {
      offClick();
    };
  }, []);

  useEffect(() => {
    draftLoadedRef.current = true;
    const storage = getSessionStorage();
    if (!storage) return;
    const draft = parseEvidenceDraft(storage.getItem(draftKey));
    if (!draft) return;
    setStatement(draft.statement);
    setShowDraftRestoredAlert(true);
  }, [draftKey]);

  useEffect(() => {
    if (!draftLoadedRef.current) return;
    if (success) {
      clearDraftFromStorage();
      return;
    }
    if (hasSavableDraftData) {
      saveDraftToStorage({ statement });
      return;
    }
    clearDraftFromStorage();
  }, [success, hasSavableDraftData, statement, saveDraftToStorage, clearDraftFromStorage]);

  const handleCancelDraftRestore = useCallback(() => {
    setStatement('');
    setError(null);
    setShowValidationErrors(false);
    setShowDraftRestoredAlert(false);
    clearDraftFromStorage();
  }, [clearDraftFromStorage]);

  const handleScreenWheel = useCallback((event: React.WheelEvent<HTMLDivElement>) => {
    if (isFromEvidenceTextarea(event.target)) return;
    if (Math.abs(event.deltaY) < 2) return;
    hideKeyboardSafe();
  }, [hideKeyboardSafe]);

  const handleScreenTouchStart = useCallback((event: React.TouchEvent<HTMLDivElement>) => {
    if (isFromEvidenceTextarea(event.target)) return;
    if (event.touches.length !== 1) return;
    touchStartYRef.current = event.touches[0].clientY;
    touchHideTriggeredRef.current = false;
  }, []);

  const handleScreenTouchMove = useCallback((event: React.TouchEvent<HTMLDivElement>) => {
    if (isFromEvidenceTextarea(event.target)) return;
    if (touchHideTriggeredRef.current) return;
    if (event.touches.length !== 1 || touchStartYRef.current === null) return;

    const deltaY = event.touches[0].clientY - touchStartYRef.current;
    if (Math.abs(deltaY) > 12) {
      hideKeyboardSafe();
      touchHideTriggeredRef.current = true;
    }
  }, [hideKeyboardSafe]);

  const resetTouchTracking = useCallback(() => {
    touchStartYRef.current = null;
    touchHideTriggeredRef.current = false;
  }, []);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (submitting) return;
    setError(null);
    setSuccess(false);

    if (isStatementTooLong) {
      setShowValidationErrors(true);
      triggerBlockedActionFeedback(() => statementInputRef.current);
      return;
    }

    if (!isRequiredFieldsFilled) {
      setShowValidationErrors(true);
      triggerBlockedActionFeedback(() => statementInputRef.current);
      return;
    }

    setShowValidationErrors(false);
    setSubmitting(true);

    try {
      const form = new FormData();
      form.append('description', statement);
      if (file) form.append('evidence', file);

      const res = await apiFetch(`/api/v1/disputes/${disputeId}/evidence`, {
        method: 'POST',
        body: form,
      });

      if (!res.ok) {
        throw new Error(`Ошибка ${res.status}`);
      }

      clearDraftFromStorage();
      setSuccess(true);
    } catch {
      setError('Не удалось отправить доказательства');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      className="create-bet-screen evidence-form-screen"
      onWheel={handleScreenWheel}
      onTouchStart={handleScreenTouchStart}
      onTouchMove={handleScreenTouchMove}
      onTouchEnd={resetTouchTracking}
      onTouchCancel={resetTouchTracking}
    >
      <form className="create-bet-page evidence-form-page" onSubmit={handleSubmit} noValidate>
        <header className="create-bet-header">
          <h3>Внесение доказательств</h3>
        </header>

        {error && <div className="create-bet-error-message">{error}</div>}

        {success ? (
          <div className="create-bet-success-message">
            Доказательства успешно отправлены
            <button
              type="button"
              className="create-bet-close-success-btn"
              onClick={() => {
                clearDraftFromStorage();
                notifySubmittedIfNeeded();
                onClose();
              }}
            >
              К пари
            </button>
          </div>
        ) : (
          <>
            <section className="create-bet-section">
              <div className="create-bet-field-label">
                ПОЯСНЕНИЕ
                <span className="create-bet-required-mark" aria-hidden="true">*</span>
              </div>
              <div className="create-bet-input-wrap create-bet-textarea-wrap">
                <AutoGrowTextarea
                  ref={statementInputRef}
                  className={`create-bet-input create-bet-textarea evidence-form-textarea${shouldShowStatementValidation ? ' create-bet-input-invalid' : ''}`}
                  minHeight={EVIDENCE_TEXT_MIN_HEIGHT_PX}
                  value={statement}
                  onValueChange={value => setStatement(value)}
                  placeholder="Опишите вашу позицию и что подтверждают файлы"
                  required
                />
                <button
                  type="button"
                  className={`create-bet-input-clear${statement.length > 0 ? ' visible' : ''}`}
                  onMouseDown={event => event.preventDefault()}
                  onClick={() => setStatement('')}
                  aria-label="Очистить поле"
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
              {statement.length > 0 && (
                <p className={`create-bet-field-hint${isStatementTooLong ? ' create-bet-hint-error' : isStatementNearLimit ? ' create-bet-hint-warning' : ''}`}>
                  {statement.length}/{EVIDENCE_TEXT_MAX_LENGTH}
                </p>
              )}
              {shouldShowStatementValidation && (
                <p className="create-bet-field-hint create-bet-hint-error">
                  {statementValidationText}
                </p>
              )}
            </section>

            <section className="create-bet-section">
              <div className="create-bet-file-section">
                <div className="create-bet-field-label">ФАЙЛЫ</div>
                <FileInput
                  className="create-bet-file-input"
                  label="Добавить"
                  maxFiles={EVIDENCE_FILE_INPUT_MAX_FILES}
                  onHasErrorChange={setFileInputHasError}
                  onPrimaryFileChange={setFile}
                />
              </div>
            </section>

            <div className="create-bet-form-actions">
              <button
                type="submit"
                className={`create-bet-submit-btn${isSubmitBlockedByValidation ? ' create-bet-submit-btn-blocked' : ''}${submitShake ? ' action-shake' : ''}`}
                disabled={submitting || fileInputHasError}
                aria-disabled={submitting || fileInputHasError || isSubmitBlockedByValidation}
                onAnimationEnd={handleSubmitShakeAnimationEnd}
              >
                {submitting ? 'Отправка…' : 'Отправить доказательства'}
              </button>
            </div>
          </>
        )}
      </form>
      <Alert
        floating
        placement="bottom"
        status="success"
        open={showDraftRestoredAlert}
        onOpenChange={setShowDraftRestoredAlert}
        durationMs={4000}
        title="Черновик был восстановлен"
        actionLabel="Очистить"
        actionVariant="default"
        onAction={handleCancelDraftRestore}
      />
    </div>
  );
};
