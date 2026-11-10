# 学生
表由 AutoMigrate 建出，没有声明外键。

## 学生端入口与页面

学生端页面路由定义在 `frontend/src/core/router.tsx`，页面组件在 `frontend/src/modules/student/`。

| 路由 | 功能 | 主要数据来源 |
|---|---|---|
| `/student` | 学习首页：课程/测验统计、继续学习、近期课程、学习任务、学习趋势和建议 | 课程列表、套题列表、学习中心汇总 |
| `/student/courses` | 我的课程：按来源、进度筛选，按标题搜索/排序；创建个人课程 | 当前学生可见课程 |
| `/student/courses/:courseId` | 课程工作区：课程内容/气泡图、学习进度、课程资源、章节 AI 问答；课程所有者可分享和生成内容 | 可见课程详情及其进度 |
| `/student/quizzes` | 测验中心：课程/章节/创建方式/状态筛选，题目列表，创建个人套题 | 当前学生可见或本人创建的套题 |
| `/student/questions/:questionId` | 答题页：逐题作答、保存、交卷、查看成绩和解析 | 套题详情与当前学生答题记录 |
| `/student/ai` | 通用 AI 工作区：技能、对话、课程上下文 | 工作区技能、本人对话、可选课程 |
| `/student/achievements` | 成就墙：未解锁成就只显示灰色占位图标 | 本人成就记录 |
| `/student/account` | 个人资料与密码管理 | 当前登录账号 |

`RequireAuth` 只负责检查 token；有 token 但本地没有用户资料时会请求 `/api/account/me`。`StudentLayout` 和部分独立页面再检查 `currentUser.role === 'ROLE_STUDENT'`，不符合时跳转到该角色首页。前端路由判断用于导航和 UI；后端仍需独立校验角色、课程可见性和资源归属。

课程工作区顶部面包屑显示“首页 > 我的课程 > 当前课程”；前两项分别链接至 `/student` 和 `/student/courses`。

## 功能与 API

下表路径省略统一前缀 `/api`。`RequireLogin` 表示必须有有效登录态，不等同于学生权限；业务接口还会由 handler/service 校验角色及数据范围。

| 功能 | 前端调用的 API | 调用时机与用途 |
|---|---|---|
| 学习首页 | `GET /student/courses`、`GET /student/questions`、`GET /student/learning/dashboard` | 首页并行加载课程、套题和任务/趋势汇总；课程与套题列表也用于计算前端统计 |
| 学习建议 | `POST /student/learning/insights`；AI 提供商选择还会调用 `GET /ai/providers` | 用户主动生成 AI 建议；请求传 `provider` 和语言。每个用户的 AI 建议会保存在浏览器本地，后续优先显示缓存，否则显示规则建议 |
| 我的课程 | `GET /student/courses`、`POST /student/courses` | 页面加载课程；创建对话框提交标题和描述，创建为私有课程 |
| 课程学习 | `GET /student/courses/:id`、`PUT /student/courses/:id/progress` | 打开详情取课程树；完成活动时提交百分比 |
| 成就 | `POST /student/achievements/sync` | 同步已有小人与好感度、获取已解锁成就并领取待显示通知；课程创建、课程进度及答题更新后重新同步 |
| 课程分享/AI 生成 | `PUT /student/courses/:id/sharing`、`POST /student/courses/:id/generate`、`GET /ai/providers` | 仅课程所有者显示分享与生成按钮；分享切换同班可见性，生成/重新生成课程内容 |
| 课程资料 | `GET /student/courses/:id/resources`、`POST /student/courses/:id/resources` | 学生仅能上传自己创建的课程资料；上传资料在课程共享给同班后可见 |
| 章节问答 | `POST /student/courses/:id/chapters/:chapterKey/ask`、`GET /student/courses/:id/conversations`、`GET /student/courses/:id/conversations/:conversationId`、`DELETE /student/courses/:id/conversations/:conversationId` | 课程内 AI 助手按课程、章节及当前用户维护对话历史；提问可带活动、小节选中文字和 conversationId |
| 学习时长 | `POST /student/courses/:id/learning/heartbeat` | 仅学生打开真实课程页时上报可见页面时长；切后台、离开页面或累计到阈值时 flush |
| 课程资源 | `GET /student/courses/:id/resources`、`GET /courses/:courseId/resources/:resourceId/preview`、`GET /courses/:courseId/resources/:resourceId/download` | 资源面板按文件类型、章节、搜索词筛选已发布资源；预览/下载返回短时 URL |
| 测验中心/套题 | `GET /student/courses`、`GET /student/questions`、`GET /student/courses/:id`、`POST /student/questions` | 页面加载套题和课程；按章节分组时取课程树；创建套题时前端先生成本地预览，用户确认后才将题目提交保存 |
| 答题 | `GET /student/questions/:id`、`PUT /student/questions/:id/contents/:contentId` | 加载单套题；保存答案时传 `option`，最后一次提交可同时传 `submit: true` |
| 通用 AI 工作区 | `GET /student/courses`、`GET /workspace/skills`、`GET /workspace/conversations`、`GET /workspace/conversations/:id`、`PUT /workspace/conversations/:id`、`DELETE /workspace/conversations/:id`、`POST /workspace/messages` | 加载可选课程、技能和本人对话；发送消息可附课程、技能、参数和 provider。`/workspace/*` 是学生/教师共用能力，不是学生专属 API |
| 个人账号 | `GET /account/me`、`PUT /account/profile`、`PUT /account/password`、`POST /account/logout` | `/me` 由登录守卫初始化；账号页更新个人资料、修改密码或退出 |

