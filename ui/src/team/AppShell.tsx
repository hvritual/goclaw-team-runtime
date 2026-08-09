import { ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTeam } from "./context";
import { Icon, IconName } from "./icons";
import { Button, Empty, ErrorState, Loading, Status } from "./primitives";
import { PageID } from "./types";

const navigationGroups: Array<{
  label: string;
  items: Array<{ id: PageID; label: string; icon: IconName }>;
}> = [
  {
    label: "工作区",
    items: [
      { id: "overview", label: "项目总览", icon: "overview" },
      { id: "chat", label: "项目对话", icon: "chat" },
      { id: "approvals", label: "我的审批", icon: "approval" },
    ],
  },
  {
    label: "交付",
    items: [
      { id: "spec", label: "需求与方案", icon: "spec" },
      { id: "work", label: "任务", icon: "document" },
      { id: "development", label: "开发执行", icon: "development" },
    ],
  },
  {
    label: "质量",
    items: [
      { id: "quality", label: "Bug 与风险", icon: "warning" },
      { id: "reviews", label: "代码评审", icon: "branch" },
      { id: "progress", label: "进度与证据", icon: "progress" },
    ],
  },
  {
    label: "知识与治理",
    items: [
      { id: "memory", label: "知识资产", icon: "memory" },
      { id: "team", label: "团队控制", icon: "team" },
      { id: "harness", label: "Harness", icon: "harness" },
    ],
  },
];
const navigation = navigationGroups.flatMap((group) => group.items);

