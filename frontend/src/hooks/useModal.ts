import { useState, useCallback } from 'react';

export interface UseModalReturn<T> {
  visible: boolean;
  current: T | null;
  isEdit: boolean;
  open: (item?: T) => void;
  close: () => void;
}

export function useModal<T>(): UseModalReturn<T> {
  const [visible, setVisible] = useState(false);
  const [current, setCurrent] = useState<T | null>(null);

  const open = useCallback((item?: T) => {
    setCurrent(item || null);
    setVisible(true);
  }, []);

  const close = useCallback(() => {
    setVisible(false);
    setCurrent(null);
  }, []);

  return {
    visible,
    current,
    isEdit: current !== null,
    open,
    close,
  };
}