成就包括首次成功创建课程像素小人，以及根据学生当前可见的不同课程的好感度判定：一门达到 50%、一门达到 100%、五门达到 100%。解锁记录永久保留；同步时会补算已有小人和进度。首次解锁通知在学生页面右上角显示 3 秒，服务端记录领取状态以避免跨设备重复通知。

## 学生课程创建与读取时序

### 创建课程：提交成功时立即写入数据库

1. 首页或“我的课程”打开 `CreateCourseDialog`。前端提交前只检查课程名非空，并去除课程名和描述首尾空白。
2. 用户选择小人生成模型后，点击创建调用 `POST /student/courses`，请求包含 `{ title, description, provider }`；界面只提供服务端配置的真实模型，不提供 mock。没有可用真实模型时不能创建课程，并提示服务端尚未配置模型。未指定 `provider` 的旧客户端继续使用后端默认 AI。界面展示的专业、课程类型没有随请求发送：专业由后端按当前用户 ID 从 `sys_user.major` 读取，课程类型由后端固定为 `structured`。
3. 后端从认证上下文取得当前用户，校验学生角色、课程名必填和长度、描述长度，然后向 `course` 插入课程基本信息：
   - `title`、`description`：来自请求（空描述存为 NULL）。
   - `owner_id`：当前登录用户；`major`：当前用户账号上的专业（空值存为 NULL）。
   - `course_type`：`structured`；`is_publish`：`0`（私有）。
   - `created_at`、`updated_at`：当前 UTC 时间。
   - 此时不写课程正文 `content_json` 或 `course_class` 指派记录。
4. 课程行插入后，后端从固定词表随机抽取 64 组候选属性（角色、体型、发色、发型、五官、配饰、轮廓、性格），排除已使用的完整组合；以各候选与最相似已有小人的加权外观差异选出最高分，性格不参与评分。同分优先选总体较少使用的外观属性，仍同分则随机选择；候选全部重复时提示重试，并发插入冲突时重新抽取。该策略只提高属性差异，不保证 AI 实际画出的图片一定不同。选定属性后调用 AI 生成该角色的初始像素图、25%、50%、75%、100% 好感度表情动作及符合性格的中英双语对白，写入 `course_pixel_avatar`。输出解析或校验失败时按 `AI_MAX_RETRIES` 重新生成，可能产生额外用量；最终失败则清理课程。属性组合有唯一约束。同时为所有者插入 `student_course_progress`：`percent` 为 0，`affection` 为 0。
5. AI 未配置、网格非法、重试后仍然冲突、组合耗尽或请求在创建完成前中断时，后端使用独立上下文事务性删除该课程及已写入的形象、进度；清理失败会记录错误并返回内部错误。创建期间关闭弹窗不会中止请求，重新打开可看到原有输入和创建状态。成功时若弹窗仍打开则导航到课程；已关闭时静默刷新列表。失败时显示提示，保留输入并允许重试。课程标题有唯一约束，重复时返回冲突。关闭或刷新浏览器可能中断请求，无法保证后台继续创建。

