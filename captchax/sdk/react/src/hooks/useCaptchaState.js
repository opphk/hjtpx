import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { useCaptcha } from '../CaptchaProvider';

export const useCaptchaState = (initialState = {}) => {
  const {
    verify,
    config,
    loading,
    error,
    clearError
  } = useCaptcha();

  const [state, setState] = useState({
    isVerified: false,
    token: null,
    attempts: 0,
    maxAttempts: 3,
    startTime: null,
    endTime: null,
    duration: null,
    ...initialState
  });

  const [localLoading, setLocalLoading] = useState(false);
  const [localError, setLocalError] = useState(null);
  const timerRef = useRef(null);

  const updateState = useCallback((updates) => {
    setState(prev => ({ ...prev, ...updates }));
  }, []);

  const startVerification = useCallback(() => {
    setLocalLoading(true);
    setLocalError(null);
    updateState({
      isVerified: false,
      token: null,
      startTime: Date.now(),
      endTime: null,
      duration: null
    });
  }, [updateState]);

  const handleSuccess = useCallback((token) => {
    const endTime = Date.now();
    updateState({
      isVerified: true,
      token,
      endTime,
      duration: endTime - state.startTime,
      attempts: state.attempts + 1
    });
    setLocalLoading(false);
    setLocalError(null);
  }, [state.startTime, state.attempts, updateState]);

  const handleError = useCallback((err) => {
    const errorMessage = err instanceof Error ? err.message : 'Verification failed';
    updateState({
      attempts: state.attempts + 1
    });
    setLocalError(errorMessage);
    setLocalLoading(false);
  }, [state.attempts, updateState]);

  const reset = useCallback(() => {
    setState({
      isVerified: false,
      token: null,
      attempts: 0,
      maxAttempts: 3,
      startTime: null,
      endTime: null,
      duration: null
    });
    setLocalError(null);
    setLocalLoading(false);
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }
  }, []);

  const verifyWithState = useCallback(async (scene, options = {}) => {
    startVerification();
    try {
      const token = await verify(scene, options);
      handleSuccess(token);
      return token;
    } catch (err) {
      handleError(err);
      throw err;
    }
  }, [verify, startVerification, handleSuccess, handleError]);

  const canRetry = useMemo(
    () => state.attempts < state.maxAttempts,
    [state.attempts, state.maxAttempts]
  );

  const retryDelay = useMemo(() => {
    return Math.min(1000 * Math.pow(2, state.attempts), 10000);
  }, [state.attempts]);

  useEffect(() => {
    if (canRetry && localError) {
      timerRef.current = setTimeout(() => {
        setLocalError(null);
      }, retryDelay);
    }

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [canRetry, localError, retryDelay]);

  return {
    state,
    loading: localLoading || loading,
    error: localError || error,
    clearError: () => {
      setLocalError(null);
      clearError?.();
    },
    verify: verifyWithState,
    reset,
    canRetry,
    retryDelay
  };
};

export default useCaptchaState;
