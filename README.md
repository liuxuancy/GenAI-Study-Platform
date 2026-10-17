# LearnHub

简易学习平台的基础骨架，重点是 **登录 + 权限系统**。业务部分（课程、选课、学习进度）只做最小闭环，用来验证权限框架。

- 后端：Go 1.26+、Gin、GORM、SQLite、slog
- 前端（计划中）：React 18、TypeScript、Vite、Ant Design v5、Zustand、React Router v6

> 名词对不上时先查 [docs/glossary.md](docs/glossary.md)。

## 目录结构

```
study_platform/
├── README.md
├── docs/
│   ├── glossary.md                   术语表和命名约定
│   └── permission-cases.json         权限匹配用例，前后端测试共用
├── server/                           Go 后端
│   ├── cmd/server/main.go            启动入口：配置 → 日志 → 汇总模块 → 数据库 → 建表 → 种子数据 → 路由 → 监听 → 优雅退出
│   ├── data/                         本地 SQLite 文件（自动生成，不进 git）
│   └── internal/
│       ├── platform/                 ① 基础设施：和权限、业务都无关，换个项目也能用
│       │   ├── config/               读取环境变量
│       │   ├── db/                   打开 SQLite（WAL、busy_timeout、IMMEDIATE 事务，时间存 UTC）、建表、删除库文件
│       │   ├── logs/                 slog 封装：从 ctx 自动带出 request_id 等字段
│       │   ├── errs/                 业务错误类型和 7 个错误码
│       │   ├── resp/                 统一响应体 {code, msg, data} 和 handler 适配器 resp.H
│       │   └── middleware/           全局中间件：RequestID、Logger、ErrorHandler
│       ├── framework/                ② 权限框架：登录 + 权限，所有业务模块都依赖它
│       │   ├── identity/             你是谁：密码哈希；JWT、认证中间件、白名单（阶段 4）
│       │   ├── access/               你能做什么：权限码匹配、权限码目录、Principal、权限缓存；RequirePermission（阶段 4）
│       │   ├── model/                框架层的两张表：sys_user、sys_role_permission
│       │   ├── module/               Module 接口和 Registry：启动时汇总各模块的权限码、默认授权、表、路由
│       │   ├── seed/                 种子数据（测试账号、默认授权）和演示数据（demo/*.json）
│       │   ├── router/               组装中间件和所有模块的路由
│       │   └── testutil/             测试脚手架：临时数据库 + 建表 + 种子数据
│       └── modules/                  ③ 业务模块，每个模块一个目录（阶段 4 起）
│           ├── account/              登录、退出、me、个人信息、改密码，接口 /api/account/*（阶段 4）
│           ├── system/               用户管理、角色权限，接口 /api/system/*（阶段 5）
│           └── course/               讲师侧 /api/teacher/*、学生侧 /api/student/*（阶段 9）
└── web/                              前端（阶段 6 开始）
    └── src/
        ├── core/                     框架：请求封装、登录状态、权限判断、路由与菜单、布局
        └── modules/                  业务页面：system、course
```

分组规则：

| 目录 | 放什么 | 判断方法 |
|---|---|---|
| `platform/` | 通用基础设施 | 换一个完全不同的项目，这个包能原样拿过去用 |
| `framework/` | 权限框架 | 和"登录 + 权限"相关，所有业务模块都依赖它 |
| `modules/` | 具体业务 | 实现 `module.Module` 接口，在 `enabledModules()` 里登记 |

## 分层与依赖方向

```
modules ──► framework ──► platform
```

规则：

1. **只能从左往右依赖**。`platform` 不依赖 `framework` 和 `modules`；`framework` 不依赖 `modules`。
2. **`errs` 不依赖 HTTP**。service 层用它返回业务错误，单元测试不需要 gin。
3. **业务模块之间不互相 import**，需要跨模块的能力由框架通过 `module.Deps` 提供。
4. **handler 不直接写错误响应**，只返回 `error`，由 `ErrorHandler` 统一转换。

## 一个请求的处理流程

```
http.Server（监听端口、请求头 10 秒超时）
  └─ RequestID      分配请求 ID，写入 ctx 和响应头 X-Request-ID
      └─ Logger      请求结束后记一条日志（状态码、耗时）
          └─ ErrorHandler   把错误和 panic 统一转成 {code, msg, data}
              └─ identity    校验 token、读权限缓存               ← 阶段 4
                  └─ access.RequirePermission("使用方:资源:动作")  ← 阶段 4
                      └─ handler → service（业务逻辑 + 归属检查）→ GORM → SQLite
```

