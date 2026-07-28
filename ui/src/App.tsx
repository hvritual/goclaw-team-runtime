import { useEffect, useState } from 'react';
import { AppShell } from './team/AppShell';
import { ApprovalsPage } from './team/ApprovalsPage';
import { ChatPage } from './team/ChatPage';
import { DevelopmentPage } from './team/DevelopmentPage';
import { HarnessPage } from './team/HarnessPage';
import { LoginScreen } from './team/LoginScreen';
import { MemoryPage } from './team/MemoryPage';
import { OverviewPage } from './team/OverviewPage';
import { ProgressPage } from './team/ProgressPage';
import { SpecPage } from './team/SpecPage';
import { TeamPage } from './team/TeamPage';
import { TeamProvider, useTeam } from './team/context';
import { Loading } from './team/primitives';
import { PageID } from './team/types';

const pages = new Set<PageID>([
  'overview',
  'chat',
  'spec',
  'memory',
  'approvals',
  'development',
  'team',
  'progress',
  'harness',
]);

function initialPage(): PageID {
  const value = window.location.hash.replace(/^#\/?/, '') as PageID;
  return pages.has(value) ? value : 'overview';
}

function ConsoleApp() {
  const { session, restoring } = useTeam();
  const [page, setPage] = useState<PageID>(initialPage);

  useEffect(() => {
    const onHashChange = () => setPage(initialPage());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  const choosePage = (next: PageID) => {
    setPage(next);
    window.history.replaceState(null, '', `#/${next}`);
  };

  if (restoring) return <div className="boot-screen"><Loading label="恢复安全会话…" /></div>;
  if (!session) return <LoginScreen />;

  return (
    <AppShell page={page} onPageChange={choosePage}>
      {page === 'overview' ? <OverviewPage /> : null}
      {page === 'chat' ? <ChatPage /> : null}
      {page === 'spec' ? <SpecPage /> : null}
      {page === 'memory' ? <MemoryPage /> : null}
      {page === 'approvals' ? <ApprovalsPage /> : null}
      {page === 'development' ? <DevelopmentPage /> : null}
      {page === 'team' ? <TeamPage /> : null}
      {page === 'progress' ? <ProgressPage /> : null}
      {page === 'harness' ? <HarnessPage /> : null}
    </AppShell>
  );
}

export default function App() {
  return <TeamProvider><ConsoleApp /></TeamProvider>;
}
