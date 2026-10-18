# CaveClass 术语表

项目里出现的概念、代码中的名字和含义。新增概念时请同步更新这份文件。

## 目录分组

| 代码里的名字 | 中文 | 说明 |
|---|---|---|
| `platform/` | 基础设施 | 和权限、业务都无关的通用包：config、db、logs、errs、resp、middleware |
| `framework/` | 权限框架 | 登录 + 权限：identity、access、model、module、seed、testutil |
| `modules/` | 业务模块 | account、system、course，每个模块一个目录，互不 import |
| `router/` | 路由层 | `router.go` 组装 gin 和全局中间件；`routes.go` 是全部接口的路由表，唯一会 import 所有模块的地方 |
| `identity` | 身份认证 | 回答"你是谁"：密码哈希、JWT、校验 token 的中间件、白名单 |
| `access` | 访问控制 | 回答"你能做什么"：权限码、目录、Principal、缓存、RequirePermission |
| `account` | 账号模块 | 账号相关的接口：登录、退出、me、个人信息、改密码，地址 `/api/account/*` |

## 权限

| 代码里的名字 | 中文 | 说明 |
|---|---|---|
| Permission | 权限码 | 一个字符串，代表一个功能点，格式 `使用方:资源:动作`，如 `teacher:course:create` |
| 使用方（命名空间） | 权限码第一段 | 按使用者划分：`system`（管理员）、`teacher`（讲师）、`student`（学生）。讲师授予 `teacher:*` 不会覆盖学生侧权限 |
| PermissionDef | 权限码定义 | 一个权限码的完整描述：Code、Name、Type、Description。各模块在代码里声明，不存数据库 |
| PermissionType | 权限类型 | `MENU` 控制页面和菜单；`BUTTON` 控制按钮，同时守护对应接口；`API` 只守护接口，没有界面元素 |
| Catalog | 权限码目录 | 所有模块声明的权限码汇总。只收录具体权限码，不收录通配码 |
| Wildcard | 通配码 | 以 `*` 结尾，如 `teacher:*`，匹配该前缀下的所有权限码。只用于授权，不能声明 |
| Match | 匹配 | 判断授予的权限码是否满足要求的权限码。规则见 `docs/permission-cases.json` |
| Grant | 授权 | 把权限码分配给角色，存于 `sys_role_permission` 表 |
| Role | 角色 | `ROLE_ADMIN`、`ROLE_TEACHER`、`ROLE_STUDENT` 三个固定角色。每个用户一个，创建后不可修改 |
| ROLE_ADMIN 放行 | 管理员放行 | 持有 `ROLE_ADMIN` 时代码中直接放行所有权限，不依赖授权数据 |
| Principal | 当前请求的身份 | 当前请求的用户 ID、角色、状态、token 版本和权限码，只用于权限判断，不含昵称等资料 |
| Snapshot | 快照 | 某一时刻从数据库读出的数据副本。缓存里存的 Principal 就是快照 |
| PrincipalLoader | 加载器 | 接口，缓存未命中时负责从数据库加载 Principal。正式实现查库，测试实现返回预设数据 |
| Cache | 权限缓存 | 进程内缓存，按用户 ID 存放 Principal，避免每个请求都查库 |
| TTL | 有效期 | Time To Live，缓存条目最多保留多久，默认 30 秒（`PERMISSION_CACHE_TTL`） |
| Reset | 清空缓存 | 任何影响权限的写操作提交后调用，让新权限在下一个请求立即生效 |
| version（Cache 字段） | 缓存版本 | 每次 Reset 加一，用来丢弃 Reset 之前发起、Reset 之后才返回的查询结果 |
| DBLoader | 数据库加载器 | `PrincipalLoader` 的正式实现：查 `sys_user` 和该角色在 `sys_role_permission` 里的权限码 |
| RequirePermission | 权限中间件 | 写在路由表每一行：要求持有某个权限码，否则 403；权限码格式写错会在启动时 panic |
| RequireLogin | 登录中间件 | 写在路由表里：只要求登录、不要求权限码的接口（如 me、改密码） |
| GetCurrentPrincipal | 取当前用户 | handler 里调用，拿到当前请求的 Principal；没有登录时返回 401 错误 |
| 权限码常量 | 权限码常量 | 如 `system.PermissionUserView`，模块声明和路由表都引用同一个常量，避免两边拼写不一致 |
| StalePermissions | 已失效授权 | 授权表里存在、但代码里已经删掉的权限码。不匹配任何接口，角色权限页标出来供管理员清理 |
| Ownership | 数据归属 | 某条数据属于谁，如课程的 `owner_id`。归属检查在 service 层完成，不满足时返回 403 |

