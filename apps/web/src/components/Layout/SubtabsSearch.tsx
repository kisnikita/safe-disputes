import React, { useEffect, useRef, useState } from 'react';
import './SubtabsSearch.css';

const DEFAULT_SORT_OPTIONS = ['Последние', 'Крупные'];

interface SubtabsSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  hidden?: boolean;
  blurOnSwipe?: boolean;
  filterOptions?: Array<string | { label: string; color?: string }>;
  sortOptions?: string[];
  resetKey?: string | number;
  selectedFilters?: string[];
  onSelectedFiltersChange?: (value: string[]) => void;
  selectedSort?: string;
  onSelectedSortChange?: (value: string) => void;
}

export const SubtabsSearch: React.FC<SubtabsSearchProps> = ({
  value,
  onChange,
  placeholder,
  hidden = false,
  blurOnSwipe = false,
  filterOptions = [],
  sortOptions = DEFAULT_SORT_OPTIONS,
  resetKey,
  selectedFilters: controlledSelectedFilters,
  onSelectedFiltersChange,
  selectedSort: controlledSelectedSort,
  onSelectedSortChange,
}) => {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const rootRef = useRef<HTMLDivElement | null>(null);
  const prevResetKeyRef = useRef<string | number | null | undefined>(resetKey);
  const [isFocused, setIsFocused] = useState(false);
  const [activeMenu, setActiveMenu] = useState<'filter' | 'sort' | null>(null);
  const [selectedFilters, setSelectedFilters] = useState<string[]>(controlledSelectedFilters ?? []);
  const [selectedSort, setSelectedSort] = useState<string>(
    controlledSelectedSort ?? (sortOptions[0] ?? 'Последние')
  );
  const hasSelectedFilters = selectedFilters.length > 0;
  const hasCustomSort = selectedSort !== (sortOptions[0] ?? 'Последние');
  const normalizedFilterOptions = filterOptions.map(option =>
    typeof option === 'string' ? { label: option } : option
  );

  useEffect(() => {
    if (!hidden && !blurOnSwipe) return;
    inputRef.current?.blur();
    setIsFocused(false);
    setActiveMenu(null);
  }, [hidden, blurOnSwipe]);

  useEffect(() => {
    const onPointerDown = (event: PointerEvent) => {
      if (!rootRef.current) return;
      if (rootRef.current.contains(event.target as Node)) return;
      setActiveMenu(null);
    };
    document.addEventListener('pointerdown', onPointerDown);
    return () => document.removeEventListener('pointerdown', onPointerDown);
  }, []);

  useEffect(() => {
    if (!controlledSelectedFilters) return;
    setSelectedFilters(controlledSelectedFilters);
  }, [controlledSelectedFilters]);

  useEffect(() => {
    if (!controlledSelectedSort) return;
    setSelectedSort(controlledSelectedSort);
  }, [controlledSelectedSort]);

  useEffect(() => {
    if (sortOptions.length === 0) return;
    if (!sortOptions.includes(selectedSort)) {
      setSelectedSort(sortOptions[0]);
      onSelectedSortChange?.(sortOptions[0]);
    }
  }, [sortOptions, selectedSort, onSelectedSortChange]);

  useEffect(() => {
    if (resetKey === undefined || resetKey === null) return;
    if (prevResetKeyRef.current === resetKey) return;
    prevResetKeyRef.current = resetKey;
    setSelectedFilters([]);
    onSelectedFiltersChange?.([]);
    const defaultSort = sortOptions[0] ?? 'Последние';
    setSelectedSort(defaultSort);
    onSelectedSortChange?.(defaultSort);
    setActiveMenu(null);
  }, [resetKey, onSelectedFiltersChange, onSelectedSortChange, sortOptions]);

  const handleCancel = () => {
    onChange('');
    inputRef.current?.blur();
    setIsFocused(false);
    setActiveMenu(null);
  };

  const toggleFilter = (value: string) => {
    setSelectedFilters(prev => {
      const next = prev.includes(value) ? prev.filter(item => item !== value) : [...prev, value];
      onSelectedFiltersChange?.(next);
      return next;
    });
  };

  return (
    <div
      ref={rootRef}
      className={`subtabs-search${hidden ? ' hidden' : ''}${isFocused ? ' focused' : ''}`}
    >
      <div className="subtabs-search-input-wrap">
        <input
          type="search"
          ref={inputRef}
          value={value}
          onFocus={() => {
            setIsFocused(true);
            setActiveMenu(null);
          }}
          onBlur={() => setIsFocused(false)}
          onChange={event => onChange(event.target.value)}
          placeholder={placeholder}
        />
        <button
          type="button"
          className={`subtabs-search-input-clear${value.length > 0 ? ' visible' : ''}`}
          onMouseDown={event => event.preventDefault()}
          onClick={() => {
            onChange('');
            inputRef.current?.focus();
          }}
          aria-label="Очистить поиск"
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
      <button
        type="button"
        className="subtabs-search-cancel"
        onMouseDown={event => event.preventDefault()}
        onClick={handleCancel}
      >
        Отмена
      </button>
      <div className="subtabs-search-controls">
        <button
          type="button"
          className={`subtabs-search-control subtabs-search-filter${activeMenu === 'filter' ? ' active' : ''}${hasSelectedFilters ? ' has-selected' : ''}`}
          onMouseDown={event => event.preventDefault()}
          onClick={() => setActiveMenu(prev => (prev === 'filter' ? null : 'filter'))}
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path
              d="M4 6.5c0-.83.67-1.5 1.5-1.5h13c.83 0 1.5.67 1.5 1.5 0 .37-.14.73-.39 1.01l-5.86 6.47a2 2 0 0 0-.52 1.34v2.74c0 .66-.32 1.28-.86 1.66l-1.66 1.19a1 1 0 0 1-1.58-.81v-4.78a2 2 0 0 0-.52-1.34L4.39 7.51A1.5 1.5 0 0 1 4 6.5z"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.9"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </button>
        <button
          type="button"
          className={`subtabs-search-control subtabs-search-sort${activeMenu === 'sort' ? ' active' : ''}${hasCustomSort ? ' has-custom-sort' : ''}`}
          onMouseDown={event => event.preventDefault()}
          onClick={() => setActiveMenu(prev => (prev === 'sort' ? null : 'sort'))}
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path
              d="M5 7h11M5 12h8M5 17h5"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.9"
              strokeLinecap="round"
            />
            <path
              d="M18.5 7.5v10m0 0-2.5-2.5m2.5 2.5 2.5-2.5"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.9"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </button>
      </div>
      <div className={`subtabs-search-dropdowns${activeMenu ? ' is-open' : ''}`}>
        <div className={`subtabs-search-dropdown${activeMenu === 'filter' ? ' active' : ''}`}>
          <div className="subtabs-search-filter-badges">
            {normalizedFilterOptions.length === 0 && (
              <div className="subtabs-search-dropdown-empty">Нет доступных фильтров</div>
            )}
            {normalizedFilterOptions.map(option => (
              <button
                key={option.label}
                type="button"
                className={`subtabs-search-badge${selectedFilters.includes(option.label) ? ' selected' : ''}`}
                onMouseDown={event => event.preventDefault()}
                onClick={() => toggleFilter(option.label)}
              >
                {option.color && <span className={`subtabs-search-badge-dot ${option.color}`} />}
                {option.label}
              </button>
            ))}
          </div>
        </div>
        <div className={`subtabs-search-dropdown${activeMenu === 'sort' ? ' active' : ''}`}>
          <div className="subtabs-search-sort-options">
            {sortOptions.map(option => (
              <button
                key={option}
                type="button"
                className={`subtabs-search-sort-option${selectedSort === option ? ' selected' : ''}`}
                onMouseDown={event => event.preventDefault()}
                onClick={() => {
                  setSelectedSort(option);
                  onSelectedSortChange?.(option);
                  setActiveMenu(null);
                }}
              >
                {option}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
