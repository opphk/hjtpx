import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';

export function useVirtualList({
  itemCount,
  itemHeight,
  overscan = 3,
  getItemKey,
  scrollContainer
}) {
  const [scrollTop, setScrollTop] = useState(0);
  const [containerHeight, setContainerHeight] = useState(0);
  const containerRef = useRef(null);
  const rafIdRef = useRef(null);

  useEffect(() => {
    const container = scrollContainer || containerRef.current;
    if (!container) return;

    const updateContainerHeight = () => {
      const rect = container.getBoundingClientRect();
      setContainerHeight(rect.height);
    };

    const handleScroll = () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }

      rafIdRef.current = requestAnimationFrame(() => {
        setScrollTop(container.scrollTop);
      });
    };

    updateContainerHeight();
    container.addEventListener('scroll', handleScroll, { passive: true });

    const resizeObserver = new ResizeObserver(updateContainerHeight);
    resizeObserver.observe(container);

    return () => {
      container.removeEventListener('scroll', handleScroll);
      resizeObserver.disconnect();
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, [scrollContainer]);

  const virtualItems = useMemo(() => {
    const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan);
    const visibleCount = Math.ceil(containerHeight / itemHeight);
    const endIndex = Math.min(itemCount - 1, startIndex + visibleCount + overscan * 2);

    const items = [];
    for (let i = startIndex; i <= endIndex; i++) {
      items.push({
        index: i,
        key: getItemKey ? getItemKey(i) : i,
        style: {
          position: 'absolute',
          top: i * itemHeight,
          left: 0,
          right: 0,
          height: itemHeight
        }
      });
    }

    return items;
  }, [scrollTop, containerHeight, itemCount, itemHeight, overscan, getItemKey]);

  const totalHeight = itemCount * itemHeight;

  return {
    virtualItems,
    totalHeight,
    containerRef,
    scrollTo: useCallback((index) => {
      const container = scrollContainer || containerRef.current;
      if (container) {
        container.scrollTop = index * itemHeight;
      }
    }, [scrollContainer, itemHeight])
  };
}

export default function VirtualList({
  items,
  height,
  itemHeight,
  renderItem,
  getItemKey,
  className,
  overscan = 3,
  emptyComponent,
  loadingComponent
}) {
  const [isLoading, setIsLoading] = useState(false);
  const containerRef = useRef(null);

  const itemCount = items.length;

  const { virtualItems, totalHeight, containerRef: scrollContainerRef } = useVirtualList({
    itemCount,
    itemHeight,
    overscan,
    getItemKey: getItemKey || ((index) => items[index]?.id || index),
    scrollContainer: containerRef.current
  });

  const renderEmpty = () => {
    if (emptyComponent) return emptyComponent;
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: height || 300,
        color: '#999',
        fontSize: 14
      }}>
        No items to display
      </div>
    );
  };

  if (itemCount === 0) {
    return renderEmpty();
  }

  return (
    <div
      ref={containerRef}
      className={className}
      style={{
        height: height || '100%',
        overflow: 'auto',
        position: 'relative'
      }}
    >
      <div style={{ height: totalHeight, position: 'relative' }}>
        {virtualItems.map(({ index, key, style }) => (
          <div key={key} style={style}>
            {renderItem(items[index], index)}
          </div>
        ))}
      </div>
      {isLoading && loadingComponent}
    </div>
  );
}
