# Execution / Runtime

执行运行时（Execution/Runtime）是受控执行尝试的上下文。它拥有一次
Run 从排队到验证交接的操作状态，但不拥有工作意图、参与者身份、发布物
或最终验收。

## Ownership

Runtime owns:

- Run identity and the frozen workspace/project/work-item revision reference;
- queue state, claim, exclusive lease, heartbeat, attempt number, bounded retry,
  cancellation, timeout, and termination;
- selected Agent Identity and pinned Agent Release/Skill version references;
- opaque `secret://` references, never secret values;
- runtime logs, artifact references, and execution Evidence references;
- validation handoff after execution ends.

Runtime does not own:

- Workspace Todo, Issue, Requirement, priority, assignee, or acceptance state;
- Auth Agent Identity, membership, role, or authorization truth;
- System Agent Release publication, upgrade policy, or Skill publication;
- Space Asset bytes, retention, or access lifecycle;
- deterministic Check results or human DoneGate acceptance.

## Language

**Run**:
对已冻结工作项修订的一次受控执行尝试。
_Avoid_: Todo、Task、Issue、Agent

**Attempt**:
Run 在有界重试策略中的一次执行编号。
_Avoid_: Run、Revision、Retry Policy

**Claim**:
一个已授权执行者请求取得 Run 执行权的动作。
_Avoid_: Assignment、Membership、Acceptance

**Lease**:
对单个 Run Attempt 的限时排他执行权。
_Avoid_: Workspace Lock、Ownership、Authorization

**Heartbeat**:
当前 Lease 持有者证明执行仍存活并延长租约的运行时信号。
_Avoid_: Progress、Audit Event、Acceptance

**Retry**:
在策略允许时为失败或租约过期的 Run 启动后续 Attempt。
_Avoid_: Reopen Todo、ChangeIntent

**Execution Evidence**:
由 Run 产生并提交给独立验证的日志、差异、测试、制品等不可变引用。
_Avoid_: Check Result、Human Acceptance、Done

**Runner**:
承载受控进程、文件、网络和 Secret 注入策略的执行节点。
_Avoid_: Agent Identity、Agent Release、Reviewer

## Todo versus Run

- Todo 属于 Workspace，表达团队想完成的工作，可由人或 Agent 负责，也可
  在没有自动执行时长期存在。
- Run 属于 Runtime，表达对某个已冻结 Todo/Task 修订的一次执行尝试。
- 一个 Todo 修订可以没有 Run，也可以因重试或不同执行策略产生多个 Run。
- Run 的取消、失败或超时不会自动关闭或取消 Todo。
- Run 成功只产生结果和 Evidence，并转交验证；它不能自行把 Todo 标为完成。

## Agent boundaries

**Agent Identity — Auth owned**:
稳定的参与者身份、成员资格、工作区角色和授权事实。Runtime 只引用并在
Claim 时重新验证。

**Agent Release — System owned**:
不可变的软件、模型/提示、工具清单和 Skill 版本组合。Runtime 在 Run
创建时固定版本，不发布或升级它。

**Agent Execution — Runtime owned**:
将一个 Agent Identity、一个不可变 Agent Release、一个冻结工作项修订
绑定到 Run/Attempt/Lease 的执行事实。它不改变身份、发布物或任务意图。

## State and invariants

A Run follows the controlled operational lifecycle implemented by the P2
control plane: queue, claim, heartbeat, completion/validation handoff, cancel,
lease expiry, and bounded retry.

Hard invariants:

1. Run and Attempt IDs are project/workspace scoped.
2. Only one unexpired Lease may own an Attempt.
3. Claim revalidates identity, membership, project authorization, capability,
   and current lease state.
4. Heartbeat and completion are accepted only from the current Lease holder.
5. Secret values never enter events, Evidence, audit, or logs.
6. Completion requires Evidence and leads to validation, never direct Done.
7. Runner, Agent, assignee, creator, or model output cannot self-accept.
8. Every retry is bounded and auditable.

## Relationships

- Workspace sends Runtime a frozen work-item revision and receives execution
  state plus Evidence references.
- Auth supplies Agent/Runner identity and authorization truth.
- System supplies immutable Agent Release and Skill versions.
- Space supplies Asset storage and access lifecycle for logs/artifacts.
- Delivery Kernel records runtime events and hands Evidence to independent
  Checker and human DoneGate without transferring acceptance authority.
