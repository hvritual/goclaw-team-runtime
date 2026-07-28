import { useState, useCallback, useEffect } from 'react';
import { rpc } from '../services/rpc';

interface UseApiReturn<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  execute: (...args: unknown[]) => Promise<T | null>;
  reset: () => void;
}

export const useApi = <T,>(apiCall: (...args: unknown[]) => Promise<T>): UseApiReturn<T> => {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const execute = useCallback(async (...args: unknown[]): Promise<T | null> => {
    if (!apiCall) {
      console.warn('apiCall is not defined');
      return null;
    }
    setLoading(true);
    setError(null);

    try {
      const result = await apiCall(...args);
      setData(result);
      return result;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'An error occurred';
      setError(errorMessage);
      return null;
    } finally {
      setLoading(false);
    }
  }, [apiCall]);

  const reset = useCallback(() => {
    setData(null);
    setLoading(false);
    setError(null);
  }, []);

  return {
    data,
    loading,
    error,
    execute,
    reset,
  };
};

// Pre-configured hooks for common API calls
// These hooks automatically fetch data on mount

export const useHealth = () => {
  const api = useApi(() => rpc.health());
  useEffect(() => {
    console.log('[useHealth] executing health check...');
    api.execute().then(result => {
      console.log('[useHealth] result:', result);
      console.log('[useHealth] error:', api.error);
    }).catch(err => {
      console.error('[useHealth] error:', err);
    });
  }, []);
  return api;
};

export const useChannels = () => {
  const api = useApi(() => rpc.channelsList());
  useEffect(() => {
    api.execute();
  }, []);
  return api;
};

export const useSessions = () => {
  const api = useApi(() => rpc.sessionsList());
  useEffect(() => {
    api.execute();
  }, []);
  return api;
};

export const useCronStatus = () => {
  const api = useApi(() => rpc.cronStatus());
  useEffect(() => {
    api.execute();
  }, []);
  return api;
};

export const useCronList = () => useApi(rpc.cronList);

export const useLogs = () => useApi(() => rpc.logsGet());

export default useApi;