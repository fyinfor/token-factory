import { useEffect, useState } from 'react';
import { getPlaygroundMediaMaxHeightPx } from '../../constants/playground.constants';

export function usePlaygroundMediaMaxHeightPx() {
  const [maxHeightPx, setMaxHeightPx] = useState(() =>
    getPlaygroundMediaMaxHeightPx(),
  );

  useEffect(() => {
    const update = () => setMaxHeightPx(getPlaygroundMediaMaxHeightPx());
    window.addEventListener('resize', update);
    return () => window.removeEventListener('resize', update);
  }, []);

  return maxHeightPx;
}
