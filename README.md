# CaveClass

GenAI 学习平台：讲师用 AI 生成课程并发布到班级，学生学习、测验并获得 AI 辅导，管理员管理用户、班级、课程和 AI 用量。

- 后端：Go、Gin、GORM、SQLite（WAL）
- 前端：React 18、TypeScript、Vite、Ant Design v5、Zustand
- 线上：https://caveclass.umberk.cn/

> 名词对不上时先查 [docs/glossary.md](docs/glossary.md)。

## 目录

```
backend/
  cmd/server/            启动入口
  internal/
    platform/            基础设施：配置、数据库、错误码、统一响应、中间件、COS
    framework/           登录与权限：JWT、权限码、模块注册、种子数据
    modules/             业务模块：account、system、course
    ai/                  AI 子系统：prompt 模板、模型调用、JSON 解析与校验（不访问数据库）
    usage/               AI 调用用量记录
    router/              routes.go 是全部接口的路由表
frontend/src/
  core/                  请求封装、登录状态、路由、权限、多语言
  modules/               admin、teacher、student、workspace 等页面
docs/
  sql/                   建表、种子数据、迁移
  deploy/                nginx、systemd、部署脚本
  UI/                    HTML 原型
```

依赖只能从左往右：`router → modules → framework → platform`；业务模块之间不互相 import。

## 本地运行

```bash
cd backend
go test ./...
go run ./cmd/server          # 默认 :8080，AI 默认用 mock，不需要 key
```

```bash
cd frontend
pnpm install
pnpm dev                     # http://localhost:5173
```

环境变量见 [backend/.env.example](backend/.env.example)（基础、AI、COS、日志四组）。

## 测试账号

线上库由 `docs/sql/seed.sql` 初始化：

| 账号 | 密码 | 说明 |
|---|---|---|
| admin | admin123 | 管理员 |
| teacher1 ~ teacher3 | teacher123 | 讲师；teacher3 不在任何班级 |
| student1 ~ student6 | student123 | 学生；student4 已停用，登录应失败 |

本地 `go run` 启动时会写入一套较小的种子账号；`--reset-demo`（仅 dev）会重建本地库并写入演示数据。

## 部署

见 [docs/deploy/](docs/deploy/)：首次用 `bootstrap.sh`，之后用 `deploy.sh` 拉代码、构建、重启服务。

## 协作约定

- 数据库文件不进 git；表结构和种子数据在代码与 `docs/sql/` 里。
- 新增模块：实现 `module.Module`，在 `getEnabledModules()` 登记；新增接口只改 `internal/router/routes.go`。
- 测试文件要提交，`go test ./...` 是验收标准。
