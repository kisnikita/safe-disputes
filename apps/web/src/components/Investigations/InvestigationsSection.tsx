// src/components/Investigations/InvestigationsSection.tsx
import React, {
  useState,
  useEffect,
  useRef,
  useCallback,
  forwardRef,
  useImperativeHandle
} from 'react';
import { apiFetch } from '../../utils/apiFetch';
import './InvestigationsSection.css';
import { InvestigationDetailsModal } from './InvestigationDetailsModal';
import { RatingButton } from '../Layout/RatingButton';
import { Spinner } from '@telegram-apps/telegram-ui';
import { HIDE_THRESHOLD } from '../../utils/constants';
import { SubtabsSearch } from '../Layout/SubtabsSearch';
import { ScrollTopHitArea, useDefaultScrollTopHit } from '../Layout/ScrollTopHitArea';
import { EmptyState } from '../EmptyState/EmptyState';
import { useTonConnect } from '../../hooks/useTonConnect';

interface Investigation {
  id: string;
  disputeID: string;
  title: string;
  status: 'current' | 'passed';
  createdAt?: string;
  endsAt: string; // ISO date string
  total?: number;
  result: 'new' | 'sent' | 'correct' | 'incorrect';
  vote?: string;
}

interface TopUser {
  username: string;
  rating: number;
}

export interface InvestigationsSectionHandle {
  refresh: () => void;
  scrollToTop: () => void;
}

interface Props {
  onModalChange: (open: boolean) => void;
}

const tabs = ['current', 'passed'] as const;
type Subtab = typeof tabs[number];
const currentInvestigationBadgeMap: Partial<Record<Investigation['result'], { color: string; text: string }>> = {
  new: { color: 'red', text: 'Новое' },
  sent: { color: 'green', text: 'Голос отдан' },
};
const passedInvestigationBadgeMap: Partial<Record<Investigation['result'], { color: string; text: string }>> = {
  correct: { color: 'green', text: 'Верно' },
  incorrect: { color: 'red', text: 'Неверно' },
};
const investigationFilterOptionsByTab: Record<Subtab, Array<{ label: string; color: string }>> = {
  current: Object.values(currentInvestigationBadgeMap).map(badge => ({
    label: badge?.text ?? '',
    color: badge?.color ?? 'gray',
  })),
  passed: Object.values(passedInvestigationBadgeMap).map(badge => ({
    label: badge?.text ?? '',
    color: badge?.color ?? 'gray',
  })),
};
const investigationSortOptionsByTab: Record<Subtab, string[]> = {
  current: ['Завершающиеся', 'Крупные'],
  passed: ['Последние', 'Крупные'],
};