因此，“创建课程”不是只在浏览器暂存草稿：提交成功时课程基本信息已经持久化到后端配置的 SQLite 数据库（默认 `backend/data/caveclass.db`，可由 `DB_PATH` 修改）。AutoMigrate 在服务启动时建表，不是创建每门课程时建表。

### 什么时候从数据库读取

| 场景 | 读取动作 |
|---|---|
| 打开学习首页 | 并行请求 `GET /student/courses`、`GET /student/questions`、`GET /student/learning/dashboard`；首页课程卡片和统计使用这些响应 |
| 打开“我的课程” | 页面挂载时请求 `GET /student/courses`；本地筛选、搜索、排序只处理该响应 |
| 创建完成后进入详情 | `POST` 返回 ID 后导航到详情路由；`CourseBubblePage` 检查学生角色和 courseId 后，再请求 `GET /student/courses/:id` 读取完整详情 |
| 刷新或重新进入详情 | 再次请求 `GET /student/courses/:id`，课程内容不只依赖 React 内存 |
| 重新进入列表/首页 | 页面加载时重新请求课程列表，不依赖创建前的列表缓存 |

`GET /student/courses` 查询 `course` 并左连接当前学生的 `student_course_progress`；没有进度行就返回 `percent: 0`。查询只返回当前学生拥有、同班共享或被指派到其班级的课程。

`GET /student/courses/:id` 根据同一可见性规则查询课程，再读取该课程的章节测验映射、课程所有者的同班学生数，以及当前学生该课程的进度。课程正文从 `course.content_json` 解析为响应的 `tree`；正文为空时 `tree` 为 `null`。不存在或当前学生不可见的课程按 404 返回。

### 正文、进度及其他关联数据何时写入

- **课程正文**：新建课程时为空。所有者在详情页明确触发 AI 生成/重新生成后，`POST /student/courses/:id/generate` 才会生成正文。成功时后端事务更新 `course.description`、`audience`、`language`、`content_json` 并替换生成的章节测验；事务成功后再将 JSON 同步到 `course_chapter`、`course_section` 等派生结构。生成接口返回更新后的详情；成功后自动关闭生成弹窗并在页面显示生成结果，失败时保留弹窗和错误提示。之后重新进入页面会通过 GET 再读取。已登记的 `PUT /student/courses/:id/content` 可保存正文，但当前学生 UI 没有直接调用它。
- **章节/小节结构**：`content_json` 是权威课程正文；`course_chapter`、`course_section` 是保存或生成正文后同步的派生数据，不是单独编辑的正文来源。
- **课程导图**：下层气泡沿上层气泡周围的圆弧排列，按已展开气泡和线段的实际位置避让；在紧凑布局基础上将边缘间连线加倍，必要时进一步增加间距以避免碰撞。根气泡和第二层不再自动放大，第三层气泡适度放大。根据展开范围和可用空间适度缩小，内容较多时保持最小尺寸并可拖动画布或滚动查看，因此屏幕上的连线长度也会受到缩放影响。教师端导图复用同一布局。已完成的小节显示与进度条相同的颜色，所有子气泡完成后上层气泡也变色；状态依据本浏览器保存的小节完成明细，而非服务端总进度推测。课程像素小人按当前好感度选择形象，沿已展开线段往返并沿气泡边缘切换路线；可以用鼠标或触屏提起小人，松手后吸附到最近的连接线或可见气泡边缘，从落点沿随机方向继续行走。提起期间暂停计时；若原本正在休息，放下后继续剩余休息。随机走动 15–30 秒后休息 2–5 秒，休息时显示 2 秒对白。2 秒内点击小人 3 次显示“不要再碰”的对白 2 秒，优先于休息对白，拖动不算点击。没有展开线段时小人停在根气泡附近，拖放后也回到该位置；无像素形象的课程不显示小人，系统设置减少动态效果时拖放后保持静止，页面隐藏时不进行休息调度。
- **课程资料**：课程所有者可在资料页上传单个文件，关联到整门课程；使用与教师上传相同的文件类型、大小限制和私有对象存储。资料立即可由所有者查看，课程分享给同班后同班学生可查看和下载，取消分享后不能再通过平台获取；已签发的短时下载链接在到期前仍可能有效。其他学生不能上传，上传失败会显示错误。
- **学习进度**：课程页先把完成 activity 的 ID 存在浏览器 `localStorage`（按用户和课程区分），并据课程树叶节点计算本地比例。完成学习动作时调用 `PUT /student/courses/:id/progress` 将百分比写进 `student_course_progress`，数据库键为 `(user_id, course_id)`；之后列表和详情再读此百分比。显示/提交时取服务端比例与本地计算比例的较大值。清理本地存储会丢失 activity 明细，但已成功写入数据库的百分比仍保留。好感度基础值为课程进度的 80% 加该课习题进度的 20%，结果向下取整；保存进度或习题答案时按基础值变化增减好感度，并保留已经发生的行为加减。
- **导图位置**：展开或收起气泡造成布局变化时，小人优先保持在原气泡边缘或连接线的相对位置；如果原位置对应的分支已消失，则重新寻找可用路线。
- **像素形象**：在学生创建课程时生成，存在 `course_pixel_avatar`，一门课一行。列表和详情带上五张网格和双语对白，按界面语言显示。生成时要求初始图站立、25% 挥手、50% 拿书、75% 举起双手欢呼、100% 跳跃并带爱心；不校验各档像素差异，实际动作及图片差异取决于模型输出，可能相同或相似。页面按当前学生的 `affection` 选图：小于 25 用初始图，25–49 用 25%，50–74 用 50%，75–99 用 75%，100 用 100%。新要求不改变已有形象。没有形象行时继续用原来的封面。旧形象缺少对白时，服务后台按性格逐个补生成并保存，不改变像素图；失败会记录日志并延迟重试，产生的 AI 用量计入课程所有者。补齐前小人不显示对白，导图打开时会自动刷新。
- **分享状态**：所有者切换分享时调用 `PUT /student/courses/:id/sharing`，写入 `course.is_publish`（0/1）；读列表/详情时按发布状态和 `course_class` 计算可见标签。学生自己的分享课程通过同班可见，不新增 `course_class` 行。
- **学习时长**：课程页可见时上报 heartbeat，后端写入 `student_learning_event`；它记录活动时长，不代替课程进度表。

