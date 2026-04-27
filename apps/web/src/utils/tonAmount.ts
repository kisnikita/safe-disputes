import { toNano } from '@ton/core';

const NANO_PER_TON = 1_000_000_000n;

export const parseTonToNano = (value: string, options?: { allowZero?: boolean }): string | null => {
  const normalized = value.trim();
  if (normalized.length === 0) return null;

  try {
    const nano = toNano(normalized);
    if (nano < 0n) return null;
    if (!options?.allowZero && nano === 0n) return null;
    return nano.toString();
  } catch {
    return null;
  }
};

export const formatNanoToTon = (
  value: string | number | bigint,
  maxFractionDigits = 2,
  options?: { keepTrailingZeros?: boolean },
): string => {
  try {
    const nano = typeof value === 'bigint' ? value : BigInt(value);
    const sign = nano < 0n ? '-' : '';
    const absolute = nano < 0n ? -nano : nano;

    const whole = absolute / NANO_PER_TON;
    const fraction = absolute % NANO_PER_TON;

    if (maxFractionDigits <= 0) {
      return `${sign}${whole.toString()}`;
    }

    const targetFractionDigits = Math.min(9, maxFractionDigits);

    let fractionString = fraction
      .toString()
      .padStart(9, '0')
      .slice(0, targetFractionDigits);

    if (!options?.keepTrailingZeros) {
      fractionString = fractionString.replace(/0+$/, '');
    }

    if (fractionString.length === 0) {
      return `${sign}${whole.toString()}`;
    }

    return `${sign}${whole.toString()}.${fractionString}`;
  } catch {
    return '0';
  }
};
