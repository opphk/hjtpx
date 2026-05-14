import React, { useState, useEffect, useRef, memo } from 'react';

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
  ...props
}) {
  const [isLoaded, setIsLoaded] = useState(false);
  const [isInView, setIsInView] = useState(false);
  const [hasError, setHasError] = useState(false);
  const imgRef = useRef(null);
  const observerRef = useRef(null);

  useEffect(() => {
    if (!imgRef.current) return;

    observerRef.current = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsInView(true);
          observerRef.current?.disconnect();
        }
      },
      { threshold, rootMargin }
    );

    observerRef.current.observe(imgRef.current);

    return () => {
      observerRef.current?.disconnect();
    };
  }, [threshold, rootMargin]);

  const handleLoad = () => {
    setIsLoaded(true);
    onLoad?.();
  };

  const handleError = () => {
    setHasError(true);
    onError?.();
  };

  const defaultPlaceholder = (
    <div
      style={{
        width: '100%',
        height: '100%',
        backgroundColor: '#f0f0f0',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <span>Loading...</span>
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
      }}
    >
      <span>Failed to load image</span>
    </div>
  );

  if (hasError) {
    return (
      <div ref={imgRef} className={className} {...props}>
        {errorFallback || defaultErrorFallback}
      </div>
    );
  }

  return (
    <div ref={imgRef} className={className} style={{ position: 'relative', overflow: 'hidden' }} {...props}>
      {!isLoaded && (placeholder || defaultPlaceholder)}
      {isInView && (
        <img
          src={src}
          alt={alt}
          onLoad={handleLoad}
          onError={handleError}
          style={{
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            opacity: isLoaded ? 1 : 0,
            transition: 'opacity 0.3s ease-in-out',
          }}
        />
      )}
    </div>
  );
}

export default memo(LazyImage);
