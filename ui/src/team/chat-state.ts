import { ChatEvent, ChatMessage } from './types';

export interface ChatViewState {
  scope: string;
  messages: ChatMessage[];
  sequences: Record<string, number>;
}

export function chatScope(projectID: string, topicID: string): string {
  return `${projectID}\u0000${topicID}`;
}

export function emptyChatState(scope: string): ChatViewState {
  return { scope, messages: [], sequences: {} };
}

export function chatEventMatches(
  event: ChatEvent,
  projectID: string,
  topicID: string,
): boolean {
  return event.project_id === projectID && event.topic_id === topicID;
}

export function applyChatEvent(
  current: ChatViewState,
  scope: string,
  event: ChatEvent,
): ChatViewState {
  const state = current.scope === scope ? current : emptyChatState(scope);
  const priorSequence = state.sequences[event.run_id] ?? -1;
  if (event.seq <= priorSequence) return state;

  const index = state.messages.findIndex((message) => message.id === event.run_id);
  const prior = index >= 0 ? state.messages[index] : {
    id: event.run_id,
    role: 'assistant' as const,
    content: '',
    pending: true,
    transient: true,
    timestamp: event.timestamp,
  };
  const message = { ...prior };
  if (event.state === 'delta') message.content += event.content ?? '';
  if (event.state === 'thinking' && !message.content) message.content = '正在分析…';
  if (event.state === 'tool' && !message.content) {
    message.content = '正在调用受控工具…';
  }
  if (event.state === 'final') {
    message.content = event.content || message.content;
    message.pending = false;
  }
  if (event.state === 'error') {
    message.content = event.content || '运行失败';
    message.pending = false;
    message.error = true;
  }
  const messages = [...state.messages];
  if (index >= 0) messages[index] = message;
  else messages.push(message);
  return {
    scope,
    messages,
    sequences: { ...state.sequences, [event.run_id]: event.seq },
  };
}

export function mergeChatHistory(
  current: ChatViewState,
  scope: string,
  history: ChatMessage[],
): ChatViewState {
  const state = current.scope === scope ? current : emptyChatState(scope);
  const unmatched = [...history];
  const transient = state.messages.filter((message) => {
    if (!message.transient) return false;
    const index = unmatched.findIndex((snapshot) =>
      snapshot.role === message.role && snapshot.content === message.content);
    if (index < 0) return true;
    unmatched.splice(index, 1);
    return false;
  });
  return {
    ...state,
    messages: [
      ...history.map((message) => ({ ...message, transient: false })),
      ...transient,
    ],
  };
}

export function appendTransientMessage(
  current: ChatViewState,
  scope: string,
  message: ChatMessage,
): ChatViewState {
  const state = current.scope === scope ? current : emptyChatState(scope);
  return {
    ...state,
    messages: [...state.messages, { ...message, transient: true }],
  };
}
