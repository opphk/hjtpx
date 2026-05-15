import { useState, useCallback } from 'react';

export const useDirection = () => {
  const [direction, setDirectionState] = useState('ltr');

  const setDirection = useCallback((dir) => {
    const newDirection = dir === 'rtl' ? 'rtl' : 'ltr';
    setDirectionState(newDirection);

    document.documentElement.dir = newDirection;
    document.documentElement.lang = dir === 'rtl' ? 'ar' : 'en';

    document.body.classList.remove('ltr', 'rtl');
    document.body.classList.add(newDirection);
  }, []);

  const isRTL = direction === 'rtl';

  return {
    direction,
    setDirection,
    isRTL
  };
};

export default useDirection;
