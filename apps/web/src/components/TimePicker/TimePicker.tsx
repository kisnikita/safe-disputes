import React from 'react';
import './TimePicker.css';

interface TimePickerProps {
  value: Date;
  onChange: (next: Date) => void;
  disabled?: boolean;
  minuteStep?: number;
  className?: string;
  min?: string;
}

const toTwo = (value: number): string => value.toString().padStart(2, '0');

const formatTime = (value: Date): string => `${toTwo(value.getHours())}:${toTwo(value.getMinutes())}`;

const cloneWithTime = (base: Date, hour: number, minute: number): Date => {
  const next = new Date(base);
  next.setHours(hour, minute, 0, 0);
  return next;
};

export const TimePicker: React.FC<TimePickerProps> = ({
  value,
  onChange,
  disabled = false,
  minuteStep = 1,
  className,
  min,
}) => {
  const stepSeconds = Math.max(1, Math.floor(minuteStep)) * 60;

  return (
    <div className={`time-picker${className ? ` ${className}` : ''}`}>
      <input
        type="time"
        value={formatTime(value)}
        min={min}
        step={stepSeconds}
        disabled={disabled}
        className="time-picker-input"
        onChange={event => {
          const raw = event.target.value;
          if (!raw) return;
          const [hourRaw, minuteRaw] = raw.split(':');
          const hour = Number(hourRaw);
          const minute = Number(minuteRaw);
          if (Number.isNaN(hour) || Number.isNaN(minute)) return;
          onChange(cloneWithTime(value, Math.max(0, Math.min(23, hour)), Math.max(0, Math.min(59, minute))));
        }}
      />
    </div>
  );
};