export const InvestigationsSection = forwardRef<InvestigationsSectionHandle, Props>(
  ({ onModalChange }, ref) => {
    const [subtab, setSubtab] = useState<Subtab>('current');
    const [itemsByTab, setItemsByTab] = useState<Record<Subtab, Investigation[]>>({
      current: [],
      passed: [],
    });
    const [loadingByTab, setLoadingByTab] = useState<Record<Subtab, boolean>>({
      current: false,
      passed: false,
    });
    const [hasMoreByTab, setHasMoreByTab] = useState<Record<Subtab, boolean>>({
      current: true,
      passed: true,
    });
    const [fetchedByTab, setFetchedByTab] = useState<Record<Subtab, boolean>>({
      current: false,
      passed: false,
    });
    const [showRating, setShowRating] = useState(false);
    const [topUsers, setTopUsers] = useState<TopUser[]>([]);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const { connected } = useTonConnect();
    const cursorRef = useRef<Record<Subtab, string | null>>({
      current: null,
      passed: null,
    });
    const observer = useRef<IntersectionObserver | null>(null);
    const [dragOffset, setDragOffset] = useState(0);
    const [isDragging, setIsDragging] = useState(false);
    const [isScrolling, setIsScrolling] = useState(false);
    const [pressedCardId, setPressedCardId] = useState<string | null>(null);
    const [transitionEnabled, setTransitionEnabled] = useState(false);
    const [searchTerm, setSearchTerm] = useState('');
    const [selectedFiltersByTab, setSelectedFiltersByTab] = useState<Record<Subtab, string[]>>({
      current: [],
      passed: [],
    });
    const [selectedSortByTab, setSelectedSortByTab] = useState<Record<Subtab, string>>({
      current: investigationSortOptionsByTab.current[0],
      passed: investigationSortOptionsByTab.passed[0],
    });
    const [panelCanScrollByTab, setPanelCanScrollByTab] = useState<Record<Subtab, boolean>>({
      current: true,
      passed: true,
    });
    const scrollTopByTabRef = useRef<Record<Subtab, number>>({
      current: 0,
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

    const fetchInvestigations = useCallback(async (tab: Subtab) => {
      setFetchedByTab(prev => ({ ...prev, [tab]: true }));
      cursorRef.current[tab] = null;
      setHasMoreByTab(prev => ({ ...prev, [tab]: true }));
      setLoadingByTab(prev => ({ ...prev, [tab]: true }));
      const params = new URLSearchParams({ status: tab, limit: '10' });
      try {
        const res = await apiFetch(`/api/v1/investigations?${params}`);
        if (!res.ok) throw new Error();
        const { data, nextCursor } = (await res.json()) as {
          data: Investigation[];
          nextCursor: string | null;
        };
        setItemsByTab(prev => ({ ...prev, [tab]: data }));
        cursorRef.current[tab] = nextCursor;
        setHasMoreByTab(prev => ({ ...prev, [tab]: !!nextCursor }));
      } catch {
        setItemsByTab(prev => ({ ...prev, [tab]: [] }));
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
        const res = await apiFetch(`/api/v1/investigations?${params}`);
        if (!res.ok) throw new Error();
        const { data, nextCursor } = (await res.json()) as {
          data: Investigation[];
          nextCursor: string | null;
        };
        setItemsByTab(prev => ({ ...prev, [tab]: [...prev[tab], ...data] }));
        cursorRef.current[tab] = nextCursor;
        setHasMoreByTab(prev => ({ ...prev, [tab]: !!nextCursor }));
      } catch {
        // ignore
      } finally {
        setLoadingByTab(prev => ({ ...prev, [tab]: false }));
      }
    }, [loadingByTab, hasMoreByTab]);

    const fetchTopUsers = useCallback(async () => {
      try {
        const res = await apiFetch(`/api/v1/users/top`);
        if (!res.ok) throw new Error();
        const { data } = (await res.json()) as { data: TopUser[] };
        setTopUsers(data.slice(0, 100));
      } catch {
        setTopUsers([]);
      }
    }, []);

    useEffect(() => {
      fetchInvestigations('current');
    }, [fetchInvestigations]);

    useEffect(() => {
      if (!fetchedByTab[subtab] && !loadingByTab[subtab]) {
        fetchInvestigations(subtab);
      }
    }, [subtab, fetchedByTab, loadingByTab, fetchInvestigations]);

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
          passed: (panels[1]?.scrollHeight ?? 0) > (panels[1]?.clientHeight ?? 0) + 1,
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
      itemsByTab.current.length,
      itemsByTab.passed.length,
      normalizedQuery,
      selectedFiltersByTab.current.join('|'),
      selectedFiltersByTab.passed.join('|'),
      selectedSortByTab.current,
      selectedSortByTab.passed,
      loadingByTab.current,
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
      }, 260);
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
      if (next && itemsByTab[next].length === 0 && !loadingByTab[next]) {
        fetchInvestigations(next);
      }
      if (prev && itemsByTab[prev].length === 0 && !loadingByTab[prev]) {
        fetchInvestigations(prev);
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
      if (Math.abs(deltaX) < 6) return;
      if (!isDragging) setIsDragging(true);
      event.preventDefault();
      setTransitionEnabled(false);
      const width = widthRef.current || event.currentTarget.getBoundingClientRect().width;
      const limit = width ? width * 1.2 : 9999;
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

    useImperativeHandle(ref, () => ({
      refresh: () => fetchInvestigations(subtab),
      scrollToTop: scrollActivePanelToTop,
    }), [fetchInvestigations, scrollActivePanelToTop, subtab]);

  const sortInvestigations = (list: Investigation[], tab: Subtab, sortOption: string) => {
    if (sortOption === 'Завершающиеся' && tab === 'current') {
      return list
        .map((inv, index) => ({ inv, index }))
        .sort((a, b) => {
          const aEndsAt = Date.parse(a.inv.endsAt);
          const bEndsAt = Date.parse(b.inv.endsAt);
          if (!Number.isNaN(aEndsAt) && !Number.isNaN(bEndsAt) && aEndsAt !== bEndsAt) {
            return aEndsAt - bEndsAt;
          }
          return a.index - b.index;
        })
        .map(({ inv }) => inv);
    }
    if (sortOption === 'Крупные') {
      return list
        .map((inv, index) => ({ inv, index }))
        .sort((a, b) => {
          const aTotal = a.inv.total ?? 0;
          const bTotal = b.inv.total ?? 0;
          if (bTotal !== aTotal) {
            return bTotal - aTotal;
          }
          return a.index - b.index;
        })
        .map(({ inv }) => inv);
    }
    if (sortOption === 'Последние') {
      return list
        .map((inv, index) => ({ inv, index }))
        .sort((a, b) => {
          const aCreatedAt = a.inv.createdAt ? Date.parse(a.inv.createdAt) : Number.NaN;
          const bCreatedAt = b.inv.createdAt ? Date.parse(b.inv.createdAt) : Number.NaN;
          if (!Number.isNaN(aCreatedAt) && !Number.isNaN(bCreatedAt) && bCreatedAt !== aCreatedAt) {
            return bCreatedAt - aCreatedAt;
          }
          return a.index - b.index;
        })
        .map(({ inv }) => inv);
    }
    return list;
  };

  const getTimeRemaining = (endsAt: string) => {
  // Парсим время окончания как UTC-момент
  const endMs = Date.parse(endsAt);

  // Считаем, сколько минут нужно добавить к локальному времени,
  // чтобы получить UTC+3:
  //   локальный оффсет = new Date().getTimezoneOffset() в минутах (от UTC до локали)
  //   нам нужно получить смещение от локали до UTC+3 → 180 мин - локальный оффсет
  const offsetToUTCPlus3Min = -new Date().getTimezoneOffset();
  // текущее «UTC+3» в миллисекундах
  const nowUtcPlus3 = Date.now() + offsetToUTCPlus3Min * 60_000;

  const diff = endMs - nowUtcPlus3;
  if (diff <= 0) return 'завершено';

  const days = Math.floor(diff / (1000 * 60 * 60 * 24));
  const hours = Math.floor((diff / (1000 * 60 * 60)) % 24);
  const minutes = Math.floor((diff / (1000 * 60)) % 60);
  return `${days}д ${hours}ч ${minutes}м`;
};

    return (
      <>
        <div className={`investigations-section${subtabsDocked ? ' subtabs-docked' : ''}`}>
          <ScrollTopHitArea enabled={subtabsDocked && !showRating} onHit={scrollActivePanelToTop} />
          <div className="header">
            <RatingButton
              onClick={() => {
                setShowRating(prev => {
                  if (!prev) fetchTopUsers();
                  return !prev;
                });
              }}
            />
          </div>

          {showRating ? (
            <div className="rating-panel">
              <h3>Топ 100 пользователей</h3>
              <ul>
                {topUsers.map((user, idx) => (
                  <li key={user.username}>
                    <span>{idx + 1}. {user.username}</span>
                    <span className="rating-value">{user.rating}</span>
                  </li>
                ))}
                {topUsers.length === 0 && <li>Нет данных</li>}
              </ul>
            </div>
          ) : (
            <>
              <div className="subtabs">
                {tabs.map((buttonTab, tabIndex) => (
                  <button
                    key={buttonTab}
                    onClick={() => handleSubtabClick(buttonTab)}
                    className={visualActiveIndex === tabIndex ? 'active' : ''}
                  >
                    {buttonTab === 'current' ? 'Текущие' : 'Прошедшие'}
                  </button>
                ))}
                <div
                  className="subtabs-indicator"
                  style={{
                    width: `${102 / tabs.length}%`,
                    transform: `translateX(${indicatorIndex * 90}%)`,
                    transition: isDragging ? 'none' : 'transform 0.3s ease-out',
                  }}
                />
              </div>
              <SubtabsSearch
                value={searchTerm}
                onChange={setSearchTerm}
                placeholder="Поиск расследований"
                hidden={subtabsDocked}
                blurOnSwipe={isDragging}
                filterOptions={investigationFilterOptionsByTab[subtab]}
                sortOptions={investigationSortOptionsByTab[subtab]}
                selectedFilters={selectedFiltersByTab[subtab]}
                onSelectedFiltersChange={filters =>
                  setSelectedFiltersByTab(prev => ({ ...prev, [subtab]: filters }))
                }
                selectedSort={selectedSortByTab[subtab]}
                onSelectedSortChange={sort =>
                  setSelectedSortByTab(prev => ({ ...prev, [subtab]: sort }))
                }
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
                    transition: transitionEnabled && !isDragging ? 'transform 0.3s ease-out' : 'none',
                  }}
                >
                  {tabs.map((tab, tabIndex) => {
                    const list = itemsByTab[tab];
                    const searchFilteredList = normalizedQuery
                      ? list.filter(inv =>`${inv.title}`.toLowerCase().includes(normalizedQuery))
                      : list;
                    const activeFilters = selectedFiltersByTab[tab];
                    const filteredList = activeFilters.length
                      ? searchFilteredList.filter(inv => {
                          const badge =
                            tab === 'current'
                              ? currentInvestigationBadgeMap[inv.result] ?? null
                              : passedInvestigationBadgeMap[inv.result] ?? null;
                          return !!badge && activeFilters.includes(badge.text);
                        })
                      : searchFilteredList;
                    const sortedList = sortInvestigations(filteredList, tab, selectedSortByTab[tab]);
                    const isActive = tab === subtab;
                    const panelTopOffset = subtabsDocked && !panelCanScrollByTab[tab] ? HIDE_THRESHOLD : 0;
                    const isLoading = loadingByTab[tab];
                    const isEmpty = !isLoading && sortedList.length === 0;
                    const hasSearch = Boolean(normalizedQuery);
                    const hasActiveFilters = activeFilters.length > 0;
                    const isNotFoundState = hasSearch || hasActiveFilters;
                    const emptyMessage = isNotFoundState
                      ? 'Ничего не найдено'
                      : tab === 'current'
                      ? 'Расследований пока нет'
                      : 'Здесь будут завершённые расследования, в которых вы принимали участие';
                    const emptyHint = hasSearch
                      ? 'Проверьте результат в других вкладках'
                      : tab === 'current'
                      ? 'Увеличивайте рейтинг, чтобы повысить вероятность получения расследований'
                      : 'Принять участие можно на вкладке Текущие';
                    const hintIconDirection =
                      tab === 'current' && !hasSearch ? 'none' : tab === 'current' ? 'right' : 'left';
                    return (
                      <div
                        key={tab}
                        className="subcontent-panel"
                        style={{
                          paddingTop: `calc(var(--subtabs-spacing) - ${panelTopOffset}px)`,
                          scrollPaddingTop: `calc(var(--subtabs-spacing) - ${panelTopOffset}px)`,
                          overflow: isDragging || !panelCanScrollByTab[tab] ? 'hidden' : '',
                        }}
                        onScroll={event => handlePanelScroll(tabIndex, event)}
                      >
                        {sortedList.map((inv, idx) => {
                          const isLast = idx === sortedList.length - 1;
                          const badge = tab === 'current' ? 
                          currentInvestigationBadgeMap[inv.result] ?? null : 
                          passedInvestigationBadgeMap[inv.result] ?? null;

                          return (
                            <div
                              key={inv.disputeID}
                              ref={isActive && isLast ? lastRef : null}
                              className={`investigation-card${pressedCardId === inv.id ? ' pressed' : ''}`}
                              onClick={() => {
                                const isTouchLike =
                                  typeof navigator !== 'undefined' && navigator.maxTouchPoints > 0;
                                if (!isActive) return;
                                if (!isTouchLike) {
                                  openDetails(inv.id);
                                  return;
                                }
                                const guard = tapGuardRef.current;
                                if (!guard || guard.id !== inv.id || guard.cancelled) {
                                  tapGuardRef.current = null;
                                  return;
                                }
                                const delay = Math.max(0, OPEN_CARD_DELAY_MS - (Date.now() - guard.startTime));
                                tapGuardRef.current = null;
                                window.setTimeout(() => {
                                  if (!isDragging) {
                                    openDetails(inv.id);
                                  }
                                }, delay);
                              }}
                              onPointerDown={event => handleCardPointerDown(event, inv.id, isActive)}
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
                              <h4>{inv.title}</h4>
                              <p>Окончание через: {getTimeRemaining(inv.endsAt)}</p>
                              {inv.vote && <p>Ваш голос: {inv.vote == 'p1' ? 'Пользователь 1' : "Пользователь 2"}</p>}

                              {badge && (
                                <div className="result-badge" style={{ borderColor: badge.color }}>
                                  <span className={`dot ${badge.color}`} />
                                  {badge.text}
                                </div>
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
                            variant={isNotFoundState ? 'notFound' : 'empty'}
                            hint={emptyHint}
                            hintIconDirection={hintIconDirection}
                          />
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </>
          )}

          {!!selectedId && (
            <InvestigationDetailsModal
              id={selectedId}
              onClose={closeDetails}
              onCompleted={() => {
                closeDetails();
                fetchInvestigations(subtab);
              }}
            />
          )}
        </div>
      </>
    );
  }
);
