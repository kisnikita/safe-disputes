import { useState, useEffect, useRef, useCallback } from 'react';
import { apiFetch } from '../../utils/apiFetch';
import './SettingsSection.css';
import { Spinner } from '@telegram-apps/telegram-ui';
import { popup } from '@tma.js/sdk-react';
import { UserAvatar } from '../UserAvatar/UserAvatar';
import {
  AmountInput,
  DEFAULT_AMOUNT_MAX_FRACTION_DIGITS,
  validateAmountValue,
} from '../AmountInput/AmountInput';
import { TonIcon } from '../TonIcon/TonIcon';
import { formatNanoToTon } from '../../utils/tonAmount';
import { useBlockedActionFeedback } from '../../hooks/useBlockedActionFeedback';

interface UserSettings {
  notificationEnabled: boolean;
  disputeReadiness: boolean;
  investigationReadiness: boolean;
  minimumDisputeAmountNano: string;
  chatID: number;
}

interface UserSettingsResponse {
  notificationEnabled: boolean;
  disputeReadiness: boolean;
  investigationReadiness: boolean;
  minimumDisputeAmountNano: string | number;
  chatID: number;
}

interface SettingsSectionProps {
  username?: string;
  userPhotoUrl?: string | null;
}

const NOTIFICATION_BOT_URL = 'https://t.me/SafeDisputesBot';

