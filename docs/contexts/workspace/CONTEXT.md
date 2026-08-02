# Workspace

工作区（Workspace）是团队协作与交付的租户上下文。所有工作对象都属于一个工作区，并受该工作区的成员资格和权限约束。

## Language

**Workspace**:
团队协作、授权和数据隔离的边界。
_Avoid_: Tenant、Organization、Space

**Project**:
组织目标、负责人、进度、资源引用和相关工作项的交付容器。
_Avoid_: Workspace、Folder

**Todo**:
团队成员维护的轻量待办，可关联项目或 issue，但不表示智能体的一次执行。
_Avoid_: Agent Task、Execution、Run

**Issue**:
团队协作的问题或工作对象，拥有编号、状态、层级、负责人和讨论历史。
_Avoid_: Todo、Agent Task

**Knowledge**:
经证据、审阅和发布形成的工作区知识。
_Avoid_: Asset、Attachment、Raw File

**Requirement**:
项目的目标、约束和验收要求，可追踪到相关 issue。
_Avoid_: Issue Description、Todo

**Workspace Setting**:
影响单个工作区行为和展示的配置。
_Avoid_: System Configuration、User Preference

**Project Actor Relation**:
项目与成员或智能体之间带角色的权威关系。
_Avoid_: Generic Entity Link、Project Lead Field

**Project Actor Role**:
项目参与者关系中的职责，规范值为 `lead`、`member` 或 `agent`。
_Avoid_: Workspace Role、Permission