### 后端已登记、但当前学生 UI 未直接调用的接口

- `PUT /student/courses/:id/content`：保存课程内容。
- `POST /student/courses/:id/chapters/generate`、`POST /student/courses/:id/activities/generate`：生成单章或单 activity。
- `GET /student/courses/:id/quizzes`：按课程列出可见套题。
- 新版 attempt 路由：`GET /quizzes/:quizId`、`GET /quizzes/:quizId/attempt`、`POST /quizzes/:quizId/attempts`、`PUT /attempts/:attemptId/answers/:questionId`、`POST /attempts/:attemptId/submit`。当前学生页面仍使用上表的 `/student/questions/*` 旧答题接口；不要把已登记的新版 attempt API 误认为当前页面正在使用。

## 核心判断规则

### 角色与请求保护

- API 在 `internal/router/routes.go` 注册；全局 `identity.Authenticate` 校验登录 token。学生路由大多标记 `RequireLogin`，再由 service 检查 `ROLE_STUDENT`。
- 业务层会拒绝非学生访问学生课程列表/详情、课程进度、答题、学生课程 AI、学习中心和资源列表等操作。通用工作区对学生和教师开放。
- 仅 UI 隐藏按钮不是权限控制。新增学生功能时，后端也要检查 principal、资源所属用户和课程可见性。

### 课程可见性、来源与所有权

后端 `studentVisibleCourseSQL` 对学生可见课程采用三条规则：

1. 课程所有者本人可见。
2. `is_publish = 1` 且课程没有 `course_class` 指派记录时，与所有者同班的学生可见（学生分享课程）。
3. `is_publish = 1` 且学生属于课程被指派的班级时可见（教师发布/指派课程）。

非所有者看不到未发布课程。接口查询会再次应用这些条件；无权课程详情按 404 处理。返回的 `visibility` 由发布状态和指派班级数区分：`private`、`class`、`assigned`。前端课程来源判断优先看 `owned`（个人），否则 `assigned` 归为教师课程，其余归为共享课程。分享按钮和 AI 课程生成按钮只对 `owned === true` 显示；后端更新分享/生成时也要求 `owner_id` 为当前用户。分享仅切换发布状态，并清理该课程的 `course_class` 记录。