## 认证

| 代码里的名字 | 中文 | 说明 |
|---|---|---|
| JWT / token | 登录凭证 | 登录后签发，只包含 `uid` 和 `tv`（token_version），不包含权限码 |
| token_version（用户表字段） | token 版本 | 每个用户各自的版本号。禁用用户、修改密码、重置密码时加一，该用户的旧 token 立即失效 |
| Whitelist / publicRoutes | 白名单 | 不需要登录就能访问的接口，只在 `identity/public.go` 一处配置（登录、健康检查） |
| TokenIssuer | token 签发器 | 登录时签发 token，认证中间件校验 token；只接受 HS256，校验签名、签发方和过期时间 |
| Authenticate | 认证中间件 | 挂在 `/api` 上：白名单放行；否则校验 token → 读权限缓存 → 检查是否禁用、token 版本，通过后把 Principal 写入请求 |
| CompareWithDummyHash | 耗时对齐 | 用户不存在时也做一次密码比较，让它和"密码错误"耗时一致，避免被用来探测用户名 |
| Dev secret | 开发密钥 | 未设置 `JWT_SECRET` 时使用的内置密钥，仅限本地开发，prod 环境拒绝启动 |

## HTTP 与分层

| 代码里的名字 | 中文 | 说明 |
|---|---|---|
| `http.Server` | HTTP 服务器 | Go 标准库，负责监听端口、解析 HTTP、超时和优雅退出 |
| gin / Engine | 路由框架 | 第三方库，叠在 `net/http` 之上，负责路由、中间件、参数绑定 |
| Middleware | 中间件 | 夹在"收到请求"和"执行 handler"之间的通用处理，按洋葱模型一层套一层 |
| RequestID | 请求 ID | 每个请求分配一个 ID，写入响应头和日志，用来串起同一个请求的所有日志 |
| Logger（中间件） | 请求日志 | 每个请求记一条日志，必须在 ErrorHandler 外层 |
| ErrorHandler | 错误处理中间件 | 把请求处理中出现的错误和 panic 统一转换成 `{code, msg, data}` 响应 |
| Handler | 接口处理函数 | 处理某一个具体接口的函数，写法统一为 `func(ctx) (any, error)` |
| `resp.ToGinHandler` | handler 转换 | 把上面的写法转成 gin 需要的函数，成功时写响应，失败时把错误交给 ErrorHandler |
| Service | 业务层 | 业务逻辑和归属检查，不依赖 HTTP |
| `resp.BindJSON` / `resp.BindQuery` | 参数解析 | 解析并校验请求体 / URL 查询参数，失败返回 400 |
| `resp.GetInt64PathParam` | 路径参数 | 取出 `/users/:id` 里的 id 并转成整数，格式不对返回 400 |
| Result | 统一响应体 | `{code, msg, data}`，code 为 0 表示成功 |

## 模块

| 代码里的名字 | 中文 | 说明 |
|---|---|---|
| Module | 业务模块 | 实现 `module.Module` 接口，向框架声明自己的使用方、权限码、默认授权和数据表。路由不在模块里，统一写在 `routes.go` |
| Namespaces | 模块拥有的使用方 | 一个模块可以拥有多个使用方，如课程模块拥有 `teacher` 和 `student`；两个模块不能拥有同一个使用方 |
| DefaultGrants | 默认授权 | 模块为角色准备的初始授权，只能授予本模块的权限码，不能给 `ROLE_ADMIN` 配授权 |
| Registry | 模块注册表 | 启动时汇总所有模块并检查冲突，任何问题都让启动失败 |
| getEnabledModules | 启用的模块列表 | `cmd/server/main.go` 里的函数，新增模块时在这里加一行 |
| routes.go | 路由表 | 所有接口按模块分组列在一处：方法、路径、所需权限、handler |
| UserDeletionGuard | 删除用户检查 | 可选接口：模块里有数据引用用户（如讲师的课程）时实现它，删除用户前返回 409 拦下 |
| DemoProvider | 演示数据提供者 | 可选接口，模块实现它就能在 `--reset-demo` 时写入自己的演示数据 |

## 错误

