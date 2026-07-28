import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { teamClient, TeamClient } from './client';
import { GovernanceInput, LoginInput, TeamProject, WebSession } from './types';

interface TeamContextValue {
  client: TeamClient;
  session: WebSession | null;
  restoring: boolean;
  connection: 'connecting' | 'connected' | 'disconnected' | 'error';
  projects: TeamProject[];
  projectsLoading: boolean;
  projectsError: unknown;
  projectID: string;
  topicID: string;
  refreshRevision: number;
  reviewerToken: string;
  setProjectID: (value: string) => void;
  setTopicID: (value: string) => void;
  setReviewerToken: (value: string) => void;
  login: (input: LoginInput) => Promise<void>;
  logout: () => Promise<void>;
  governance: (input: Omit<GovernanceInput, 'reviewerToken'>) => Record<string, unknown>;
}

const TeamContext = createContext<TeamContextValue | null>(null);

export function TeamProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<WebSession | null>(null);
  const [restoring, setRestoring] = useState(true);
  const [connection, setConnection] =
    useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');
  const [projects, setProjects] = useState<TeamProject[]>([]);
  const [projectsLoading, setProjectsLoading] = useState(false);
  const [projectsError, setProjectsError] = useState<unknown>(null);
  const [projectID, setProjectIDState] = useState('');
  const [topicID, setTopicIDState] = useState('inbox');
  const [reviewerToken, setReviewerToken] = useState('');
  const [refreshRevision, setRefreshRevision] = useState(0);

  useEffect(() => {
    const unsubscribeConnection = teamClient.onConnection((next) => {
      setConnection(next);
      if (next === 'connected') setRefreshRevision((value) => value + 1);
    });
    const unsubscribeSession = teamClient.onSession((next) => {
      setSession(next);
      if (!next) {
        setReviewerToken('');
        setProjects([]);
        setProjectIDState('');
      }
    });
    return () => {
      unsubscribeConnection();
      unsubscribeSession();
    };
  }, []);

  useEffect(() => {
    let active = true;
    void teamClient.resume()
      .then((value) => {
        if (active) setSession(value);
      })
      .catch(() => {
        if (active) setSession(null);
      })
      .finally(() => {
        if (active) setRestoring(false);
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!session) return undefined;
    const refresh = () => {
      if (document.visibilityState === 'visible') {
        setRefreshRevision((value) => value + 1);
      }
    };
    const onVisibility = () => refresh();
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', onVisibility);
    const interval = window.setInterval(refresh, 15_000);
    return () => {
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', onVisibility);
      window.clearInterval(interval);
    };
  }, [session]);

  useEffect(() => {
    if (!session) {
      setProjectsLoading(false);
      setProjectsError(null);
      return undefined;
    }
    let active = true;
    setProjectsLoading(true);
    setProjectsError(null);
    void teamClient.rpc<TeamProject[]>('project.list')
      .then((items) => {
        if (!active) return;
        setProjects(items);
        setProjectIDState((current) => {
          if (items.some((project) => project.id === current)) return current;
          return items.find((project) => project.status === 'active')?.id ?? items[0]?.id ?? '';
        });
      })
      .catch((reason: unknown) => {
        if (active) setProjectsError(reason);
      })
      .finally(() => {
        if (active) setProjectsLoading(false);
      });
    return () => {
      active = false;
    };
  }, [session, refreshRevision]);

  const login = useCallback(async (input: LoginInput) => {
    const value = await teamClient.login(input);
    setSession(value);
  }, []);

  const logout = useCallback(async () => {
    setReviewerToken('');
    await teamClient.logout();
    setSession(null);
  }, []);

  const setProjectID = useCallback((value: string) => {
    const next = value.trim();
    if (projects.some((project) => project.id === next)) setProjectIDState(next);
  }, [projects]);

  const setTopicID = useCallback((value: string) => {
    setTopicIDState(value.trim() || 'inbox');
  }, []);

  const governance = useCallback((input: Omit<GovernanceInput, 'reviewerToken'>) => ({
    reviewer_token: reviewerToken,
    rationale: input.rationale,
    counterargument: input.counterargument,
    evidence_refs: input.evidenceRefs.filter(Boolean),
  }), [reviewerToken]);

  const value = useMemo<TeamContextValue>(() => ({
    client: teamClient,
    session,
    restoring,
    connection,
    projects,
    projectsLoading,
    projectsError,
    projectID,
    topicID,
    refreshRevision,
    reviewerToken,
    setProjectID,
    setTopicID,
    setReviewerToken,
    login,
    logout,
    governance,
  }), [
    session,
    restoring,
    connection,
    projects,
    projectsLoading,
    projectsError,
    projectID,
    topicID,
    refreshRevision,
    reviewerToken,
    setProjectID,
    setTopicID,
    login,
    logout,
    governance,
  ]);

  return <TeamContext.Provider value={value}>{children}</TeamContext.Provider>;
}

export function useTeam(): TeamContextValue {
  const value = useContext(TeamContext);
  if (!value) throw new Error('useTeam must be used inside TeamProvider');
  return value;
}
