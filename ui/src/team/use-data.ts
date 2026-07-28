import { DependencyList, useCallback, useEffect, useRef, useState } from 'react';
import { useTeam } from './context';

export interface AsyncData<T> {
  data: T | null;
  error: unknown;
  loading: boolean;
  reload: () => void;
}

export function sameDependencies(
  left: DependencyList,
  right: DependencyList,
): boolean {
  return left.length === right.length &&
    left.every((dependency, index) => Object.is(dependency, right[index]));
}

export function useAsyncData<T>(
  loader: () => Promise<T>,
  dependencies: DependencyList,
): AsyncData<T> {
  const { refreshRevision } = useTeam();
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [loading, setLoading] = useState(true);
  const [revision, setRevision] = useState(0);
  const scopeDependencies = useRef<DependencyList>(dependencies);
  const scopeChanged = !sameDependencies(
    dependencies,
    scopeDependencies.current,
  );
  if (scopeChanged) scopeDependencies.current = dependencies;

  const reload = useCallback(() => setRevision((value) => value + 1), []);

  useEffect(() => {
    let active = true;
    if (scopeChanged) setData(null);
    setLoading(true);
    setError(null);
    void loader()
      .then((value) => {
        if (active) setData(value);
      })
      .catch((reason: unknown) => {
        if (active) setError(reason);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  // The caller owns the dependency list, matching React's useEffect contract.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...dependencies, revision, refreshRevision]);

  return {
    data: scopeChanged ? null : data,
    error: scopeChanged ? null : error,
    loading: scopeChanged || loading,
    reload,
  };
}