export function SettingsSection({ username = '', userPhotoUrl = null }: SettingsSectionProps) {
  const [settings, setSettings] = useState<UserSettings>({
    notificationEnabled: false,
    disputeReadiness: true,
    investigationReadiness: true,
    minimumDisputeAmountNano: '0',
    chatID: 0,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving]   = useState(false);

  const [minInput, setMinInput] = useState<string>('');
  const minInputRef = useRef<HTMLInputElement>(null);
  const {
    isShaking: minSaveShake,
    triggerShake: triggerMinSaveShake,
    handleShakeAnimationEnd: handleMinSaveShakeAnimationEnd,
  } = useBlockedActionFeedback();
  const refreshInFlightRef = useRef(false);
  const lastSilentRefreshAtRef = useRef(0);

  const loadSettings = useCallback(async (options?: { silent?: boolean; syncMinInput?: boolean }) => {
    const silent = options?.silent ?? false;
    const syncMinInput = options?.syncMinInput ?? false;
    if (refreshInFlightRef.current) return;

    refreshInFlightRef.current = true;
    if (!silent) setLoading(true);
    try {
      const res = await apiFetch('/api/v1/users/me');
      if (!res.ok) throw new Error();
      const { data } = await res.json() as { data: UserSettingsResponse };
      const normalizedData: UserSettings = {
        ...data,
        investigationReadiness: data.investigationReadiness ?? true,
        minimumDisputeAmountNano: String(data.minimumDisputeAmountNano ?? '0'),
      };
      setSettings(normalizedData);
      if (syncMinInput) {
        setMinInput(formatNanoToTon(normalizedData.minimumDisputeAmountNano, 9));
      }
    } catch (err) {
      console.error(err);
    } finally {
      if (!silent) setLoading(false);
      refreshInFlightRef.current = false;
    }
  }, []);

  useEffect(() => {
    void loadSettings({ syncMinInput: true });
  }, [loadSettings]);

  useEffect(() => {
    const blurMinInput = () => {
      if (document.activeElement === minInputRef.current) {
        minInputRef.current?.blur();
      }
    };

    const contentEl = document.querySelector('.content');
    contentEl?.addEventListener('scroll', blurMinInput, { passive: true });
    window.addEventListener('wheel', blurMinInput, { passive: true });
    window.addEventListener('touchmove', blurMinInput, { passive: true });

    return () => {
      contentEl?.removeEventListener('scroll', blurMinInput);
      window.removeEventListener('wheel', blurMinInput);
      window.removeEventListener('touchmove', blurMinInput);
    };
  }, []);

  useEffect(() => {
    const refreshAfterReturn = () => {
      if (saving) return;
      const now = Date.now();
      if (now - lastSilentRefreshAtRef.current < 1200) return;
      lastSilentRefreshAtRef.current = now;
      void loadSettings({ silent: true });
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refreshAfterReturn();
      }
    };

    window.addEventListener('focus', refreshAfterReturn);
    window.addEventListener('pageshow', refreshAfterReturn);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      window.removeEventListener('focus', refreshAfterReturn);
      window.removeEventListener('pageshow', refreshAfterReturn);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [loadSettings, saving]);

  const updateField = async <K extends keyof UserSettings>(field: K, value: UserSettings[K]) => {
    setSaving(true);
    try {
      const res = await apiFetch('/api/v1/users', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ [field]: value }),
      });
      if (!res.ok) throw new Error();
      setSettings(prev => ({ ...prev, [field]: value }));
    } catch (err) {
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  const openNotificationBot = () => {
    const webApp = (window as any)?.Telegram?.WebApp;
    if (typeof webApp?.openTelegramLink === 'function') {
      webApp.openTelegramLink(NOTIFICATION_BOT_URL);
      return;
    }
    window.open(NOTIFICATION_BOT_URL, '_blank', 'noopener,noreferrer');
  };

  const showNotificationBotPopup = async () => {
    const message = 'Чтобы включить уведомления, отправьте любое сообщение боту приложения';

    if (popup.isSupported()) {
      const buttonId = await popup.show({
        message,
        buttons: [
          { id: 'open', type: 'default', text: 'Открыть бота' },
          { id: 'ok', type: 'ok'},
        ],
      });

      if (buttonId === 'open') {
        openNotificationBot();
      }
      return;
    }

    if (window.confirm(`${message}\n\nОткрыть бота сейчас?`)) {
      openNotificationBot();
    }
  };

  if (loading) {
    return (
      <div className="settings-status">
        <Spinner size="l" className="spinner"/>
      </div>
    );
  }

  const minAmountValidation = validateAmountValue(minInput, {
    maxFractionDigits: DEFAULT_AMOUNT_MAX_FRACTION_DIGITS,
    allowZero: true,
  });
  const parsedMinNano = minAmountValidation.parsedNano;
  const minValidationText = minAmountValidation.validationText;
  const isMinInvalid = minAmountValidation.isInvalid;
  const minChanged = parsedMinNano !== null && parsedMinNano !== settings.minimumDisputeAmountNano;
  const shouldShowMinSaveButton = minChanged || isMinInvalid;

  const notifDisabled = saving;
  const normalizedUsername = username.replace(/^@+/, '').trim();
  const login = normalizedUsername ? `@${normalizedUsername}` : '@user';

  return (
    <div className="settings-screen">
      <div className="settings-profile">
        <UserAvatar
          username={normalizedUsername || 'user'}
          photoUrl={userPhotoUrl}
          size={96}
          className="settings-profile-avatar"
        />
        <p className="settings-profile-login">{login}</p>
      </div>

      <div className="settings-card">
        {/* Уведомления */}
        <div className="settings-row">
          <span className="settings-label">Уведомления</span>
          <label className="switch">
            <input
              type="checkbox"
              checked={settings.notificationEnabled}
              disabled={notifDisabled}
              onChange={async e => {
                const next = e.target.checked;
                if (next && settings.chatID === 0) {
                  await showNotificationBotPopup();
                  return;
                }
                await updateField('notificationEnabled', next);
              }}
            />
            <span className="slider" />
          </label>
        </div>
      </div>

      <div className="settings-card">
        {/* Готовность к новым пари */}
        <div className="settings-row">
          <span className="settings-label">Готовность к новым пари</span>
          <label className="switch">
            <input
              type="checkbox"
              checked={settings.disputeReadiness}
              disabled={saving}
              onChange={e => updateField('disputeReadiness', e.target.checked)}
            />
            <span className="slider" />
          </label>
        </div>

        {/* Готовность к новым расследованиям */}
        <div className="settings-row">
          <span className="settings-label">Готовность к новым расследованиям</span>
          <label className="switch">
            <input
              type="checkbox"
              checked={settings.investigationReadiness}
              disabled={saving}
              onChange={e => updateField('investigationReadiness', e.target.checked)}
            />
            <span className="slider" />
          </label>
        </div>
      </div>

      <div className="settings-card">
        {/* Минимальная ставка */}
        <div className="settings-row settings-row-min-amount">
          <span className="settings-label settings-label-with-icon">
            <TonIcon className="settings-ton-icon" title="TON" />
            <span>Минимальная ставка</span>
          </span>
          <div className="settings-min-amount-controls">
            <div className="min-input-wrapper">
              <AmountInput
                className={`settings-input${isMinInvalid ? ' settings-input-invalid' : ''}`}
                ref={minInputRef}
                value={minInput}
                maxFractionDigits={DEFAULT_AMOUNT_MAX_FRACTION_DIGITS}
                disabled={saving}
                onValueChange={setMinInput}
              />
              {shouldShowMinSaveButton && (
                <button
                  className={`save-button${isMinInvalid ? ' save-button-blocked' : ''}${minSaveShake ? ' action-shake' : ''}`}
                  onClick={() => {
                    if (parsedMinNano === null || isMinInvalid) {
                      triggerMinSaveShake();
                      return;
                    }
                    void updateField('minimumDisputeAmountNano', parsedMinNano);
                  }}
                  disabled={saving}
                  aria-disabled={saving || isMinInvalid}
                  onAnimationEnd={handleMinSaveShakeAnimationEnd}
                >
                  {saving ? '…' : 'Сохранить'}
                </button>
              )}
            </div>
            {isMinInvalid && minValidationText && (
              <p className="settings-field-hint settings-hint-error">
                {minValidationText}
              </p>
            )}
          </div>
        </div>
      </div>

      {saving && <div className="settings-saving">
        <Spinner size="l" className="spinner"/>
      </div>}
    </div>
  );
}
