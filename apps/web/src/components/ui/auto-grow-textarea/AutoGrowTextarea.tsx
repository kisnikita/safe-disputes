import React, {
  ForwardedRef,
  forwardRef,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from 'react';
import './AutoGrowTextarea.css';

type Props = Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, 'value' | 'onChange'> & {
  value: string;
  onValueChange: (value: string) => void;
  minHeight?: number;
};

const assignRef = (
  forwardedRef: ForwardedRef<HTMLTextAreaElement>,
  value: HTMLTextAreaElement | null,
) => {
  if (typeof forwardedRef === 'function') {
    forwardedRef(value);
    return;
  }
  if (forwardedRef) {
    forwardedRef.current = value;
  }
};

export const AutoGrowTextarea = forwardRef<HTMLTextAreaElement, Props>(({
  value,
  onValueChange,
  minHeight = 80,
  className,
  style,
  onInput,
  ...rest
}, ref) => {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  const setTextareaRef = useCallback((node: HTMLTextAreaElement | null) => {
    textareaRef.current = node;
    assignRef(ref, node);
  }, [ref]);

  const resize = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = `${Math.max(textarea.scrollHeight, minHeight)}px`;
  }, [minHeight]);

  useEffect(() => {
    resize();
  }, [resize, value]);

  const mergedStyle = useMemo<React.CSSProperties>(() => ({
    ...style,
    minHeight: `${minHeight}px`,
  }), [style, minHeight]);

  return (
    <textarea
      {...rest}
      ref={setTextareaRef}
      className={`auto-grow-textarea${className ? ` ${className}` : ''}`}
      style={mergedStyle}
      value={value}
      onChange={event => onValueChange(event.target.value)}
      onInput={event => {
        resize();
        onInput?.(event);
      }}
    />
  );
});

AutoGrowTextarea.displayName = 'AutoGrowTextarea';
