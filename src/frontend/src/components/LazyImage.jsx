import React, { useState, useEffect, useRef, memo, useCallback } from 'react';

function LazyImage({
  src,
  alt,
  className,
  placeholder,
  errorFallback,
  threshold = 0.1,
  rootMargin = '50px',
  onLoad,
  onError,
  srcSet,
  sizes,
  aspectRatio,
  objectFit = 'cover',
  priority = false,
  blurAmount = 10,
  ...props
}) {
  const [isLoaded, setIsLoaded] = useState(false);
  const [isInView, setIsInView] = useState(false);
  const [hasError, setHasError] = useState(false);
  const [isBlurred, setIsBlurred] = useState(true);
  const imgRef = useRef(null);
  const observerRef = useRef(null);
  const containerRef = useRef(null);

  useEffect(() => {
    if (priority) {
      setIsInView(true);
      return;
    }

    if (!containerRef.current) return;

    observerRef.current = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsInView(true);
          observerRef.current?.disconnect();
        }
      },
      { threshold, rootMargin }
    );

    observerRef.current.observe(containerRef.current);

    return () => {
      observerRef.current?.disconnect();
    };
  }, [threshold, rootMargin, priority]);

  const handleLoad = useCallback(() => {
    setIsLoaded(true);
    setTimeout(() => setIsBlurred(false), 50);
    onLoad?.();
  }, [onLoad]);

  const handleError = useCallback(() => {
    setHasError(true);
    onError?.();
  }, [onError]);

  const defaultPlaceholder = (
    <div
      style={{
        width: '100%',
        height: '100%',
        backgroundColor: '#f0f0f0',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        position: 'absolute',
        top: 0,
        left: 0
      }}
    >
      <div
        style={{
          width: 24,
          height: 24,
          border: '2px solid #e0e0e0',
          borderTopColor: '#2196f3',
          borderRadius: '50%',
          animation: 'spin 1s linear infinite'
        }}
      />
    </div>
  );

  const defaultErrorFallback = (
    <div
      style={{
        width: '100%',
        height: '100%',
        backgroundColor: '#ffebee',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#c62828',
        position: 'absolute',
        top: 0,
        left: 0
      }}
    >
      <span>Failed to load image</span>
    </div>
  );

  if (hasError) {
    return (
      <div
        ref={containerRef}
        className={className}
        style={{
          position: 'relative',
          aspectRatio,
          overflow: 'hidden'
        }}
        {...props}
      >
        {errorFallback || defaultErrorFallback}
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className={className}
      style={{
        position: 'relative',
        overflow: 'hidden',
        aspectRatio
      }}
      {...props}
    >
      {!isLoaded && (placeholder || defaultPlaceholder)}
      {isInView && (
        <img
          ref={imgRef}
          src={src}
          srcSet={srcSet}
          sizes={sizes}
          alt={alt}
          onLoad={handleLoad}
          onError={handleError}
          loading={priority ? 'eager' : 'lazy'}
          decoding="async"
          style={{
            width: '100%',
            height: '100%',
            objectFit,
            opacity: isLoaded ? 1 : 0,
            filter: isBlurred ? `blur(${blurAmount}px)` : 'none',
            transition: 'opacity 0.3s ease-in-out, filter 0.3s ease-in-out'
          }}
        />
      )}
      <style>
        {`
          @keyframes spin {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
          }
        `}
      </style>
    </div>
  );
}

export default memo(LazyImage);
