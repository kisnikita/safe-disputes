import { ChangeEvent, forwardRef, InputHTMLAttributes } from 'react';

type NativeInputProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  'type' | 'value' | 'onChange' | 'inputMode' | 'pattern'
>;

export interface AmountInputProps extends NativeInputProps {
  value: string;
  onValueChange: (value: string) => void;
  maxFractionDigits?: number;
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
