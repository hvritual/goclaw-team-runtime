import {
  FormEvent,
  KeyboardEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  appendTransientMessage,
  applyChatEvent,
  chatEventMatches,
  chatScope,
  emptyChatState,
  mergeChatHistory,
} from './chat-state';
import { useTeam } from './context';
import { Button, Empty, Loading, PageHeader, Status } from './primitives';
import { ChatEvent, ChatHistory } from './types';
import { useAsyncData } from './use-data';

export function ChatPage() {
  const { client, projectID, topicID } = useTeam();
  const scope = useMemo(() => chatScope(projectID, topicID), [projectID, topicID]);
  const [view, setView] = useState(() => emptyChatState(scope));
  const [content, setContent] = useState('');
  const [sending, setSending] = useState(false);
  const endRef = useRef<HTMLDivElement | null>(null);
  const scopeRef = useRef(scope);
  scopeRef.current = scope;
  const loadHistory = useCallback(
    () => client.rpc<ChatHistory>('chat.history', {
      project_id: projectID,
      topic_id: topicID,
      limit: 200,
    }),
    [client, projectID, topicID],
  );
  const history = useAsyncData(loadHistory, [loadHistory]);
  const messages = view.scope === scope ? view.messages : [];

  useEffect(() => client.on('chat.event', (value) => {
    const payload = value as ChatEvent;
    if (!payload?.run_id) return;
    if (!chatEventMatches(payload, projectID, topicID)) return;
    setView((current) => applyChatEvent(current, scope, payload));
  }), [client, projectID, scope, topicID]);

  useEffect(() => {
    setContent('');
    setSending(false);
    setView((current) => current.scope === scope ? current : emptyChatState(scope));
  }, [scope]);

  useEffect(() => {
    const snapshot = history.data;
    if (!snapshot) return;
    setView((current) => mergeChatHistory(current, scope, snapshot.messages));
  }, [history.data, scope]);

  useEffect(
    () => endRef.current?.scrollIntoView({ behavior: 'smooth' }),
    [messages],
  );

  const submit = async (event?: FormEvent) => {
    event?.preventDefault();
    const value = content.trim();
    if (!value || sending) return;
    setView((current) => appendTransientMessage(current, scope, {
      id: crypto.randomUUID(),
      role: 'user',
      content: value,
      timestamp: new Date().toISOString(),
    }));
    setContent('');
    setSending(true);
    try {
      await client.rpc('agent', { content: value, project_id: projectID, topic_id: topicID }, 600_000);
    } catch (reason) {
      if (scopeRef.current === scope) {
        setView((current) => appendTransientMessage(current, scope, {
          id: crypto.randomUUID(),
          role: 'system',
          content: reason instanceof Error ? reason.message : String(reason),
          error: true,
        }));
      }
    } finally {
      if (scopeRef.current === scope) setSending(false);
    }
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      void submit();
    }
  };

  return (
    <div className="chat-page">
      <PageHeader title="项目对话" description={`所有消息绑定 ${projectID} / ${topicID}，与飞书入口共享同一会话。`} />
      <section className="chat-surface">
        <div className="chat-context">
          <Status tone="accent">{projectID}</Status>
          <span>Topic {topicID}</span>
          <span>消息不会绕过审批、Seed 或 DoneGate</span>
        </div>
        <div className="message-list">
          {history.loading && messages.length === 0 ? (
            <Loading label="恢复项目会话…" />
          ) : messages.length === 0 ? (
            <Empty title="开始项目对话" detail="适合澄清、查询和提出候选；需要执行的开发需求请在“规格”中结晶。" />
          ) : messages.map((message) => (
            <article key={message.id} className={`message is-${message.role}${message.error ? ' is-error' : ''}`}>
              <div className="message-author">{message.role === 'user' ? '你' : message.role === 'assistant' ? 'GoClaw' : '系统'}</div>
              <div>{message.content || '…'}{message.pending ? <span className="typing-caret" /> : null}</div>
            </article>
          ))}
          {history.error ? (
            <p className="inline-error">
              历史恢复失败：{history.error instanceof Error ? history.error.message : String(history.error)}
            </p>
          ) : null}
          <div ref={endRef} />
        </div>
        <form className="chat-composer" onSubmit={(event) => void submit(event)}>
          <textarea
            value={content}
            onChange={(event) => setContent(event.target.value)}
            onKeyDown={onKeyDown}
            rows={4}
            placeholder="给当前项目发送消息…"
            aria-label="项目消息"
          />
          <div><span>⌘ / Ctrl + Enter 发送</span><Button tone="accent" busy={sending} disabled={!content.trim()}>发送</Button></div>
        </form>
      </section>
    </div>
  );
}
