# 协议文档

本文档说明 malice-network 的通信协议体系，包括 gRPC 服务、Spite 消息协议和 Parser 机制。

## 协议分层

```
┌───────────────────────────────────────────────┐
│              Client / SDK / MCP                │
├───────────────────────────────────────────────┤
│          gRPC / mTLS (控制面)                   │
│   MaliceRPC · RootRPC · ListenerRPC           │
├───────────────────────────────────────────────┤
│        Spite 消息协议 (数据面)                   │
│   Header → Body (protobuf Spites)             │
├───────────────────────────────────────────────┤
│          Parser (协议解析)                       │
│   Malefic Parser · Pulse Parser               │
├───────────────────────────────────────────────┤
│        Encryption (传输加密)                     │
│   AES-CFB · XOR · TLS                         │
├───────────────────────────────────────────────┤
│          Transport (传输层)                      │
│   TCP · HTTP · REM · Bind                     │
└───────────────────────────────────────────────┘
```

## gRPC 服务

Server 通过 gRPC 暴露三类服务：

| 服务 | Proto 定义 | 调用方 | 说明 |
|------|-----------|--------|------|
| **MaliceRPC** | `services/clientrpc/service.proto` | Client / SDK | Session、Task、Build、Module 等全部操作 |
| **RootRPC** | `client/rootpb/root.proto` | 本地管理 | Operator 管理、Listener 增删（仅 localhost） |
| **ListenerRPC** | `services/listenerrpc/service.proto` | Listener | Pipeline 注册、SpiteStream/JobStream 双向流 |

### Proto 定义位置

Proto 文件位于 `external/IoM-go/generate/proto/`：

| Proto 文件 | 用途 |
|-----------|------|
| `client/clientpb/client.proto` | Client 消息定义 |
| `client/rootpb/root.proto` | Admin 服务定义 |
| `implant/implantpb/implant.proto` | Implant 协议 |
| `services/clientrpc/service.proto` | Client RPC 服务 |
| `services/listenerrpc/service.proto` | Listener RPC 服务 |

!!! warning "修改规范"
    Proto 变更在 `external/IoM-go` 子模块内进行，不要手动编辑生成的 Go 代码。变更后需更新子模块引用并执行 `go mod tidy`。

## Spite 消息协议

Spite 是 Implant 与 Server 之间的核心通信消息单元，基于 protobuf 序列化。

### 消息结构

```
┌──────────┬──────────┬─────────────────────┐
│  Header  │  Length  │   Body (protobuf)    │
│  1 byte  │  4 bytes │   variable length    │
└──────────┴──────────┴─────────────────────┘
```

- **Header**：协议标识字节，用于 `DetectProtocol` 自动识别 Implant 类型
- **Length**：Body 长度（大端序）
- **Body**：protobuf 序列化的 `Spites` 消息（可包含多个 `Spite`）

### 协议标识

| Header 字节 | 协议类型 | 说明 |
|-------------|---------|------|
| `0xd1` | Malefic | 主 Implant 协议 |
| `0x41` | Pulse | 轻量 Implant 协议 |

### 分块传输

大数据量传输通过分块机制处理：

- `Count(size)` — 计算给定大小需要的分块数
- `Chunked(content)` — 将内容按固定大小切分为 channel 流式输出
- `SpitesCache` — 缓存多个 Spite 消息后批量发送

## Parser 机制

Parser 是 Pipeline 与 Implant 之间的协议解析层，负责将原始字节流解析为结构化的 Spite 消息。

### PacketParser 接口

```go
type PacketParser interface {
    ReadHeader(conn io.ReadWriteCloser) (uint32, uint32, error)
    Parse([]byte) (*implantpb.Spites, error)
    Marshal(*implantpb.Spites, uint32) ([]byte, error)
}
```

| 方法 | 职责 |
|------|------|
| `ReadHeader` | 从连接中读取消息头，返回消息长度和序列号 |
| `Parse` | 将字节数组反序列化为 Spites 消息 |
| `Marshal` | 将 Spites 消息序列化为字节数组 |

### Parser 类型

| Parser | 对应 Implant | 特点 |
|--------|-------------|------|
| **malefic** | Malefic（主 Implant） | 完整协议支持 |
| **pulse** | Pulse（轻量 Implant） | 精简协议 |
| **auto** | 自动检测 | 通过 `DetectProtocol` 根据 Header 字节自动选择 |

### 配置方式

在 Pipeline 配置中指定 Parser：

```yaml
tcp:
  - name: tcp
    parser: auto          # auto / malefic / pulse
```

!!! tip "auto 模式"
    推荐使用 `auto`，Pipeline 会根据连接的第一个字节自动识别 Implant 类型并选择对应 Parser。

### 实现位置

| 文件 | 职责 |
|------|------|
| `server/internal/parser/parser.go` | PacketParser 接口、协议检测、读写函数 |
| `server/internal/parser/malefic/parser.go` | Malefic 协议实现 |
| `server/internal/parser/pulse/parser.go` | Pulse 协议实现 |
| `server/internal/parser/chunk.go` | 分块传输机制 |

## Encryption 机制

Pipeline 与 Implant 之间的通信加密，独立于 TLS 层：

| 类型 | 实现 | 说明 |
|------|------|------|
| **AES** | AES-CFB 模式 | 对称加密，密钥需与 Implant profile 一致 |
| **XOR** | 流式 XOR | 轻量加密，适合低开销场景 |

支持多层加密叠加（如同时启用 AES + XOR）。

配置见 [Listener 架构 - Encryption 机制](../../server/listeners.md#encryption-机制)。

## MCP 协议

MCP (Model Context Protocol) 用于将 IoM 能力暴露给外部 AI Agent。

Client 通过 `--mcp <addr>` 启动 MCP Server，自动将命令树注册为 MCP Tools 和 Resources。

详细说明见 [AI Agent 集成](../../client/agent.md)。

## 相关文档

- [核心概念](../../concept.md) - 系统架构说明
- [Server 内部机制](../../server/internals.md) - RPC 通信详解
- [Listener 架构](../../server/listeners.md) - Pipeline 与 TLS/Encryption 配置
- [开发指南](../) - 开发文档索引
