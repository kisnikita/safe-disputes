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
import { Loader } from '../Loader/Loader';
import { HIDE_THRESHOLD } from '../../utils/constants';
import './BetsSection.css';

interface Bet {
  id: string;
  title: string;
  amount: number;
  opponent: string;
  status: string;
  result:
    | 'evidence'
    | 'sent'
    | 'answered'
    | 'inspected'
    | 'win'
    | 'lose'
    | 'draw'
    | 'rejected'
    | 'processed'
    | 'evidence_answered';
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

export const BetsSection = forwardRef<BetsSectionHandle, Props>(({onModalChange}, ref) => {
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

  const currentIndex = tabs.indexOf(subtab);
  const width = containerWidth || widthRef.current || 0;
  const trackTranslate = width ? -currentIndex * width + dragOffset : 0;
  const swipeProgress = width ? -dragOffset / width : 0;
  const rawIndicatorIndex = currentIndex + swipeProgress;
  const indicatorIndex = Math.max(0, Math.min(tabs.length - 1, rawIndicatorIndex));
  const visualActiveIndex = Math.round(indicatorIndex);

  useEffect(() => {
    const node = subcontentRef.current;
    if (!node) return;
    const panel = node.querySelectorAll<HTMLElement>('.subcontent-panel')[currentIndex];
    const prevTab = prevSubtabRef.current;
    const prevScrollTop = scrollTopByTabRef.current[prevTab] ?? 0;
    const wasDocked = prevScrollTop >= HIDE_THRESHOLD;
    const savedScrollTop = scrollTopByTabRef.current[subtab] ?? 0;
    const targetScrollTop = wasDocked ? Math.max(HIDE_THRESHOLD, savedScrollTop) : 0;
    if (panel) {
      requestAnimationFrame(() => {
        panel.scrollTop = targetScrollTop;
        scrollTopByTabRef.current[subtab] = targetScrollTop;
      });
    }
    setSubtabsDocked(targetScrollTop >= HIDE_THRESHOLD);
    window.dispatchEvent(new CustomEvent('subtab-scroll-sync', { detail: { scrollTop: targetScrollTop } }));
    prevSubtabRef.current = subtab;
  }, [subtab, currentIndex]);

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

  // бейджи для вкладки «Текущие»
  const getCurrentBadge = (result: Bet['result']) => {
    switch (result) {
      case 'evidence':
        return { color: 'red', text: 'Нужны доказательства' };
      case 'sent':
        return { color: 'gray', text: 'Ожидание ответа' };
      case 'answered':
        return { color: 'green', text: 'Результат выбран' };
      case 'inspected':
        return { color: 'yellow', text: 'Расследование' };
      case 'processed':
        return { color: 'yellow', text: 'В процессе' };
      case 'evidence_answered':
        return { color: 'gray', text: 'Ожидание доказательств оппонента' };
      default:
        return null;
    }
  };

  // бейджи для вкладки «Прошедшие»
  const getPassedBadge = (result: Bet['result']) => {
    switch (result) {
      case 'win':
        return { color: 'green', text: 'Победа' };
      case 'lose':
        return { color: 'red', text: 'Поражение' };
      case 'draw':
        return { color: 'yellow', text: 'Ничья' };
      case 'rejected':
        return { color: 'gray', text: 'Отменено' };
      default:
        return null;
    }
  };

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
        if (panel) panel.scrollTop = 0;
        scrollTopByTabRef.current[subtab] = 0;
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
    scrollTopByTabRef.current[subtab] = scrollTop;
    setSubtabsDocked(scrollTop >= HIDE_THRESHOLD);
    setIsScrolling(true);
    if (scrollStateTimeoutRef.current !== null) {
      window.clearTimeout(scrollStateTimeoutRef.current);
    }
    scrollStateTimeoutRef.current = window.setTimeout(() => {
      setIsScrolling(false);
      scrollStateTimeoutRef.current = null;
    }, 120);
  };

    return (
      <>
        <div className={`bets-section${subtabsDocked ? ' subtabs-docked' : ''}`}>
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
                width: `${101 / tabs.length}%`,
                transform: `translateX(${indicatorIndex * 110}%)`,
                transition: isDragging ? 'none' : 'transform 0.2s ease-out',
              }}
            />
          </div>

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
                const isActive = tab === subtab;
                const isLoading = loadingByTab[tab];
                const isEmpty = !isLoading && list.length === 0;
                return (
                  <div
                    key={tab}
                    className="subcontent-panel"
                    onScroll={event => handlePanelScroll(tabIndex, event)}
                  >
                    {list.map((bet, idx) => {
                      const isLast = idx === list.length - 1;
                      const badge =
                        tab === 'current'
                          ? getCurrentBadge(bet.result)
                          : tab === 'passed'
                          ? getPassedBadge(bet.result)
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
                            <div
                              className="result-badge"
                              style={{ borderColor: badge.color }}
                            >
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
                        <Loader />
                      </div>
                    )}
                    {isEmpty &&  <div className="empty-message">Тут пока пусто</div>}
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
