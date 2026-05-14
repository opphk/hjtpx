import { useState, useEffect, useCallback } from 'react';

export const DeviceType = {
  MOBILE: 'mobile',
  TABLET: 'tablet',
  DESKTOP: 'desktop'
};

export const Breakpoints = {
  xs: 320,
  sm: 480,
  md: 768,
  lg: 1024,
  xl: 1280,
  xxl: 1536
};

export function useResponsive() {
  const [windowSize, setWindowSize] = useState({
    width: typeof window !== 'undefined' ? window.innerWidth : 0,
    height: typeof window !== 'undefined' ? window.innerHeight : 0
  });

  const [deviceType, setDeviceType] = useState(DeviceType.DESKTOP);

  useEffect(() => {
    function handleResize() {
      const width = window.innerWidth;
      const height = window.innerHeight;

      setWindowSize({ width, height });

      if (width < Breakpoints.sm) {
        setDeviceType(DeviceType.MOBILE);
      } else if (width < Breakpoints.md) {
        setDeviceType(DeviceType.TABLET);
      } else {
        setDeviceType(DeviceType.DESKTOP);
      }
    }

    handleResize();

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const isMobile = deviceType === DeviceType.MOBILE;
  const isTablet = deviceType === DeviceType.TABLET;
  const isDesktop = deviceType === DeviceType.DESKTOP;
  const isTouch = isMobile || isTablet;

  const isBelow = (breakpoint) => windowSize.width < Breakpoints[breakpoint];
  const isAbove = (breakpoint) => windowSize.width >= Breakpoints[breakpoint];
  const isBetween = (min, max) =>
    windowSize.width >= Breakpoints[min] && windowSize.width < Breakpoints[max];

  return {
    windowSize,
    deviceType,
    isMobile,
    isTablet,
    isDesktop,
    isTouch,
    isBelow,
    isAbove,
    isBetween,
    breakpoints: Breakpoints
  };
}

export function useMediaQuery(query) {
  const [matches, setMatches] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const mediaQuery = window.matchMedia(query);

    setMatches(mediaQuery.matches);

    const handler = (event) => setMatches(event.matches);

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [query]);

  return matches;
}

export function useOrientation() {
  const [orientation, setOrientation] = useState('portrait');

  useEffect(() => {
    if (typeof window === 'undefined') return;

    function handleOrientationChange() {
      const isPortrait = window.innerHeight > window.innerWidth;
      setOrientation(isPortrait ? 'portrait' : 'landscape');
    }

    handleOrientationChange();

    window.addEventListener('resize', handleOrientationChange);
    return () => window.removeEventListener('resize', handleOrientationChange);
  }, []);

  return orientation;
}

export function useTouchGestures(options = {}) {
  const {
    onSwipeLeft,
    onSwipeRight,
    onSwipeUp,
    onSwipeDown,
    onTap,
    onLongPress,
    threshold = 50,
    longPressDelay = 500
  } = options;

  const touchState = {
    startX: 0,
    startY: 0,
    startTime: 0,
    isLongPress: false
  };

  useEffect(() => {
    if (typeof window === 'undefined') return;

    let longPressTimer;

    const handleTouchStart = (e) => {
      touchState.startX = e.touches[0].clientX;
      touchState.startY = e.touches[0].clientY;
      touchState.startTime = Date.now();
      touchState.isLongPress = false;

      if (onLongPress) {
        longPressTimer = setTimeout(() => {
          touchState.isLongPress = true;
          onLongPress(e);
        }, longPressDelay);
      }
    };

    const handleTouchEnd = (e) => {
      clearTimeout(longPressTimer);

      if (touchState.isLongPress) {
        return;
      }

      const endX = e.changedTouches[0].clientX;
      const endY = e.changedTouches[0].clientY;
      const deltaX = endX - touchState.startX;
      const deltaY = endY - touchState.startY;
      const absDeltaX = Math.abs(deltaX);
      const absDeltaY = Math.abs(deltaY);

      if (absDeltaX < threshold && absDeltaY < threshold) {
        if (onTap) {
          onTap(e);
        }
        return;
      }

      if (absDeltaX > absDeltaY) {
        if (deltaX > threshold && onSwipeRight) {
          onSwipeRight(e);
        } else if (deltaX < -threshold && onSwipeLeft) {
          onSwipeLeft(e);
        }
      } else {
        if (deltaY > threshold && onSwipeDown) {
          onSwipeDown(e);
        } else if (deltaY < -threshold && onSwipeUp) {
          onSwipeUp(e);
        }
      }
    };

    const handleTouchCancel = () => {
      clearTimeout(longPressTimer);
    };

    window.addEventListener('touchstart', handleTouchStart, { passive: true });
    window.addEventListener('touchend', handleTouchEnd, { passive: true });
    window.addEventListener('touchcancel', handleTouchCancel);

    return () => {
      window.removeEventListener('touchstart', handleTouchStart);
      window.removeEventListener('touchend', handleTouchEnd);
      window.removeEventListener('touchcancel', handleTouchCancel);
      clearTimeout(longPressTimer);
    };
  }, [onSwipeLeft, onSwipeRight, onSwipeUp, onSwipeDown, onTap, onLongPress, threshold, longPressDelay]);
}

export function usePullToRefresh(options = {}) {
  const { onRefresh, threshold = 80, disabled = false } = options;
  const [isPulling, setIsPulling] = useState(false);
  const [pullDistance, setPullDistance] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const startY = { current: 0 };

  const handleTouchStart = useCallback((e) => {
    if (window.scrollY === 0 && !disabled) {
      startY.current = e.touches[0].clientY;
      setIsPulling(true);
    }
  }, [disabled]);

  const handleTouchMove = useCallback((e) => {
    if (!isPulling || isRefreshing || disabled) return;

    const currentY = e.touches[0].clientY;
    const distance = Math.max(0, currentY - startY.current);

    if (distance > 0 && window.scrollY === 0) {
      e.preventDefault();
      setPullDistance(Math.min(distance, threshold * 2));
    }
  }, [isPulling, isRefreshing, disabled, threshold]);

  const handleTouchEnd = useCallback(() => {
    if (!isPulling || disabled) return;

    setIsPulling(false);

    if (pullDistance >= threshold && onRefresh) {
      setIsRefreshing(true);
      onRefresh().finally(() => {
        setIsRefreshing(false);
        setPullDistance(0);
      });
    } else {
      setPullDistance(0);
    }
  }, [isPulling, pullDistance, threshold, onRefresh, disabled]);

  useEffect(() => {
    if (disabled) return;

    window.addEventListener('touchstart', handleTouchStart, { passive: true });
    window.addEventListener('touchmove', handleTouchMove, { passive: false });
    window.addEventListener('touchend', handleTouchEnd);

    return () => {
      window.removeEventListener('touchstart', handleTouchStart);
      window.removeEventListener('touchmove', handleTouchMove);
      window.removeEventListener('touchend', handleTouchEnd);
    };
  }, [handleTouchStart, handleTouchMove, handleTouchEnd, disabled]);

  return {
    isPulling,
    pullDistance,
    isRefreshing,
    pullProgress: Math.min(pullDistance / threshold, 1)
  };
}

export default {
  useResponsive,
  useMediaQuery,
  useOrientation,
  useTouchGestures,
  usePullToRefresh,
  DeviceType,
  Breakpoints
};
