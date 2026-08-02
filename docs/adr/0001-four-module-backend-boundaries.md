---
status: accepted
---

# Organize backend ownership into four modules

Multica 将后端长期边界收敛为工作区、认证、素材和系统四个模块，而不是按数据表或每个 HTTP handler 建立独立模块。工作区模块聚合协作服务，认证模块拥有成员与智能体身份，素材模块拥有素材生命周期，系统模块拥有智能体版本发布升级和全局版本化 skill；跨模块只传递稳定契约与 ID，以保留工作区租户隔离并避免共享数据库表成为隐式集成接口。智能体执行生命周期的业务归属尚未确认，本 ADR 不为其分配模块。
