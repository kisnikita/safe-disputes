import { ChangeEvent, forwardRef, InputHTMLAttributes } from 'react';
import { parseTonToNano } from '../../utils/tonAmount';

type NativeInputProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  'type' | 'value' | 'onChange' | 'inputMode' | 'pattern'
>;

export interface AmountInputProps extends NativeInputProps {
  value: string;
  onValueChange: (value: string) => void;
  maxFractionDigits?: number;
}

export interface AmountValidationOptions {
  maxFractionDigits?: number;
  allowZero?: boolean;
  minNano?: string | bigint | null;
  minDisplayTon?: string;
}

export interface AmountValidationResult {
  parsedNano: string | null;
  validationText: string | null;
  isEmpty: boolean;
  isInvalid: boolean;
}

export const DEFAULT_AMOUNT_MAX_FRACTION_DIGITS = 2;
const MAX_INT64_NANO = 9_223_372_036_854_775_807n;

export function validateAmountValue(
  value: string,
  options?: AmountValidationOptions,
): AmountValidationResult {
  const maxFractionDigits = options?.maxFractionDigits ?? DEFAULT_AMOUNT_MAX_FRACTION_DIGITS;
  const isEmpty = value.trim().length === 0;
  const fractionPart = value.split('.')[1] ?? '';
  const hasTooManyFractionDigits = fractionPart.length > maxFractionDigits;
  const parsedNano = parseTonToNano(value, { allowZero: options?.allowZero });
  const isAmountInvalidNumber = value !== '' && parsedNano === null;
  const amountNanoBigInt = parsedNano !== null ? BigInt(parsedNano) : null;
  const minNanoBigInt = options?.minNano !== null && options?.minNano !== undefined
    ? BigInt(options.minNano)
    : null;
  const isAmountBelowMin = amountNanoBigInt !== null
    && minNanoBigInt !== null
    && amountNanoBigInt < minNanoBigInt;
  const isAmountTooLarge = amountNanoBigInt !== null && amountNanoBigInt > MAX_INT64_NANO;

  const validationText = hasTooManyFractionDigits
    ? `Ставка может содержать не более ${maxFractionDigits} знаков после запятой`
    : isAmountInvalidNumber
      ? 'Введите корректную сумму ставки'
      : isAmountBelowMin
        ? options?.minDisplayTon
          ? `Минимальная ставка: ${options.minDisplayTon} TON`
          : 'Сумма меньше минимально допустимой'
        : isAmountTooLarge
          ? 'Максимальная ставка: 9 млрд TON'
          : null;
  const isInvalid = !isEmpty && validationText !== null;

  return {
    parsedNano,
    validationText,
    isEmpty,
    isInvalid,
  };
}

export function normalizeAmountInput(
  raw: string,
  prevValue = '',
  options?: { maxFractionDigits?: number },
): string {
  let value = raw.replace(',', '.').replace(/[^\d.]/g, '');

  // If previous value was just "0", replace it with a typed non-zero digit
  // even when keyboard inserts that digit before zero (e.g. "50" -> "5").
  if (prevValue === '0' && (/^0[1-9]$/.test(value) || /^[1-9]0$/.test(value))) {
    value = value.replace('0', '');
  }

  const firstDotIdx = value.indexOf('.');
  if (firstDotIdx !== -1) {
    value = `${value.slice(0, firstDotIdx + 1)}${value.slice(firstDotIdx + 1).replace(/\./g, '')}`;
  }

  const hasDot = value.includes('.');
  const [rawIntPart = '', rawFractionPart = ''] = value.split('.');
  let intPart = rawIntPart;
  const maxFractionDigits = options?.maxFractionDigits;

  if (intPart.length > 1) {
    intPart = intPart.replace(/^0+(?=\d)/, '');
  }

  if (hasDot && intPart === '') {
    intPart = '0';
  }

  if (!hasDot) {
    return intPart;
  }

  const fractionPart = typeof maxFractionDigits === 'number' && maxFractionDigits >= 0
    ? rawFractionPart.slice(0, maxFractionDigits)
    : rawFractionPart;

  return `${intPart}.${fractionPart}`;
}

export const AmountInput = forwardRef<HTMLInputElement, AmountInputProps>(
  function AmountInput({ value, onValueChange, maxFractionDigits, ...props }, ref) {
    const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
      onValueChange(normalizeAmountInput(event.target.value, value, { maxFractionDigits }));
    };

    return (
      <input
        {...props}
        ref={ref}
        type="text"
        inputMode="decimal"
        pattern="[0-9]*[.,]?[0-9]*"
        value={value}
        onChange={handleChange}
      />
    );
  },
);
