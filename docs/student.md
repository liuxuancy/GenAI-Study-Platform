# 学生
表由 AutoMigrate 建出，没有声明外键。

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

没有这一行时，列表里的进度为 0。

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

## 业务规则

学生可以查看自己创建的和公开的课程和题目。

学生端：
    课程列表页：展示课程名、课程类型、进度条 
    题目列表页：展示套题标题、所属课程、套题难度、进度条
    课程详情页：结构气泡图+内容侧+对话框 （点击叶子气泡后出现右侧内容栏）
    题目详情页：题目+答案解析+对话框

待补充：
    课程列表页：创建课程逻辑
    题目列表页：创建题目逻辑
    课程详情页：结构内容生成、展示逻辑，对话框实现
    题目详情页：页面待优化，对话框实现