| 代码里的名字 | 中文 | 说明 |
|---|---|---|
| `errs.Error` | 业务错误 | 包含 Code、Msg（给用户看）和 cause（只进日志） |
| `System*` 常量 | 系统错误码 | `SystemOK`、`SystemForbidden` 等 7 个，数值同时作为 HTTP 状态码 |
| Sentinel error | 哨兵错误 | 预先定义的错误常量，如 `errs.ErrForbidden`，用 `errors.Is` 按错误码判断 |
| Internal | 内部错误 | 意外错误统一包装成 500，用户只看到"服务器内部错误"，真实原因写日志 |

## 数据

| 代码里的名字 | 中文 | 说明 |
|---|---|---|
| `sys_user` | 用户表 | 用户、角色、状态、token_version |
| `sys_role_permission` | 角色权限表 | 每个角色持有的权限码，可以包含通配码 |
| Seed | 种子数据 | 每次启动都会执行：补齐测试账号（已存在的不动）；授权表为空时写入默认授权。不覆盖管理员的修改 |
| Fixture / 演示数据 | 演示数据 | `internal/framework/seed/demo/` 下的 JSON 文件和各模块的 DemoProvider，团队共享同一份展示数据，用 `--reset-demo` 加载 |
| `--reset-demo` | 重置演示数据 | 启动参数：删除本地数据库，重新建表、写种子数据和演示数据后启动服务，仅限 dev |
| testutil | 测试脚手架 | `testutil.NewDB(t, 模块...)` 返回一个已建表、已写种子数据的临时库，测试结束自动删除 |
| AutoMigrate | 自动建表 | GORM 根据结构体自动建表、加字段；不删字段、不改字段名 |
| WAL | 预写日志模式 | SQLite 的一种模式，读写互不阻塞 |

## 命名约定

- **核心规则：名字要让人一看就知道是做什么的。** 名字长一点不影响性能，可读性提高了，后期维护的价值远大于少打几个字。例如 handler 转换函数叫 `resp.ToGinHandler`，不叫 `resp.H`。
- 变量名写全，不用单字母：gin 的上下文写 `ctx`，不写 `c`。`ctx`、`cfg`、`err` 这类通用缩写可以用；测试参数 `t *testing.T` 是 Go 惯例，保留。
- 概念统一用全称：`Permission`，不缩写成 `Perm`。
- 不在函数里写匿名辅助函数，改为包级具名函数。
- 字段已经属于某个类型时，不再重复类型名：`cache.version`，不写 `cache.cacheVersion`。
- 方法名带上处理的对象，比如写 `LoadPrincipal`，不写泛泛的 `Load`，避免和标准库的同名方法混淆。
- 取值、查询类的函数用 `Get` 开头，如 `GetCurrentPrincipal`、`GetRoleName`、`registry.GetCatalog()`、`testutil.GetUserID`。判断类用 `Is` / `Has`，动作类用动词（`Issue`、`Parse`、`Reset`），Go、GORM、slog 规定的方法名（`Error`、`TableName`、`Handle`）保持原样。
- 构造函数和初始化函数：
  - 有返回值、创建一个对象的，用 `New` 开头，如 `NewRegistry`；修改全局状态、没有返回值的，用 `Setup` / `Init` 开头，如 `logs.Setup`。
  - 只在启动时调用一次的，注释里写明"全局只初始化一次：启动时由 xxx 调用"；没有这句的，表示可以按需多次创建（如测试里的 `testutil.NewDB`）。
- 数据库访问：**查询和写入一律直接写 SQL，建表交给 AutoMigrate**。
  - 查询用 `database.Raw("SELECT ...", 参数...).Scan(&结果)`；写入用 `database.Exec("INSERT / UPDATE / DELETE ...", 参数...)`。
  - 参数一律用 `?` 占位符，不拼接字符串，防止 SQL 注入。
  - `Raw(...).Scan` 查不到不会报错，用 `result.RowsAffected == 0` 判断。
  - 时间字段自己写，统一用 `time.Now().UTC()`。
  - SQL 完整写死，不在代码里拼接字符串。可选的筛选条件写成 `(? = '' OR 条件)`，没传参数时条件恒为真；没传数字类参数用 `-1` 表示。这样一眼能看到最终执行的 SQL，IDE 也能检查。
  - 测试里用 `testutil.MustExec`、`testutil.GetUserID` 少写重复代码。
- 文件开头写一行说明，不单独建 `doc.go`：
  - 单文件包：`// Package xxx ……` 紧贴 `package` 行，既是包说明也是文件说明。
  - 多文件包：每个文件第一行写自己的说明，和 `package` 行之间空一行，否则会被当成包说明拼在一起。
  - 测试文件不写，`xxx_test.go` 测的就是 `xxx.go`。
