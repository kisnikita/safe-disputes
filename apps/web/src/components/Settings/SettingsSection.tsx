// src/components/Settings/SettingsSection.tsx
import React, { useState, useEffect } from 'react';
import { apiFetch } from '../../utils/apiFetch';
import './SettingsSection.css';

interface UserSettings {
  notificationEnabled: boolean;
  disputeReadiness: boolean;
  minimumDisputeAmount: number;
  chatID: number;
}

export function SettingsSection() {
  const [settings, setSettings] = useState<UserSettings>({
    notificationEnabled: false,
    disputeReadiness: true,
    minimumDisputeAmount: 0,
    chatID: 0,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving]   = useState(false);
  const [showHint, setShowHint] = useState(false); // <-- для подсказки

  const [minInput, setMinInput] = useState<string>('');

  useEffect(() => {
    (async () => {
      try {
        const res = await apiFetch('/api/v1/users/me');
        if (!res.ok) throw new Error();
        const { data } = await res.json() as { data: UserSettings };
        setSettings(data);
        setMinInput(data.minimumDisputeAmount.toString());
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

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

  if (loading) {
    return <div className="settings-status">Загрузка настроек…</div>;
  }

  const parsedMin = parseFloat(minInput);
  const minChanged = !isNaN(parsedMin) && parsedMin !== settings.minimumDisputeAmount;

  const notifDisabled = saving || settings.chatID === 0;

  return (
    <div className="settings-panel">
      {/* Уведомления */}
      <div className="settings-row" style={{ position: 'relative' }}>
        <span className="settings-label">Уведомления</span>
        <label
          className="switch"
          onMouseEnter={() => settings.chatID === 0 && setShowHint(true)}
          onMouseLeave={() => setShowHint(false)}
          onMouseUp={()  => settings.chatID === 0 && setShowHint(true)}
        >
          <input
            type="checkbox"
            checked={settings.notificationEnabled}
            disabled={notifDisabled}
            onChange={e => updateField('notificationEnabled', e.target.checked)}
          />
          <span className="slider" />
        </label>
        {showHint && (
          <div className="big-hint">
            Чтобы включить уведомления,<br/>
            отправьте сообщение боту приложения
          </div>
        )}
      </div>

      {/* остальные настройки как было */}
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

      {/* Минимальная ставка */}
      <div className="settings-row">
        <span className="settings-label">Минимальная ставка (TON)</span>
        <div className="min-input-wrapper">
          <input
            type="number"
            min="0"
            step="0.01"
            className="settings-input"
            value={minInput}
            disabled={saving}
            onChange={e => setMinInput(e.target.value)}
          />
          {minChanged && (
            <button
              className="save-button"
              onClick={() => updateField('minimumDisputeAmount', parsedMin)}
              disabled={saving}
            >
              {saving ? '…' : 'Сохранить'}
            </button>
          )}
        </div>
      </div>

      {saving && <div className="settings-status">Сохранение…</div>}
    </div>
  );
}