### 课程进度与学习统计

- 课程页把已完成 activity 的 ID 存在 `localStorage`，键为 `caveclass.lesson-done.<userId>.<courseId>`；本地完成比例按课程树叶节点数计算。
- 页面显示/保存比例取服务端进度与本地叶节点比例的较大值。只有点击完成/继续等主操作才会标记 activity 完成并调用 `PUT .../progress`；服务端要求比例在 `0–100`，且课程必须对当前学生可见。
- 首页课程状态：未开始 `percent === 0`，进行中 `0 < percent < 100`，已完成 `percent === 100`。继续学习优先选进行中的课程，其次未完成课程。
- 学习时长由课程页心跳写入事件；后台将课程心跳按时长聚合，进度保存、答题、交卷和章节问答也会写学习事件。
- 学习中心任务由课程进度和套题状态推导：课程未完成时推荐继续；未交卷套题可形成测验任务；已交卷且分数低于 70 分时形成复习任务。点击课程任务进入对应课程详情，点击测验或复习任务进入对应套题详情；点击任务前的完成状态图标不会跳转或更改状态。当前首页统计卡的“待完成测验”只计前端状态为 `pending` 的套题，不包含 `in_progress`；学习任务列表则包含未交卷的进行中套题。

### 套题可见性、答题和交卷

- 套题列表/详情/答题对创建者开放；非创建者必须同时满足课程对其可见、套题有效发布状态为 `published` 且不是 `archived`。旧数据缺 `publish_status` 时从 `is_publish` 推导 draft/published。
- 学生创建套题前端只在本地拼出预览（当前是占位题干和选项，不会请求 AI）；确认后 `POST /student/questions`。服务端校验当前角色、题目输入和目标课程可见性，并将学生创建的套题存为未发布。
- 已答题数按非空 `user_option` 计，不按正确率计；前端状态为 `pending`（0 题已答）、`in_progress`（已答至少一题但未交卷）、`completed`（已交卷）。
- 测验作答页复用测验中心的标题显示规则，隐藏系统生成标题中的时间和内部标识；章节测验优先显示章节名称，答题计时仍单独展示。
- 未提交时详情不返回正确答案和解析；提交后返回每题正确答案、解析和分数。成绩按答对题数计算；提交前的答案只属于当前学生。
- 前端在切题/保存时逐题调用更新接口；交卷时先保存待保存答案，再提交当前题。服务端会校验答案必须是该题选项之一，并将结果写入当前用户自己的进度记录。

### 课程资源与 AI

- 学生资源列表要求当前用户为学生、课程对其可见，并且资源 `publish_status = published`；预览/下载会重复检查课程访问权和资源发布状态。教师的草稿资源不会出现在学生面板。
- 课程 AI 问答要求课程可见、章节 key 存在；服务端仅把目标章节及相关历史传给 AI，并按当前用户保存对话。课程内 AI 对话和通用 AI 工作区是两套上下文/API。
- 课程生成只允许所有者执行；生成内容保存在课程 JSON 并同步课程结构。AI 工作区技能另外按其允许角色和技能所需的课程上下文处理。

## 表

新增列
### `sys_user`
| `major` | 专业，最长 64，可空 |

新增表
### `course`

一门课。

| 列 | 说明 |
|---|---|
| `id` | 主键 |
| `title` | 课程名，最长 200，非空，唯一 |
| `course_type` | 课程类型，最长 32，可空。目前只有 `structured`。没有 `default`，已有空值不回填 |
| `major` | 专业，最长 64，可空。没有 `default`。种子在该列为空时写入 `Computer Science` |
| `is_publish` | `0` 未公开，`1` 已公开。非空，没有 `default` |
| `owner_id` | 所有者，指向 `sys_user.id`，非空 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

### `student_course_progress`

一个学生对一门课的进度。

| 列 | 说明 |
|---|---|
| `user_id` | 学生，指向 `sys_user.id`。和 `course_id` 一起做主键 |
| `course_id` | 课程，指向 `course.id`。和 `user_id` 一起做主键 |
| `percent` | 进度百分比，非空 |
| `affection` | 好感度，0–100，非空，没有 `default`。基础值为课程进度的 80% 加该课习题进度的 20%，结果向下取整；其他行为可以在此基础上加减 |

