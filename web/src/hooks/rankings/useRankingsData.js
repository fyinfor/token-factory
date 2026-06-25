/*
Copyright (C) 2025 QuantumNous
*/

import { useCallback, useEffect, useState } from 'react';
import { fetchRankings, normalizeRankingPeriod } from '../../helpers/rankings';

export function useRankingsData(initialPeriod = 'week') {
  const [period, setPeriod] = useState(() => normalizeRankingPeriod(initialPeriod));
  const [snapshot, setSnapshot] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async (nextPeriod) => {
    const normalized = normalizeRankingPeriod(nextPeriod ?? period);
    setLoading(true);
    setError('');
    try {
      const data = await fetchRankings(normalized);
      setSnapshot(data);
      setPeriod(normalized);
    } catch (err) {
      setSnapshot(null);
      setError(err?.message || String(err));
    } finally {
      setLoading(false);
    }
  }, [period]);

  useEffect(() => {
    load(period);
  }, []);

  const changePeriod = useCallback((nextPeriod) => {
    load(nextPeriod);
  }, [load]);

  return {
    period,
    changePeriod,
    snapshot,
    loading,
    error,
    reload: () => load(period),
  };
}
