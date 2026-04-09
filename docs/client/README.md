# Client 架构与配置

本目录包含 malice-network Client 的架构说明和配置文档。

## 文档列表

- [命令总览](commands.md) - Client 命令参考
- [配置说明](configuration.md) - Client 配置参考

## Client 架构

Client 是 malice-network 的操作入口，提供 CLI/TUI 交互界面，负责：
- 命令派发与会话管理
- 插件加载与本地集成
- MCP/LocalRPC 接口暴露

详细架构说明请参考 [系统架构](../architecture.md)。

## 使用指南

关于 Client 的具体使用方法，请参考：
- [快速开始](../getting-started.md) - 快速上手指南
- [操作指南](../operations/) - 完整操作手册
- [后渗透操作](../operations/post-exploitation/) - 后渗透操作指南

## 相关文档

- [架构概览](../architecture.md) - 系统架构说明
- [Server 架构](../server/) - Server 端架构与配置
- [开发文档](../development/) - 开发与扩展指南

