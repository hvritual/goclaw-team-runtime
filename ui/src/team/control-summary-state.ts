import { TeamControlSummary } from './types';

export type ControlSummaryView =
  | { kind: 'loading' }
  | { kind: 'empty'; data: TeamControlSummary }
  | { kind: 'denied'; message: string }
  | { kind: 'error'; error: unknown }
  | { kind: 'ready'; data: TeamControlSummary };

interface ControlSummaryAsyncState {
  data: TeamControlSummary | null;
  error: unknown;
  loading: boolean;
}

export function controlSummaryState(
  state: ControlSummaryAsyncState,
): ControlSummaryView {
  if (state.loading && !state.data) return { kind: 'loading' };
  if (state.error) {
    const message = state.error instanceof Error
      ? state.error.message
      : String(state.error);
    if (/(^|\D)403(\D|$)|forbidden|denied|无权|拒绝/i.test(message)) {
      return { kind: 'denied', message };
    }
    return { kind: 'error', error: state.error };
  }
  if (!state.data) return { kind: 'loading' };
  const data = state.data;
  if (
    data.budget_count === 0 &&
    data.knowledge_count === 0 &&
    data.skill_count === 0 &&
    data.runner_release_count === 0 &&
    data.context_bundle_count === 0
  ) {
    return { kind: 'empty', data };
  }
  return { kind: 'ready', data };
}