没有这一行时，列表里的进度为 0，好感度按基础值计算且不插入行。学生创建课程时会为所有者插入一行，进度和好感度都是 0。

升级时按新旧基础值之差一次性调整已有好感度并限制在 0–100，以尽量保留行为加减；此前因上下限截断的部分无法恢复。已解锁成就不会撤销，达到新阈值的成就将在下次同步时解锁。

### `course_pixel_avatar`

一门学生课对应一个像素小人。教师课和种子课没有这一行。

| 列 | 说明 |
|---|---|
| `id` | 主键 |
| `course_id` | 课程，指向 `course.id`，非空，唯一 |
| `role_type` | 角色类型，最长 32，非空 |
| `body_type` | 体型，最长 32，非空 |
| `hair_color` | 发色，最长 32，非空 |
| `hair_style` | 发型，最长 32，非空 |
| `facial_feature` | 五官，最长 32，非空 |
| `accessory` | 配饰，最长 32，非空 |
| `silhouette_key` | 轮廓标识，最长 32，非空 |
| `personality` | 性格，最长 32，非空 |
| `pixel_json` | 初始形象，非空 |
| `pixel_json_25` | 好感度 25% 的表情动作，非空 |
| `pixel_json_50` | 好感度 50% 的表情动作，非空 |
| `pixel_json_75` | 好感度 75% 的表情动作，非空 |
| `pixel_json_100` | 好感度 100% 的表情动作，非空 |
| `created_at` | 创建时间 |

八个外观属性组成唯一索引。两门课不能保存同一套属性。

### `course_question`

一套题。

| 列 | 说明 |
|---|---|
| `id` | 主键 |
| `title` | 套题标题，最长 200，非空，唯一 |
| `course_id` | 所属课程，指向 `course.id`，非空 |
| `creator_id` | 创建者，指向 `sys_user.id`，非空 |
| `question_count` | 题目数量，非空 |
| `is_publish` | `0` 未公开，`1` 已公开。非空，没有 `default`。学生创建的套题写入 `0` |
| `difficulty` | 难度，最长 16，可空。取值 `easy`、`medium`、`hard`。没有 `default`，已有套题不回填。标题仍可带 `· easy · 时间` 以保证唯一 |

### `course_question_content`

一套题里的一道题。

| 列 | 说明 |
|---|---|
| `id` | 主键 |
| `question_id` | 套题，指向 `course_question.id`，非空 |
| `content` | 题干，非空 |
| `options` | 选项，用 `\|` 拼成一段，非空 |
| `correct_answer` | 正确答案，最长 200，非空 |
| `explanation` | 解析，非空 |

### `student_question_progress`

一个学生对一道题的答案。

| 列 | 说明 |
|---|---|
| `user_id` | 学生，指向 `sys_user.id`。和 `content_id` 一起做主键 |
| `content_id` | 题目，指向 `course_question_content.id`。和 `user_id` 一起做主键 |
| `question_id` | 套题，指向 `course_question.id`，非空 |
| `user_option` | 学生答案，最长 200，可空 |

### `student_question_record`

一个学生对一套题是否已提交。

| 列 | 说明 |
|---|---|
| `user_id` | 学生，指向 `sys_user.id`。和 `question_id` 一起做主键 |
| `question_id` | 套题，指向 `course_question.id`。和 `user_id` 一起做主键 |
| `is_submit` | `0` 未提交，`1` 已提交。非空，没有 `default` |

没有这一行表示未提交。

### `student_learning_event`

学生学习活动事件，供学习中心趋势与日历聚合。

| 列 | 说明 |
|---|---|
| `id` | 主键 |
| `user_id` | 学生 |
| `event_type` | `course_session`、`course_progress`、`quiz_answer`、`quiz_submit`、`chapter_qa` |
| `course_id` | 课程，可空 |
| `quiz_id` | 套题，可空 |
| `ref_key` | 附加键，如章节 key，可空 |
| `duration_seconds` | 时长（秒），非负 |
| `occurred_at` | 发生时间（UTC） |

写入来源：课程页心跳、保存进度、答题/提交测验、章节问答。
