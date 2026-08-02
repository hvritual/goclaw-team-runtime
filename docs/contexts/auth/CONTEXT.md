# Auth

认证（Auth）是团队参与者身份、成员资格和授权的上下文。它拥有成员与智能体的身份事实，但不拥有这些参与者在项目中的关系。

## Language

**Member**:
加入某个工作区的人类参与者，具有工作区角色。
_Avoid_: User、Project Member、Agent

**User**:
可登录产品并在一个或多个工作区中成为成员的人类身份。
_Avoid_: Member、Actor

**Agent**:
可加入团队并被业务对象引用的智能参与者身份。
_Avoid_: Agent Release、Runtime、Execution

**Workspace Role**:
成员在工作区中的授权角色，规范值为 `owner`、`admin` 或 `member`。
_Avoid_: Project Actor Role

**Membership**:
用户与工作区之间带角色的有效关系。
_Avoid_: Project Actor Relation

**Last Owner**:
工作区中最后一位 owner；任何操作都不得使工作区没有 owner。
_Avoid_: Project Lead
