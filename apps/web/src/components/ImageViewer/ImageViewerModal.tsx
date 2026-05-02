import React, { useEffect, useRef, useState, useCallback } from 'react';
import { createPortal } from 'react-dom';
import './ImageViewerModal.css';

interface Props {
  isOpen: boolean;
  src: string | null;
  alt?: string;
  onClose: () => void;
}

type Point = { x: number; y: number };

const MIN_SCALE = 1;
const MAX_SCALE = 4;

export const ImageViewerModal: React.FC<Props> = ({ isOpen, src, alt = 'Изображение', onClose }) => {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const pointersRef = useRef<Map<number, Point>>(new Map());
  const panStartRef = useRef<{ point: Point; offset: Point } | null>(null);
  const pinchStartRef = useRef<{ distance: number; scale: number; midpoint: Point; offset: Point } | null>(null);
  const lastTapRef = useRef<number>(0);

  const [scale, setScale] = useState<number>(1);
  const [offset, setOffset] = useState<Point>({ x: 0, y: 0 });

  const clampOffset = useCallback((nextOffset: Point, nextScale: number): Point => {
    const container = contentRef.current;
    if (!container || nextScale <= 1) return { x: 0, y: 0 };

    const maxX = ((nextScale - 1) * container.clientWidth) / 2;
    const maxY = ((nextScale - 1) * container.clientHeight) / 2;

    return {
      x: Math.max(-maxX, Math.min(maxX, nextOffset.x)),
      y: Math.max(-maxY, Math.min(maxY, nextOffset.y)),
    };
  }, []);

  const resetTransform = useCallback(() => {
    setScale(1);
    setOffset({ x: 0, y: 0 });
    pointersRef.current.clear();
    panStartRef.current = null;
    pinchStartRef.current = null;
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    resetTransform();
  }, [isOpen, src, resetTransform]);

  useEffect(() => {
    if (!isOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen || !src) {
    return null;
  }

  const updatePinchStart = () => {
    const pointers = Array.from(pointersRef.current.values());
    if (pointers.length !== 2) return;
    const [first, second] = pointers;
    const distance = Math.hypot(second.x - first.x, second.y - first.y);
    const midpoint = { x: (first.x + second.x) / 2, y: (first.y + second.y) / 2 };
    pinchStartRef.current = {
      distance,
      scale,
      midpoint,
      offset,
    };
  };

  const onPointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    contentRef.current?.setPointerCapture(event.pointerId);
    pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY });

    if (pointersRef.current.size === 1) {
      panStartRef.current = {
        point: { x: event.clientX, y: event.clientY },
        offset,
      };
      pinchStartRef.current = null;
      return;
    }

    if (pointersRef.current.size === 2) {
      panStartRef.current = null;
      updatePinchStart();
    }
  };

  const onPointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!pointersRef.current.has(event.pointerId)) return;
    pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY });

    if (pointersRef.current.size === 2 && pinchStartRef.current) {
      const pointers = Array.from(pointersRef.current.values());
      const [first, second] = pointers;
      const distance = Math.hypot(second.x - first.x, second.y - first.y);
      const midpoint = { x: (first.x + second.x) / 2, y: (first.y + second.y) / 2 };
      const ratio = distance / Math.max(1, pinchStartRef.current.distance);
      const nextScale = Math.max(MIN_SCALE, Math.min(MAX_SCALE, pinchStartRef.current.scale * ratio));
      const dragFromPinch = {
        x: pinchStartRef.current.offset.x + (midpoint.x - pinchStartRef.current.midpoint.x),
        y: pinchStartRef.current.offset.y + (midpoint.y - pinchStartRef.current.midpoint.y),
      };
      setScale(nextScale);
      setOffset(clampOffset(dragFromPinch, nextScale));
      return;
    }

    if (pointersRef.current.size === 1 && panStartRef.current && scale > 1) {
      const active = pointersRef.current.values().next().value as Point;
      const nextOffset = {
        x: panStartRef.current.offset.x + (active.x - panStartRef.current.point.x),
        y: panStartRef.current.offset.y + (active.y - panStartRef.current.point.y),
      };
      setOffset(clampOffset(nextOffset, scale));
    }
  };

  const onPointerUpOrCancel = (event: React.PointerEvent<HTMLDivElement>) => {
    pointersRef.current.delete(event.pointerId);
    if (pointersRef.current.size === 1) {
      const active = pointersRef.current.values().next().value as Point;
      panStartRef.current = { point: active, offset };
      pinchStartRef.current = null;
      return;
    }
    if (pointersRef.current.size === 0) {
      panStartRef.current = null;
      pinchStartRef.current = null;
      if (scale <= 1) {
        setOffset({ x: 0, y: 0 });
      }
    }
  };

  const onDoubleClick = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    const nextScale = scale > 1 ? 1 : 2;
    setScale(nextScale);
    setOffset({ x: 0, y: 0 });
  };

  const onTouchStart = () => {
    const now = Date.now();
    if (now - lastTapRef.current < 280) {
      const nextScale = scale > 1 ? 1 : 2;
      setScale(nextScale);
      setOffset({ x: 0, y: 0 });
    }
    lastTapRef.current = now;
  };

  const handleOverlayPointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    // Block pointer events from reaching underlying bet/investigation modals.
    event.stopPropagation();
  };

  const handleOverlayClick = (event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    if (event.target !== event.currentTarget) return;
    onClose();
  };

  const handleContentClick = (event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    if (event.target !== event.currentTarget) return;
    onClose();
  };

  const modal = (
    <div
      className="image-viewer-overlay"
      onPointerDown={handleOverlayPointerDown}
      onClick={handleOverlayClick}
    >
      <button
        type="button"
        className="image-viewer-close-btn"
        onClick={event => {
          event.stopPropagation();
          onClose();
        }}
        aria-label="Закрыть просмотр"
      >
        ×
      </button>
      <div
        ref={contentRef}
        className="image-viewer-content"
        onClick={handleContentClick}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUpOrCancel}
        onPointerCancel={onPointerUpOrCancel}
        onDoubleClick={onDoubleClick}
        onTouchStart={onTouchStart}
      >
        <img
          src={src}
          alt={alt}
          className="image-viewer-image"
          draggable={false}
          style={{
            transform: `translate3d(${offset.x}px, ${offset.y}px, 0) scale(${scale})`,
          }}
        />
      </div>
    </div>
  );

  if (typeof document === 'undefined') {
    return modal;
  }

  return createPortal(modal, document.body);
};
