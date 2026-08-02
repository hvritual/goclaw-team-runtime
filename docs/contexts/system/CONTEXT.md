# System

系统（System）是智能体可发布/升级的软件版本和全局 skill 目录的上下文。它不拥有团队成员资格、智能体身份或工作区内的启用配置。智能体执行生命周期的业务归属尚未确认。

## Language

**Agent Release**:
面向智能体发布的、可校验并可升级的软件交付物集合。
_Avoid_: Agent、Runtime Session、Execution

**Agent Version**:
智能体版本发布使用的不可变版本标识。
_Avoid_: Agent ID、Runtime Profile

**Upgrade Policy**:
决定可用目标版本和升级约束的系统规则。
_Avoid_: Workspace Setting

**Skill Definition**:
系统级 skill 的稳定身份与说明。
_Avoid_: Workspace Skill Configuration、Agent Binding

**Skill Version**:
skill 内容与文件集合的一次不可变发布。
_Avoid_: Skill Definition、Asset Version

**Skill Publication**:
使一个 skill 版本可被工作区引用的业务事实。
_Avoid_: Workspace Enablement
