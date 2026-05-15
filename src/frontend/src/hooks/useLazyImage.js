import { useState, useEffect, useRef, useCallback } from 'react';

export const useLazyImage = ({
  src,
  threshold = 0.1,
  rootMargin = '50px',
  placeholder = null,
  errorFallback = null
} = {}) => {
  const [isLoaded, setIsLoaded] = useState(false);
  const [isInView, setIsInView] = useState(false);
  const [hasError, setHasError] = useState(false);
  const [currentSrc, setCurrentSrc] = useState(null);
  const elementRef = useRef(null);
  const observerRef = useRef(null);
  const imgRef = useRef(null);

  useEffect(() => {
    if (!elementRef.current) return;

    observerRef.current = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsInView(true);
          observerRef.current?.disconnect();
        }
      },
      { threshold, rootMargin }
    );

    observerRef.current.observe(elementRef.current);

    return () => {
      observerRef.current?.disconnect();
    };
  }, [threshold, rootMargin]);

  useEffect(() => {
    if (!isInView || !src) return;

    setCurrentSrc(src);

    const img = new Image();
    imgRef.current = img;

    img.onload = () => {
      setIsLoaded(true);
      setHasError(false);
    };

    img.onerror = () => {
      setHasError(true);
      setIsLoaded(false);
    };

    img.src = src;

    return () => {
      img.onload = null;
      img.onerror = null;
    };
  }, [isInView, src]);

  const retry = useCallback(() => {
    setHasError(false);
    setIsLoaded(false);
    if (src) {
      setCurrentSrc(src + `?t=${Date.now()}`);
    }
  }, [src]);

  return {
    ref: elementRef,
    isLoaded,
    isInView,
    hasError,
    currentSrc,
    retry,
    placeholder,
    errorFallback
  };
};

export default useLazyImage;
