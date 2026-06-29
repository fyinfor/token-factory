/*
Copyright (C) 2025 QuantumNous
*/

import { useCallback, useEffect, useState } from 'react';
import {
  fetchRankings,
  normalizeRankingCategory,
  normalizeRankingPeriod,
} from '../../helpers/rankings';

export function useRankingsData(initialPeriod = 'week', initialCategory = 'all') {
  const [period, setPeriod] = useState(() => normalizeRankingPeriod(initialPeriod));
  const [category, setCategory] = useState(() => normalizeRankingCategory(initialCategory));
  const [snapshot, setSnapshot] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(
    async (nextPeriod, nextCategory) => {
      const normalizedPeriod = normalizeRankingPeriod(nextPeriod ?? period);
      const normalizedCategory = normalizeRankingCategory(nextCategory ?? category);
      setLoading(true);
      setError('');
      try {
        const data = await fetchRankings(normalizedPeriod, normalizedCategory);
        setSnapshot(data);
        setPeriod(normalizedPeriod);
        setCategory(normalizedCategory);
      } catch (err) {
        setSnapshot(null);
        setError(err?.message || String(err));
      } finally {
        setLoading(false);
      }
    },
    [period, category],
  );

  useEffect(() => {
    load(period, category);
  }, []);

  const changePeriod = useCallback(
    (nextPeriod) => {
      load(nextPeriod, category);
    },
    [load, category],
  );

  const changeCategory = useCallback(
    (nextCategory) => {
      load(period, nextCategory);
    },
    [load, period],
  );

  return {
    period,
    category,
    changePeriod,
    changeCategory,
    snapshot,
    loading,
    error,
    reload: () => load(period, category),
  };
}
