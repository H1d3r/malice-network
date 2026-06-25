# 审计导出

Client 通过 `audit` 导出某个 Server session 的结构化任务审计记录。当前导出聚焦 Task 数据，不包含 `.malice/audit/<sessionID>.log` 里的 RPC 文本日志。

## 概览

审计导出由 Server 端的 Task 数据组装：

- Task 元数据：task id、类型、进度、调用者、完成状态、创建时间、完成时间、请求摘要与 request hash。
- Task response：从 context task 目录读取 implant 回传的 `Spite` 结果。
- Task request：当 `server.audit > 1` 时，导出 raw request `Spite`。
- Task result：Client 使用现有 task renderer 将 response 渲染为可读文本。

## 用法

导出 JSON：

```bash
audit session <session_id>
```

也可以使用简写：

```bash
audit <session_id>
```

导出 HTML：

```bash
audit session <session_id> -o html
```

指定输出文件：

```bash
audit session <session_id> -o json -f audit.json
```

## 字段

JSON 导出包含：

| 字段 | 含义 |
|------|------|
| `session` | Session ID |
| `task` | Task ID |
| `type` | Task 类型 |
| `status` | Task 状态值 |
| `command` | 兼容旧导出的命令描述 |
| `commandSummary` | 一行命令摘要 |
| `callby` | 调用该任务的 client |
| `total` / `cur` | 任务进度 |
| `taskFinished` | Task 是否完成 |
| `timeout` | Task 是否超时 |
| `created` / `finished` / `lasted` | 兼容旧导出的格式化时间字符串 |
| `createdAt` / `finishedAt` | Unix 时间戳 |
| `requestSummary` | 结构化 request 摘要 |
| `requestSize` | raw request protobuf 字节数 |
| `requestSha256` | raw request protobuf 的 SHA-256 |
| `hasRequest` | Server 是否保存了 request cache |
| `resultIndex` | 多结果任务中的 response 索引 |
| `response` | raw response `Spite` |
| `request` | raw request `Spite`，仅在 Server 返回时存在 |
| `taskResult` | Client renderer 渲染后的可读结果 |

## 配置

`server.audit` 控制审计详细程度：

- `1`：导出 Task 元数据和 response。
- `> 1`：额外导出 raw request。

审计导出不读取 RPC 文本日志；RPC 文本日志仅用于服务端本地排障。