export function AppShell({
  page,
  onPageChange,
  children,
}: {
  page: PageID;
  onPageChange: (page: PageID) => void;
  children: ReactNode;
}) {
  const {
    session,
    connection,
    projects,
    projectsLoading,
    projectsError,
    projectID,
    topicID,
    reviewerToken,
    setProjectID,
    setTopicID,
    setReviewerToken,
    logout,
  } = useTeam();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [credentialsOpen, setCredentialsOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const searchRef = useRef<HTMLInputElement | null>(null);
  const [topicDraft, setTopicDraft] = useState(topicID);
  const currentLabel = useMemo(
    () => navigation.find((item) => item.id === page)?.label ?? "总览",
    [page],
  );

  const choosePage = (next: PageID) => {
    onPageChange(next);
    setMobileOpen(false);
  };
  useEffect(() => setTopicDraft(topicID), [topicID]);
  useEffect(() => {
    const onShortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setSearchOpen(true);
        window.setTimeout(() => searchRef.current?.focus(), 0);
      }
      if (event.key === "Escape") setSearchOpen(false);
    };
    window.addEventListener("keydown", onShortcut);
    return () => window.removeEventListener("keydown", onShortcut);
  }, []);

  const commitTopic = () => {
    const next = topicDraft.trim() || "inbox";
    if (/^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/.test(next)) {
      setTopicID(next);
    } else {
      setTopicDraft(topicID);
    }
  };
  const searchResults = navigation.filter(
    (item) =>
      !searchQuery.trim() ||
      item.label.toLowerCase().includes(searchQuery.trim().toLowerCase()),
  );

  return (
    <div className="app-shell">
      <aside className={`sidebar ${mobileOpen ? "is-open" : ""}`}>
        <div className="sidebar-brand">
          <span className="brand-mark">G</span>
          <div>
            <strong>GoClaw</strong>
            <span>Team Console</span>
          </div>
          <button
            className="icon-button mobile-only"
            onClick={() => setMobileOpen(false)}
            aria-label="关闭导航"
          >
            <Icon name="close" />
          </button>
        </div>
        <label className="project-switcher">
          <span>当前项目</span>
          <select
            value={projectID}
            onChange={(event) => setProjectID(event.target.value)}
            aria-label="当前项目 ID"
            disabled={projectsLoading && projects.length === 0}
            style={{
              width: "100%",
              border: "1px solid #dedbd8",
              borderRadius: 4,
              background: "#fff",
              padding: "7px 9px",
              fontSize: 12,
            }}
          >
            {projects.length === 0 ? (
              <option value="">无授权项目</option>
            ) : null}
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name} · {project.key}
                {project.status === "archived" ? "（已归档）" : ""}
              </option>
            ))}
          </select>
        </label>
        <nav>
          {navigationGroups.map((group) => (
            <section className="nav-group" key={group.label}>
              <span className="nav-group-label">{group.label}</span>
              {group.items.map((item) => (
                <button
                  key={item.id}
                  className={page === item.id ? "is-active" : ""}
                  onClick={() => choosePage(item.id)}
                >
                  <Icon name={item.icon} />
                  <span>{item.label}</span>
                </button>
              ))}
            </section>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div>
            <Icon name="shield" />
            <span>策略与证据受控</span>
          </div>
          <small>Runtime 0.8.0-pilot.1</small>
        </div>
      </aside>
      {mobileOpen ? (
        <button
          className="sidebar-scrim"
          onClick={() => setMobileOpen(false)}
          aria-label="关闭导航"
        />
      ) : null}
      <div className="workspace">
        <header className="topbar">
          <button
            className="icon-button mobile-only"
            onClick={() => setMobileOpen(true)}
            aria-label="打开导航"
          >
            <Icon name="menu" />
          </button>
          <div className="topbar-context">
            <strong>{currentLabel}</strong>
            <span>{projectID}</span>
            <Status
              tone={
                connection === "connected"
                  ? "success"
                  : connection === "error"
                    ? "danger"
                    : "warning"
              }
            >
              {connection === "connected"
                ? "在线"
                : connection === "connecting"
                  ? "连接中"
                  : "离线"}
            </Status>
          </div>
          <div className="command-search">
            <Icon name="search" />
            <input
              ref={searchRef}
              value={searchQuery}
              onChange={(event) => {
                setSearchQuery(event.target.value);
                setSearchOpen(true);
              }}
              onFocus={() => setSearchOpen(true)}
              placeholder="跳转页面 / 命令…"
              aria-label="全局搜索"
            />
            <kbd>⌘ K</kbd>
          </div>
          {searchOpen ? (
            <div
              className="command-palette"
              role="dialog"
              aria-label="页面跳转"
            >
              <small>跳转到</small>
              {searchResults.map((item) => (
                <button
                  key={item.id}
                  onClick={() => {
                    choosePage(item.id);
                    setSearchOpen(false);
                    setSearchQuery("");
                  }}
                >
                  <Icon name={item.icon} />
                  <span>{item.label}</span>
                  <kbd>↵</kbd>
                </button>
              ))}
              {searchResults.length === 0 ? <p>没有匹配页面</p> : null}
            </div>
          ) : null}
          <button className="icon-button" aria-label="通知">
            <Icon name="bell" />
            <i className="notification-dot" />
          </button>
          <button
            className="identity-button"
            onClick={() => setCredentialsOpen((value) => !value)}
          >
            <Icon name="user" />
            <span>{session?.principal_id ?? "Unknown"}</span>
          </button>
          {credentialsOpen ? (
            <div className="identity-menu">
              <label>
                <span>Topic ID</span>
                <input
                  value={topicDraft}
                  onChange={(event) => setTopicDraft(event.target.value)}
                  onBlur={commitTopic}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") {
                      event.preventDefault();
                      commitTopic();
                    }
                  }}
                  pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,63}"
                />
              </label>
              <label>
                <span>Reviewer Token</span>
                <input
                  type="password"
                  autoComplete="off"
                  value={reviewerToken}
                  onChange={(event) => setReviewerToken(event.target.value)}
                  placeholder="仅保存在页面内存"
                />
              </label>
              <p>高风险审批需要 Reviewer Token；关闭或刷新页面后清空。</p>
              <Button icon="logout" onClick={() => void logout()}>
                退出登录
              </Button>
            </div>
          ) : null}
        </header>
        <main className="page-content">
          {projectsLoading && projects.length === 0 ? (
            <Loading label="加载授权项目…" />
          ) : projectsError && projects.length === 0 ? (
            <ErrorState error={projectsError} />
          ) : !projectID ? (
            <Empty
              title="没有授权项目"
              detail="请让项目管理员把当前账号加入试点项目。"
            />
          ) : (
            children
          )}
        </main>
      </div>
    </div>
  );
}
