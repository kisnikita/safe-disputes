// src/components/Bets/BetsSection.tsx
import React, {
  useState,
  useEffect,
  useRef,
  useCallback,
  forwardRef,
  useImperativeHandle
} from 'react';
import { apiFetch } from '../../utils/apiFetch';
import { BetDetailsModal } from './BetDetailsModal';
import { Spinner } from '@telegram-apps/telegram-ui';
import { HIDE_THRESHOLD } from '../../utils/constants';
import { SubtabsSearch } from '../Layout/SubtabsSearch';
import { ScrollTopHitArea, useDefaultScrollTopHit } from '../Layout/ScrollTopHitArea';
import './BetsSection.css';
import { EmptyState } from '../EmptyState/EmptyState';
import { useTonConnect } from '../../hooks/useTonConnect';

interface Bet {
  id: string;
  title: string;
  amount: number;
  opponent: string;
  status: string;
  result:
    | 'new'
    | 'sent'
    | 'processed'
    | 'answered'
    | 'evidence'
    | 'evidence_answered'
    | 'inspected'
    | 'rejected'
    | 'win'
    | 'lose'
    | 'draw'
  claim: boolean;
}

export interface BetsSectionHandle {
  refresh: () => void;
}

interface Props {
  onModalChange: (open: boolean) => void;
}

const tabs = ['current', 'new', 'passed'] as const;
type Subtab = typeof tabs[number];
const currentBadgeMap: Partial<Record<Bet['result'], { color: string; text: string }>> = {
  processed: { color: 'yellow', text: 'В процессе' },
  answered: { color: 'green', text: 'Результат выбран' },
  evidence: { color: 'red', text: 'Нужны доказательства' },
  evidence_answered: { color: 'gray', text: 'Ожидание доказательств оппонента' },
  inspected: { color: 'orange', text: 'Расследование' },
};
const newBadgeMap: Partial<Record<Bet['result'], { color: string; text: string }>> = {
  new: { color: 'green', text: 'Получено' },
  sent: { color: 'gray', text: 'Отправлено' },
};
const passedBadgeMap: Partial<Record<Bet['result'], { color: string; text: string }>> = {
  rejected: { color: 'gray', text: 'Отменено' },
  win: { color: 'green', text: 'Победа' },
  lose: { color: 'red', text: 'Поражение' },
  draw: { color: 'yellow', text: 'Ничья' },
};

