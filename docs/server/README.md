# Server 架构与配置

本目录包含 malice-network Server 的架构说明和配置文档。

## 文档列表

- [Listener 架构](listeners.md) - Listener 与 Pipeline 架构
- [构建系统架构](build.md) - Artifact 构建系统架构
- [配置说明](configuration.md) - Server 配置参考

## Server 架构

Server 是 malice-network 的控制核心，负责：
- 状态管理与任务编排
- RPC 服务与审计通知
- Listener/Pipeline 管理
- 构建控制与 Profile 管理

详细架构说明请参考 [系统架构](../architecture.md)。

## 使用指南

关于 Server 的具体使用方法，请参考：
- [部署指南](../operations/deployment.md) - Server 部署与配置
- [Listener 操作](../operations/listener.md) - Listener 配置与管理
- [构建操作](../operations/build.md) - Payload 构建与 Profile 配置

## 相关文档

- [架构概览](../architecture.md) - 系统架构说明
- [Client 架构](../client/) - Client 端架构与配置
- [操作指南](../operations/) - 完整操作手册
- [开发文档](../development/) - 开发与扩展指南

