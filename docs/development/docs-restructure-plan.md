# Malice Network 文档重构规划

## 背景

当前需要将原始 IoM 文档 (`D:\Programing\blog\chainreactor-docs\docs\IoM\`) 迁移到仓库内部，同时优化现有文档结构。Implant 相关文档已拆分到独立仓库 (`D:\Programing\rust\implant\`)。

## 当前状态分析

### 现有文档结构

```
docs/
├── README.md                          # 中文索引
├── architecture.md                    # 架构概览（已完善）
├── getting-started.md                 # 快速开始
├── deployment.md                      # 部署指南
├── post-exploitation.md               # 后渗透操作手册
├── client/
│   └── commands.md                    # Client 命令总览
├── server/
│   ├── listeners.md                   # Listener 与 Pipeline
│   └── build.md                       # 构建与 Profile
├── implant/
│   └── overview.md                    # Implant 概览（仅边界说明）
├── development/
│   ├── mal/                           # MAL 插件开发文档（完整）
│   ├── custom-pipeline-guide.md       # 自定义 Pipeline
│   └── testing.md                     # 测试文档
└── experiments/
    └── proposal-agent-skills.md       # Agent Skills 提案
```

### 原始 IoM 文档结构

```
IoM/
├── index.md                           # 项目总览
├── concept.md                         # 基本概念
├── design.md                          # 架构与设计
├── quickstart.md                      # 快速开始
├── roadmap.md                         # 路线图
└── guideline/
    ├── deploy.md                      # 部署指南
    ├── listener.md                    # Listener 配置
    ├── payload.md                     # Payload 生成
    ├── post_exploitation.md           # 后渗透操作
    ├── proxy.md                       # 代理配置
    ├── embed_mal.md                   # 嵌入式 MAL
    ├── mal/                           # MAL 相关
    ├── develop/                       # 开发指南
    ├── advance/                       # 高级主题
    └── common/                        # 通用内容
```

## 重构目标

1. **统一文档入口**：保持 `docs/README.md` 作为中文索引，新增 `docs/index.md` 作为项目总览
2. **清晰的层次结构**：按照 CLAUDE.md 规范组织目录
3. **避免重复**：合并重复内容，保留最新最完整的版本
4. **边界清晰**：明确 malice-network 与 implant 仓库的文档边界
5. **保持可维护性**：每个功能模块有对应文档，便于后续更新

## 目标文档结构

```
docs/
├── index.md                           # 项目总览（新增，基于 IoM/index.md）
├── README.md                          # 中文文档索引（保留）
├── concept.md                         # 基本概念（新增，来自 IoM/concept.md）
├── architecture.md                    # 架构概览（保留，已完善）
├── getting-started.md                 # 快速开始（保留，可能需要合并 IoM/quickstart.md）
├── deployment.md                      # 部署指南（保留，可能需要合并 IoM/guideline/deploy.md）
├── roadmap.md                         # 路线图（新增，来自 IoM/roadmap.md）
│
├── client/                            # Client 相关文档
│   ├── README.md                      # Client 概览
│   ├── commands.md                    # 命令总览（保留）
│   ├── session-management.md          # 会话管理（新增）
│   └── post-exploitation.md           # 后渗透操作（移动自根目录）
│
├── server/                            # Server 相关文档
│   ├── README.md                      # Server 概览
│   ├── listeners.md                   # Listener 与 Pipeline（保留）
│   ├── build.md                       # 构建与 Profile（保留）
│   ├── proxy.md                       # 代理配置（新增，来自 IoM/guideline/proxy.md）
│   └── configuration.md               # 配置管理（新增）
│
├── protocol/                          # 协议与通信
│   ├── README.md                      # 协议概览
│   ├── grpc.md                        # gRPC 接口（新增）
│   ├── spite.md                       # Spite 协议（新增）
│   └── mcp.md                         # MCP 集成（新增）
│
├── development/                       # 开发文档
│   ├── README.md                      # 开发指南索引
│   ├── contributing.md                # 贡献指南（新增）
│   ├── testing.md                     # 测试文档（保留）
│   ├── custom-pipeline-guide.md       # 自定义 Pipeline（保留）
│   │
│   ├── mal/                           # MAL 插件开发（保留）
│   │   ├── README.md
│   │   ├── quickstart.md
│   │   ├── builtin.md
│   │   ├── beacon.md
│   │   ├── rpc.md
│   │   └── embed.md
│   │
│   ├── advanced/                      # 高级开发主题（新增）
│   │   ├── architecture-deep-dive.md  # 架构深入
│   │   ├── performance.md             # 性能优化
│   │   └── security.md                # 安全考虑
│   │
│   └── sdk/                           # SDK 与 API（新增）
│       ├── README.md                  # SDK 概览
│       ├── go-sdk.md                  # Go SDK (IoM-go)
│       ├── python-sdk.md              # Python SDK
│       └── typescript-sdk.md          # TypeScript SDK
│
├── operations/                        # 运维与实战（新增）
│   ├── README.md                      # 运维概览
│   ├── best-practices.md              # 最佳实践
│   ├── troubleshooting.md             # 故障排查
│   └── opsec.md                       # OPSEC 指南
│
├── experiments/                       # 实验性功能（保留）
│   ├── README.md                      # 实验性功能索引
│   └── proposal-agent-skills.md       # Agent Skills 提案（保留）
│
└── assets/                            # 资源文件（保留）
    └── showcase/                      # 展示图片
```

## 迁移计划

### Phase 1: 基础结构搭建（优先级：高）

**目标**：建立新的目录结构，迁移核心文档

1. **新增顶层文档**
   - [ ] `docs/index.md` - 基于 `IoM/index.md`，更新项目信息
   - [ ] `docs/concept.md` - 直接迁移 `IoM/concept.md`
   - [ ] `docs/roadmap.md` - 直接迁移 `IoM/roadmap.md`

2. **重组 client/ 目录**
   - [ ] 创建 `docs/client/README.md` - Client 概览
   - [ ] 移动 `docs/post-exploitation.md` → `docs/client/post-exploitation.md`
   - [ ] 合并 `IoM/guideline/post_exploitation.md` 到 `docs/client/post-exploitation.md`
   - [ ] 创建 `docs/client/session-management.md` - 会话管理专题

3. **完善 server/ 目录**
   - [ ] 创建 `docs/server/README.md` - Server 概览
   - [ ] 迁移 `IoM/guideline/proxy.md` → `docs/server/proxy.md`
   - [ ] 创建 `docs/server/configuration.md` - 配置管理
   - [ ] 合并 `IoM/guideline/listener.md` 到 `docs/server/listeners.md`（如有新内容）
   - [ ] 合并 `IoM/guideline/payload.md` 到 `docs/server/build.md`（如有新内容）

4. **更新部署文档**
   - [ ] 合并 `IoM/guideline/deploy.md` 到 `docs/deployment.md`
   - [ ] 合并 `IoM/quickstart.md` 到 `docs/getting-started.md`

### Phase 2: 协议与开发文档（优先级：中）

**目标**：补充协议文档和开发指南

5. **新建 protocol/ 目录**
   - [ ] 创建 `docs/protocol/README.md` - 协议概览
   - [ ] 创建 `docs/protocol/grpc.md` - gRPC 接口文档
   - [ ] 创建 `docs/protocol/spite.md` - Spite 协议说明
   - [ ] 创建 `docs/protocol/mcp.md` - MCP 集成文档

6. **完善 development/ 目录**
   - [ ] 创建 `docs/development/README.md` - 开发指南索引
   - [ ] 创建 `docs/development/contributing.md` - 贡献指南
   - [ ] 创建 `docs/development/advanced/` 目录
   - [ ] 迁移 `IoM/guideline/develop/` 相关内容到 `docs/development/advanced/`
   - [ ] 合并 `IoM/guideline/embed_mal.md` 到 `docs/development/mal/embed.md`
   - [ ] 创建 `docs/development/sdk/` 目录
   - [ ] 创建 SDK 使用文档（Go/Python/TypeScript）

### Phase 3: 运维与 SDK 文档（优先级：中）

**目标**：补充运维实战文档和 SDK 使用指南

8. **新建 operations/ 目录**
   - [ ] 创建 `docs/operations/README.md` - 运维概览
   - [ ] 创建 `docs/operations/best-practices.md` - 最佳实践
   - [ ] 创建 `docs/operations/troubleshooting.md` - 故障排查
   - [ ] 创建 `docs/operations/opsec.md` - OPSEC 指南
   - [ ] 迁移 `IoM/guideline/advance/` 相关内容

### Phase 4: 清理与优化（优先级：低）

**目标**：清理冗余，优化索引

9. **清理工作**
   - [ ] 删除 `docs/implant/overview.md`（仅保留边界说明，或移到 README.md）
   - [ ] 更新所有文档的内部链接
   - [ ] 统一文档格式和风格
   - [ ] 检查并修复所有相对路径引用

10. **优化索引**
    - [ ] 更新 `docs/README.md` - 完整的中文文档索引
    - [ ] 为每个子目录创建 README.md 索引
    - [ ] 添加文档间的交叉引用
    - [ ] 创建快速导航链接

11. **实验性功能整理**
    - [ ] 创建 `docs/experiments/README.md` - 实验性功能索引
    - [ ] 评估 `proposal-agent-skills.md` 是否需要移到正式文档

## 文档边界说明

### malice-network 仓库负责

- **控制平面**：Client、Server、Listener、RPC
- **构建编排**：Profile、Artifact、Pipeline
- **插件生态**：MAL 插件开发、Addon 管理
- **集成接口**：MCP、LocalRPC、SDK
- **运维部署**：配置、部署、监控、故障排查

### implant 仓库负责

- **Implant 实现**：malefic、malefic-pulse、malefic-prelude
- **模块开发**：malefic-modules、malefic-3rd-template
- **工具包**：malefic-win-kit、malefic-linux-kit
- **SRDI 实现**：malefic-srdi
- **交叉编译**：cross-rust

### 交叉引用原则

- malice-network 文档可以引用 implant 仓库链接，但不深入实现细节
- 保持 `docs/implant/overview.md` 作为边界说明，指向外部仓库
- 构建相关文档（`docs/server/build.md`）可以说明如何使用 implant，但不涉及内部实现

## 文档规范

### 文件命名

- 使用 kebab-case：`session-management.md`
- 目录使用小写单数或复数：`client/`, `operations/`
- README.md 作为目录索引

### 文档结构

每个文档应包含：

1. **标题**：一级标题，简洁明确
2. **概述**：简短说明文档内容和目标读者
3. **主体内容**：分层次组织，使用二级、三级标题
4. **相关文档**：列出相关文档链接
5. **示例**：提供实际使用示例（如适用）

### 语言规范

- **文档内容**：中文（当前项目文档主要是中文）
- **代码示例**：英文注释
- **命令示例**：保持原样
- **技术术语**：首次出现时中英文对照

### 链接规范

- 使用相对路径：`[架构概览](../architecture.md)`
- 外部链接使用完整 URL
- 仓库内代码引用：`server/internal/core/session.go`

## 执行时间表

- **Week 1**: Phase 1 - 基础结构搭建
- **Week 2**: Phase 2 - 协议与开发文档
- **Week 3**: Phase 3 - 集成与运维文档
- **Week 4**: Phase 4 - 清理与优化

## 验收标准

1. ✅ 所有原始 IoM 文档内容已评估并迁移或归档
2. ✅ 新文档结构符合 CLAUDE.md 规范
3. ✅ 每个子目录有 README.md 索引
4. ✅ 所有内部链接正确无误
5. ✅ 文档边界清晰，无 implant 实现细节
6. ✅ `docs/README.md` 提供完整的文档导航
7. ✅ 所有文档格式统一，符合规范

## 注意事项

1. **保留历史**：不要删除原始文档，先迁移后归档
2. **增量更新**：每完成一个 Phase 提交一次，便于回滚
3. **交叉验证**：迁移时对比原始文档和当前文档，保留最新最完整的内容
4. **链接检查**：每次迁移后检查所有相关链接
5. **用户反馈**：重构完成后收集用户反馈，持续优化

## 后续维护

- 每个新功能 PR 必须包含对应文档更新
- 定期检查文档与代码的一致性
- 根据用户反馈优化文档结构和内容
- 保持文档的时效性，及时更新过时内容