export const BetsSection = forwardRef<BetsSectionHandle, Props>(({onModalChange}, ref) => {
  const { connected } = useTonConnect();
  const [subtab, setSubtab] = useState<Subtab>('current');
  const [betsByTab, setBetsByTab] = useState<Record<Subtab, Bet[]>>({
    current: [],
    new: [],
    passed: [],
  });
  const [loadingByTab, setLoadingByTab] = useState<Record<Subtab, boolean>>({
    current: false,
    new: false,
    passed: false,
  });
  const [hasMoreByTab, setHasMoreByTab] = useState<Record<Subtab, boolean>>({
    current: true,
    new: true,
    passed: true,
  });
  const [fetchedByTab, setFetchedByTab] = useState<Record<Subtab, boolean>>({
    current: false,
    new: false,
    passed: false,
  });
  const cursorRef = useRef<Record<Subtab, string | null>>({
    current: null,
    new: null,
    passed: null,
  });
  const observer = useRef<IntersectionObserver | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [dragOffset, setDragOffset] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isScrolling, setIsScrolling] = useState(false);
  const [pressedCardId, setPressedCardId] = useState<string | null>(null);
  const [transitionEnabled, setTransitionEnabled] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [panelCanScrollByTab, setPanelCanScrollByTab] = useState<Record<Subtab, boolean>>({
    current: true,
    new: true,
    passed: true,
  });
  const scrollTopByTabRef = useRef<Record<Subtab, number>>({
    current: 0,
    new: 0,
    passed: 0,
  });
  const prevSubtabRef = useRef<Subtab>('current');
  const [subtabsDocked, setSubtabsDocked] = useState(false);
  const pullTriggeredRef = useRef(false);
  const subcontentRef = useRef<HTMLDivElement | null>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const dragStartRef = useRef<{ x: number; y: number } | null>(null);
  const dragLastRef = useRef<{ x: number; y: number } | null>(null);
  const widthRef = useRef(0);
  const animatingRef = useRef(false);
  const animationTimeoutRef = useRef<number | null>(null);
  const scrollStateTimeoutRef = useRef<number | null>(null);
  const subtabsPrefetchedRef = useRef(false);
  const gestureLockRef = useRef<'x' | 'y' | null>(null);
  const pressTimeoutRef = useRef<number | null>(null);
  const pressStartRef = useRef<{ x: number; y: number; pointerId: number } | null>(null);
  const tapGuardRef = useRef<{ id: string; cancelled: boolean; startTime: number } | null>(null);
  const tapStateRef = useRef<{
    id: string;
    startX: number;
    startY: number;
    pointerId: number;
    pointerType: string;
    startTime: number;
  } | null>(null);
  const swipeAngleRatio = 1; // ~45° from X axis
  const OPEN_CARD_DELAY_MS = 150;
  const TAP_MOVE_THRESHOLD = 2;
  const TAP_CARD_DELAY_MS = 80;
  const normalizedQuery = searchTerm.trim().toLowerCase();

  const fetchFirstPage = useCallback(async (tab: Subtab) => {
    setFetchedByTab(prev => ({ ...prev, [tab]: true }));
    cursorRef.current[tab] = null;
    setHasMoreByTab(prev => ({ ...prev, [tab]: true }));
    setLoadingByTab(prev => ({ ...prev, [tab]: true }));
    const params = new URLSearchParams({ status: tab, limit: '10' });
    try {
      const res = await apiFetch(`/api/v1/disputes?${params}`);
      if (!res.ok) throw new Error();
      const { data, nextCursor } = (await res.json()) as {
        data: Bet[];
        nextCursor: string | null;
      };
      setBetsByTab(prev => ({ ...prev, [tab]: data }));
      cursorRef.current[tab] = nextCursor;
      setHasMoreByTab(prev => ({ ...prev, [tab]: !!nextCursor }));
    } catch {
      setBetsByTab(prev => ({ ...prev, [tab]: [] }));
      setHasMoreByTab(prev => ({ ...prev, [tab]: false }));
    } finally {
      setLoadingByTab(prev => ({ ...prev, [tab]: false }));
    }
  }, []);

  const loadMore = useCallback(async (tab: Subtab) => {
    if (loadingByTab[tab] || !hasMoreByTab[tab]) return;
    setLoadingByTab(prev => ({ ...prev, [tab]: true }));
    const params = new URLSearchParams({ status: tab, limit: '10' });
    const cursor = cursorRef.current[tab];
    if (cursor) params.set('cursor', cursor);
    try {
      const res = await apiFetch(`/api/v1/disputes?${params}`);
      if (!res.ok) throw new Error();
      const { data, nextCursor } = (await res.json()) as {
        data: Bet[];
        nextCursor: string | null;
      };
      setBetsByTab(prev => ({ ...prev, [tab]: [...prev[tab], ...data] }));
      cursorRef.current[tab] = nextCursor;
      setHasMoreByTab(prev => ({ ...prev, [tab]: !!nextCursor }));
    } catch {
      // ignore
    } finally {
      setLoadingByTab(prev => ({ ...prev, [tab]: false }));
    }
  }, [loadingByTab, hasMoreByTab]);

  useEffect(() => {
    fetchFirstPage('current');
  }, [fetchFirstPage]);

  useImperativeHandle(ref, () => ({ refresh: () => fetchFirstPage(subtab) }));

  useEffect(() => {
    if (!fetchedByTab[subtab] && !loadingByTab[subtab]) {
      fetchFirstPage(subtab);
    }
  }, [subtab, fetchedByTab, loadingByTab, fetchFirstPage]);

  useEffect(() => {
    if (subtabsPrefetchedRef.current) return;
    if (!fetchedByTab.current || loadingByTab.current) return;
    subtabsPrefetchedRef.current = true;
    tabs.forEach(tab => {
      if (tab !== 'current' && !fetchedByTab[tab] && !loadingByTab[tab]) {
        fetchFirstPage(tab);
      }
    });
  }, [fetchedByTab, loadingByTab, fetchFirstPage]);

  useEffect(() => {
    const node = subcontentRef.current;
    if (!node) return;
    const update = () => setContainerWidth(node.getBoundingClientRect().width);
    update();
    if (typeof ResizeObserver !== 'undefined') {
      const ro = new ResizeObserver(update);
      ro.observe(node);
      return () => ro.disconnect();
    }
    window.addEventListener('resize', update);
    return () => window.removeEventListener('resize', update);
  }, []);

  useEffect(() => {
    const node = subcontentRef.current;
    if (!node) return;
    const updateCanScroll = () => {
      const panels = node.querySelectorAll<HTMLElement>('.subcontent-panel');
      setPanelCanScrollByTab({
        current: (panels[0]?.scrollHeight ?? 0) > (panels[0]?.clientHeight ?? 0) + 1,
        new: (panels[1]?.scrollHeight ?? 0) > (panels[1]?.clientHeight ?? 0) + 1,
        passed: (panels[2]?.scrollHeight ?? 0) > (panels[2]?.clientHeight ?? 0) + 1,
      });
    };
    const rafId = requestAnimationFrame(updateCanScroll);
    let ro: ResizeObserver | null = null;
    if (typeof ResizeObserver !== 'undefined') {
      ro = new ResizeObserver(updateCanScroll);
      node.querySelectorAll<HTMLElement>('.subcontent-panel').forEach(panel => ro?.observe(panel));
    }
    return () => {
      cancelAnimationFrame(rafId);
      ro?.disconnect();
    };
  }, [
    containerWidth,
    betsByTab.current.length,
    betsByTab.new.length,
    betsByTab.passed.length,
    normalizedQuery,
    loadingByTab.current,
    loadingByTab.new,
    loadingByTab.passed,
  ]);

  const currentIndex = tabs.indexOf(subtab);
  const width = containerWidth || widthRef.current || 0;
  const trackTranslate = width ? -currentIndex * width + dragOffset : 0;
  const swipeProgress = width ? -dragOffset / width : 0;
  const rawIndicatorIndex = currentIndex + swipeProgress;
  const indicatorIndex = Math.max(0, Math.min(tabs.length - 1, rawIndicatorIndex));
  const visualActiveIndex = Math.round(indicatorIndex);
  const getTargetScrollTopForSwitch = useCallback((fromTab: Subtab, toTab: Subtab) => {
    const fromScrollTop = scrollTopByTabRef.current[fromTab] ?? 0;
    const fromWasDocked = fromScrollTop >= HIDE_THRESHOLD;
    const savedTargetScrollTop = scrollTopByTabRef.current[toTab] ?? 0;
    return fromWasDocked ? Math.max(HIDE_THRESHOLD, savedTargetScrollTop) : 0;
  }, []);
  const syncPanelForTab = useCallback((toTab: Subtab, fromTab: Subtab) => {
    const node = subcontentRef.current;
    if (!node) return;
    const targetScrollTop = getTargetScrollTopForSwitch(fromTab, toTab);
    const tabIndex = tabs.indexOf(toTab);
    const panel = node.querySelectorAll<HTMLElement>('.subcontent-panel')[tabIndex];
    if (panel && Math.abs(panel.scrollTop - targetScrollTop) > 1) {
      panel.scrollTop = targetScrollTop;
    }
  }, [getTargetScrollTopForSwitch]);
  const resetAllTabsScrollState = useCallback(() => {
    tabs.forEach(tab => {
      scrollTopByTabRef.current[tab] = 0;
    });
    const node = subcontentRef.current;
    if (!node) return;
    node.querySelectorAll<HTMLElement>('.subcontent-panel').forEach(panel => {
      if (panel.scrollTop !== 0) {
        panel.scrollTop = 0;
      }
    });
  }, []);
  const resetInactiveTabsScrollState = useCallback((activeTab: Subtab) => {
    tabs.forEach(tab => {
      if (tab !== activeTab) {
        scrollTopByTabRef.current[tab] = 0;
      }
    });
  }, []);

  useEffect(() => {
    const node = subcontentRef.current;
    if (!node) return;
    const panel = node.querySelectorAll<HTMLElement>('.subcontent-panel')[currentIndex];
    const prevTab = prevSubtabRef.current;
    const targetScrollTop = getTargetScrollTopForSwitch(prevTab, subtab);
    if (panel) {
      if (Math.abs(panel.scrollTop - targetScrollTop) > 1) {
        panel.scrollTop = targetScrollTop;
      }
      scrollTopByTabRef.current[subtab] = targetScrollTop;
    }
    setSubtabsDocked(targetScrollTop >= HIDE_THRESHOLD);
    window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: targetScrollTop } }));
    prevSubtabRef.current = subtab;
  }, [subtab, currentIndex, getTargetScrollTopForSwitch]);

  const lastRef = useCallback(
    (node: HTMLDivElement | null) => {
      if (loadingByTab[subtab] || !hasMoreByTab[subtab]) return;
      if (observer.current) observer.current.disconnect();
      observer.current = new IntersectionObserver(entries => {
        if (entries[0].isIntersecting) loadMore(subtab);
      });
      if (node) observer.current.observe(node);
    },
    [loadingByTab, hasMoreByTab, loadMore, subtab]
  );

  const betFilterOptionsByTab = (() => {
    const collect = (source: Partial<Record<Bet['result'], { color: string; text: string }>>) => {
      const seen = new Set<string>();
      return Object.values(source)
        .filter((badge): badge is { color: string; text: string } => badge !== null)
        .filter(badge => {
          if (seen.has(badge.text)) return false;
          seen.add(badge.text);
          return true;
        })
        .map(badge => ({ label: badge.text, color: badge.color }));
    };

    return {
      current: collect(currentBadgeMap),
      new: collect(newBadgeMap),
      passed: collect(passedBadgeMap),
    };
  })();

  const openDetails = (id: string) => {
    if (isDragging) return;
    setSelectedId(id);
    onModalChange(true);
  };
  const closeDetails = () => {
    setSelectedId(null);
    onModalChange(false);
  };

  const clearTapState = () => {
    tapStateRef.current = null;
  };

  const clearPressState = () => {
    if (pressTimeoutRef.current !== null) {
      window.clearTimeout(pressTimeoutRef.current);
    }
    pressTimeoutRef.current = null;
    pressStartRef.current = null;
    setPressedCardId(null);
  };

  const handleCardPointerDown = (
    event: React.PointerEvent<HTMLDivElement>,
    id: string,
    isActive: boolean
  ) => {
    if (!isActive || isDragging) return;
    clearTapState();
    clearPressState();
    tapGuardRef.current = { id, cancelled: false, startTime: Date.now() };
    const isTouchLike =
      event.pointerType !== 'mouse' ||
      (typeof navigator !== 'undefined' && navigator.maxTouchPoints > 0);
    const state = {
      id,
      startX: event.clientX,
      startY: event.clientY,
      pointerId: event.pointerId,
      pointerType: event.pointerType,
      startTime: Date.now(),
    };
    pressStartRef.current = { x: event.clientX, y: event.clientY, pointerId: event.pointerId };
    if (!isScrolling) {
      if (isTouchLike) {
        if (TAP_CARD_DELAY_MS > 0) {
          pressTimeoutRef.current = window.setTimeout(() => {
            setPressedCardId(id);
            pressTimeoutRef.current = null;
          }, TAP_CARD_DELAY_MS);
        } else {
          setPressedCardId(id);
        }
      } else {
        setPressedCardId(id);
      }
    }
    tapStateRef.current = state;
  };

  const handleCardPointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    const state = tapStateRef.current;
    if (!state || state.pointerId !== event.pointerId) return;
    const dx = event.clientX - state.startX;
    const dy = event.clientY - state.startY;
    if (Math.hypot(dx, dy) > TAP_MOVE_THRESHOLD) {
      clearTapState();
      clearPressState();
      if (tapGuardRef.current) {
        tapGuardRef.current.cancelled = true;
      }
    }
    const pressStart = pressStartRef.current;
    if (pressStart && pressStart.pointerId === event.pointerId) {
      const pressDx = event.clientX - pressStart.x;
      const pressDy = event.clientY - pressStart.y;
      if (Math.hypot(pressDx, pressDy) > TAP_MOVE_THRESHOLD) {
        clearPressState();
        if (tapGuardRef.current) {
          tapGuardRef.current.cancelled = true;
        }
      }
    }
  };

  const handleCardPointerUp = (event: React.PointerEvent<HTMLDivElement>, isActive: boolean) => {
    const state = tapStateRef.current;
    if (!state || state.pointerId !== event.pointerId) return;
    clearTapState();
    const releasePress = () => {
      clearPressState();
    };
    if (!isActive || isDragging) return;
    const elapsed = Date.now() - state.startTime;
    const pressDuration = Math.max(0, OPEN_CARD_DELAY_MS - elapsed);
    if (pressDuration > 0) {
      window.setTimeout(releasePress, pressDuration);
    } else {
      releasePress();
    }
  };

  const handleSubtabClick = (nextTab: Subtab) => {
    if (nextTab === subtab) return;
    syncPanelForTab(nextTab, subtab);
    if (animatingRef.current) {
      if (animationTimeoutRef.current !== null) {
        window.clearTimeout(animationTimeoutRef.current);
        animationTimeoutRef.current = null;
      }
      animatingRef.current = false;
      setTransitionEnabled(false);
      setDragOffset(0);
    }
    const nextIndex = tabs.indexOf(nextTab);
    if (nextIndex === -1) return;
    const width = containerWidth || widthRef.current || 0;
    if (!width) {
      setSubtab(nextTab);
      return;
    }
    const offset = (currentIndex - nextIndex) * width;
    setIsDragging(false);
    setTransitionEnabled(true);
    animatingRef.current = true;
    setDragOffset(offset);
    animationTimeoutRef.current = window.setTimeout(() => {
      setSubtab(nextTab);
      setDragOffset(0);
      setTransitionEnabled(false);
      animatingRef.current = false;
      animationTimeoutRef.current = null;
    }, 200);
  };

  const handlePointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (event.pointerType === 'mouse' && event.button !== 0) return;
    if (animatingRef.current) {
      if (animationTimeoutRef.current !== null) {
        window.clearTimeout(animationTimeoutRef.current);
        animationTimeoutRef.current = null;
      }
      animatingRef.current = false;
      setTransitionEnabled(false);
      setDragOffset(0);
    }
    dragStartRef.current = { x: event.clientX, y: event.clientY };
    dragLastRef.current = { x: event.clientX, y: event.clientY };
    gestureLockRef.current = null;
    pullTriggeredRef.current = false;
    setIsDragging(false);
    widthRef.current = event.currentTarget.getBoundingClientRect().width;
    if (event.pointerType !== 'mouse') {
      event.currentTarget.setPointerCapture(event.pointerId);
    }
    const next = tabs[currentIndex + 1];
    const prev = tabs[currentIndex - 1];
    if (next) {
      syncPanelForTab(next, subtab);
    }
    if (prev) {
      syncPanelForTab(prev, subtab);
    }
    if (next && betsByTab[next].length === 0 && !loadingByTab[next]) {
      fetchFirstPage(next);
    }
    if (prev && betsByTab[prev].length === 0 && !loadingByTab[prev]) {
      fetchFirstPage(prev);
    }
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    const start = dragStartRef.current;
    if (!start) return;
    dragLastRef.current = { x: event.clientX, y: event.clientY };
    const deltaX = event.clientX - start.x;
    const deltaY = event.clientY - start.y;
    if (!gestureLockRef.current) {
      if (Math.abs(deltaX) > 8 && Math.abs(deltaY) <= Math.abs(deltaX) * swipeAngleRatio) {
        gestureLockRef.current = 'x';
      } else if (Math.abs(deltaY) > 12) {
        gestureLockRef.current = 'y';
      }
    }
    if (gestureLockRef.current === 'y' && !pullTriggeredRef.current && subtabsDocked && deltaY > 30) {
      const node = subcontentRef.current;
      const panel = node?.querySelectorAll<HTMLElement>('.subcontent-panel')[currentIndex];
      const canScroll = panel ? panel.scrollHeight > panel.clientHeight + 1 : true;
      if (!canScroll) {
        pullTriggeredRef.current = true;
        resetAllTabsScrollState();
        setSubtabsDocked(false);
        requestAnimationFrame(() => {
          window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: 0 } }));
        });
      }
    }
    if (gestureLockRef.current !== 'x') return;
    if (!event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.setPointerCapture(event.pointerId);
    }
    if (Math.abs(deltaX) < 6) return;
    if (!isDragging) setIsDragging(true);
    event.preventDefault();
    setTransitionEnabled(false);
    const width = widthRef.current || event.currentTarget.getBoundingClientRect().width;
    const limit = width ? width * 1.1 : 9999;
    const clamped = Math.max(-limit, Math.min(limit, deltaX));
    setDragOffset(clamped);
  };

  const handlePointerUp = (event: React.PointerEvent<HTMLDivElement>) => {
    if (animatingRef.current) return;
    const start = dragStartRef.current;
    const end = dragLastRef.current;
    dragStartRef.current = null;
    dragLastRef.current = null;
    gestureLockRef.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    if (!start || !end) {
      setIsDragging(false);
      setDragOffset(0);
      return;
    }

    const deltaX = end.x - start.x;
    const deltaY = end.y - start.y;
    setIsDragging(false);
    const width = widthRef.current || event.currentTarget.getBoundingClientRect().width;
    const threshold = width * 0.10;
    if (Math.abs(deltaX) < threshold || Math.abs(deltaX) < Math.abs(deltaY)) {
      setTransitionEnabled(true);
      setDragOffset(0);
      animationTimeoutRef.current = window.setTimeout(() => {
        setTransitionEnabled(false);
        animationTimeoutRef.current = null;
      }, 260);
      return;
    }

    const currentIndex = tabs.indexOf(subtab);
    const nextIndex = deltaX < 0 ? currentIndex + 1 : currentIndex - 1;
    if (nextIndex < 0 || nextIndex >= tabs.length) {
      setTransitionEnabled(true);
      setDragOffset(0);
      animationTimeoutRef.current = window.setTimeout(() => {
        setTransitionEnabled(false);
        animationTimeoutRef.current = null;
      }, 260);
      return;
    }

    setTransitionEnabled(true);
    animatingRef.current = true;
    syncPanelForTab(tabs[nextIndex], subtab);
    setSubtab(tabs[nextIndex]);
    setDragOffset(0);
    animationTimeoutRef.current = window.setTimeout(() => {
      setTransitionEnabled(false);
      animatingRef.current = false;
      animationTimeoutRef.current = null;
    }, 260);
  };

  const handlePanelScroll = (index: number, event: React.UIEvent<HTMLDivElement>) => {
    if (index !== currentIndex) return;
    const scrollTop = event.currentTarget.scrollTop;
    const prevScrollTop = scrollTopByTabRef.current[subtab] ?? 0;
    const wasDocked = prevScrollTop >= HIDE_THRESHOLD;
    const isDockedNow = scrollTop >= HIDE_THRESHOLD;
    scrollTopByTabRef.current[subtab] = scrollTop;
    if (wasDocked && !isDockedNow) {
      resetAllTabsScrollState();
      setSubtabsDocked(false);
      window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: 0 } }));
    } else {
      setSubtabsDocked(isDockedNow);
    }
    setIsScrolling(true);
    if (scrollStateTimeoutRef.current !== null) {
      window.clearTimeout(scrollStateTimeoutRef.current);
    }
    scrollStateTimeoutRef.current = window.setTimeout(() => {
      setIsScrolling(false);
      scrollStateTimeoutRef.current = null;
    }, 120);
  };

  const scrollActivePanelToTop = useDefaultScrollTopHit(subcontentRef, currentIndex, () => {
    resetInactiveTabsScrollState(subtab);
    scrollTopByTabRef.current[subtab] = 0;
    setSubtabsDocked(false);
  });

    return (
      <>
        <div className={`bets-section${subtabsDocked ? ' subtabs-docked' : ''}`}>
          <ScrollTopHitArea enabled={subtabsDocked} onHit={scrollActivePanelToTop} />
          <div className="subtabs">
            {tabs.map((buttonTab, tabIndex) => (
              <button
                key={buttonTab}
                onClick={() => handleSubtabClick(buttonTab)}
                className={visualActiveIndex === tabIndex ? 'active' : ''}
              >
                {buttonTab === 'current'
                  ? 'Текущие'
                  : buttonTab === 'new'
                  ? 'Новые'
                  : 'Прошедшие'}
              </button>
            ))}
            <div
              className="subtabs-indicator"
              style={{
                width: `${97 / tabs.length}%`,
                transform: `translateX(${indicatorIndex * 99}%)`,
                transition: isDragging ? 'none' : 'transform 0.3s ease-out',
              }}
            />
          </div>
          <SubtabsSearch
            value={searchTerm}
            onChange={setSearchTerm}
            placeholder="Поиск пари"
            hidden={subtabsDocked}
            blurOnSwipe={isDragging}
            filterOptions={betFilterOptionsByTab[subtab]}
            resetKey={subtab}
          />

          <div
            className={`subcontent${isDragging ? ' swiping-x' : ''}${isScrolling ? ' scrolling' : ''}`}
            ref={subcontentRef}
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            onPointerCancel={handlePointerUp}
          >
            <div
              className="subcontent-track"
              style={{
                transform: `translateX(${trackTranslate}px)`,
                transition: transitionEnabled && !isDragging ? 'transform 0.26s ease-out' : 'none',
              }}
            >
              {tabs.map((tab, tabIndex) => {
                const list = betsByTab[tab];
                const filteredList = normalizedQuery
                  ? list.filter(bet =>`${bet.title} ${bet.opponent}`.toLowerCase().includes(normalizedQuery))
                  : list;
                const isActive = tab === subtab;
                const panelTopOffset = subtabsDocked && !panelCanScrollByTab[tab] ? HIDE_THRESHOLD : 0;
                const isLoading = loadingByTab[tab];
                const isEmpty = !isLoading && filteredList.length === 0;
                const isSearchEmpty = Boolean(normalizedQuery);
                const showNewTabHint = tab === 'new' && !isSearchEmpty;
                const showCurrentHint = tab === 'current' && !isSearchEmpty;
                const showPassedMessageOnly = tab === 'passed' && !isSearchEmpty;
                const hintIconDirection =
                  isSearchEmpty && tab === 'new'
                    ? 'both'
                    : tab === 'current'
                    ? 'right'
                    : tab === 'passed'
                    ? 'left'
                    : 'up';
                const emptyMessage = isSearchEmpty
                  ? 'Ничего не найдено'
                  : showCurrentHint
                  ? 'Примите пари, чтобы оно появилось здесь'
                  : showPassedMessageOnly
                  ? 'Здесь будут ваши завершённые и отменённые пари'
                  : connected
                  ? 'Создайте пари, чтобы оно появилось здесь'
                  : 'Подключите кошелек, чтобы создавать и принимать пари';
                const emptyHint = showNewTabHint
                  ? subtabsDocked
                    ? 'Прокрутите вверх, чтобы увидеть кнопки действий'
                    : 'Кнопки действий доступны в верхней части экрана'
                  : isSearchEmpty
                  ? 'Проверьте результат в других вкладках'
                  : showCurrentHint
                  ? 'Принять пари можно на вкладке Новые'
                  : undefined;
                return (
                  <div
                    key={tab}
                    className="subcontent-panel"
                    style={{
                      paddingTop: `calc(var(--subtabs-spacing) - ${panelTopOffset}px)`,
                      scrollPaddingTop: `calc(var(--subtabs-spacing) - ${panelTopOffset}px)`,
                      overflow: isDragging || !panelCanScrollByTab[tab] ? 'hidden' : 'auto',
                    }}
                    onScroll={event => handlePanelScroll(tabIndex, event)}
                  >
                    {filteredList.map((bet, idx) => {
                      const isLast = idx === filteredList.length - 1;
                      const badge =
                        tab === 'current'
                          ? currentBadgeMap[bet.result] ?? null
                          : tab === 'new'
                          ? newBadgeMap[bet.result] ?? null
                          : tab === 'passed'
                          ? passedBadgeMap[bet.result] ?? null
                          : null;

                      return (
                        <div
                          key={bet.id}
                          ref={isActive && isLast ? lastRef : null}
                          className={`bet-card${pressedCardId === bet.id ? ' pressed' : ''}`}
                          onClick={() => {
                            const isTouchLike =
                              typeof navigator !== 'undefined' && navigator.maxTouchPoints > 0;
                            if (!isActive) return;
                            if (!isTouchLike) {
                              openDetails(bet.id);
                              return;
                            }
                            const guard = tapGuardRef.current;
                            if (!guard || guard.id !== bet.id || guard.cancelled) {
                              tapGuardRef.current = null;
                              return;
                            }
                            const delay = Math.max(0, OPEN_CARD_DELAY_MS - (Date.now() - guard.startTime));
                            tapGuardRef.current = null;
                            window.setTimeout(() => {
                              if (!isDragging) {
                                openDetails(bet.id);
                              }
                            }, delay);
                          }}
                          onPointerDown={event => handleCardPointerDown(event, bet.id, isActive)}
                          onPointerMove={handleCardPointerMove}
                          onPointerUp={event => handleCardPointerUp(event, isActive)}
                          onPointerCancel={() => {
                            clearTapState();
                            clearPressState();
                            if (tapGuardRef.current) {
                              tapGuardRef.current.cancelled = true;
                            }
                          }}
                        >
                          <h4>{bet.title}</h4>
                          <p>Ставка: {bet.amount} TON</p>
                          <p>Оппонент: {bet.opponent}</p>

                          {badge && (
                            <div className="result-badge">
                              <span className={`dot ${badge.color}`} />
                              {badge.text}
                            </div>
                          )}
                          {tab === 'passed' && bet.claim && bet.result !== "win" && (
                            <div className="claim-label">Возврат доступен</div>
                          )}
                          {tab === 'passed' && bet.claim && bet.result === "win" && (
                            <div className="claim-label">Награда доступна</div>
                          )}
                        </div>
                      );
                    })}

                    {isLoading && (
                      <div className="loading">
                        <Spinner size="m" className="spinner"/>
                      </div>
                    )}
                    {isEmpty && (
                      <EmptyState
                        message={emptyMessage}
                        variant={normalizedQuery ? 'notFound' : 'empty'}
                        hint={emptyHint}
                        hintIconDirection={hintIconDirection}
                        onHintClick={showNewTabHint && subtabsDocked ? scrollActivePanelToTop : undefined}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {!!selectedId && (
          <BetDetailsModal
            id={selectedId}
            onClose={closeDetails}
            onCompleted={() => {
              fetchFirstPage(subtab);
            }}
            showActions={subtab === 'new'}
            showResultActions={subtab === 'current'}
            showClaimActions={subtab === 'passed'}
          />
        )}
      </>
    );
  }
);