## 权限模型（简述）

- 三个固定角色：`ROLE_ADMIN`、`ROLE_TEACHER`、`ROLE_STUDENT`。每人一个，创建后不可修改。
- 权限码格式为 `使用方:资源:动作`，支持 `teacher:*` 这样的通配码。持有 `ROLE_ADMIN` 时全部放行。
- JWT 只放 `uid` 和 `token_version`。权限每次请求按 uid 查询，进程内缓存 30 秒，权限变更时主动清空。
- 数据归属（如"讲师只能改自己的课程"）在 service 层检查，不只依赖角色权限。

完整说明见 `docs/auth.md`（阶段 10 编写）。

## 快速开始

```bash
cd server
go test ./...
go run ./cmd/server
```

服务默认监听 `:8080`，检查是否启动成功：

```bash
curl http://localhost:8080/api/health
```

首次启动会自动建表并写入测试账号。想要一份干净的、带演示数据的库：

```bash
cd server
go run ./cmd/server --reset-demo
```

它会删除本地数据库，重新建表，写入种子数据和演示数据，然后正常启动服务。只允许在 `APP_ENV=dev` 下使用。

### 测试账号

| 账号 | 密码 | 角色 | 来源 |
|---|---|---|---|
| admin | admin123 | 管理员 | 种子数据，每次启动自动补齐 |
| teacher1 | teacher123 | 讲师 | 种子数据 |
| teacher2 | teacher123 | 讲师 | 种子数据 |
| student1 | student123 | 学生 | 种子数据 |
| student2、student3 | student123 | 学生 | 演示数据，`--reset-demo` 时写入 |

种子数据可重复执行：已存在的账号不会被重置密码；默认授权只在授权表为空（首次启动）时写入，管理员的调整重启后保留。

## 配置（环境变量）

| 变量 | 默认值 | 说明 |
|---|---|---|
| `APP_ENV` | `dev` | `dev` 或 `prod`。prod 输出 JSON 日志，并要求设置 `JWT_SECRET` |
| `HTTP_ADDR` | `:8080` | 监听地址 |
| `DB_PATH` | `data/learnhub.db` | SQLite 文件路径，目录不存在时自动创建 |
| `JWT_SECRET` | 内置开发密钥 | 未设置时启动会打印警告，prod 环境拒绝启动 |
| `JWT_TTL` | `24h` | token 有效期 |
| `PERMISSION_CACHE_TTL` | `30s` | 权限缓存时长，即撤销权限后最迟多久生效 |

## 团队协作约定

- **数据库文件不进 git**。表结构和种子数据写在代码里，每人本地自动生成；想要干净的库就删掉 `server/data/` 后重启。
- **共享的演示数据**写成 JSON 放进 git（框架部分在 `server/internal/framework/seed/demo/`，各模块的放在模块自己的目录，通过 `DemoProvider` 写入），用 `--reset-demo` 加载，不要互相发送 `.db` 文件。演示数据之间用用户名、课程标题这类业务字段关联，不用数据库 id。
- **新增一个模块**：实现 `module.Module` 接口，再在 `cmd/server/main.go` 的 `enabledModules()` 里加一行。
- **需要共同数据时**部署一个服务器实例，大家访问它的接口；本地开发环境不要直接连接服务器上的数据库。
- **测试文件要提交**。`go test ./...` 是验收标准之一。
- 命名约定见 [docs/glossary.md](docs/glossary.md#命名约定)。

## 进度

| 阶段 | 内容 | 状态 |
|---|---|---|
| 0 | 项目初始化 | ✅ |
| 1 | 后端基础设施：配置、数据库、错误、响应、中间件、健康检查 | ✅ |
| 2 | 权限核心库：匹配、目录、Principal、缓存 | ✅ |
| 3 | 模块机制、数据表、种子数据、演示数据加载 | ✅ |
| 4 | 认证：JWT、登录、Authn、RequirePermission | |
| 5 | 系统管理接口：用户、角色权限 | |
| 6 | 前端骨架与登录 | |
| 7 | 动态菜单与路由守卫 | |
| 8 | 管理员页面 | |
| 9 | 课程模块后端（归属检查） | |
| 10 | 课程模块前端、文档与整体验收 | |
