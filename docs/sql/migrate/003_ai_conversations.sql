-- 已有库手工迁移：学生课程 AI 会话。新启动的服务也会由 AutoMigrate 建这两张表。
CREATE TABLE IF NOT EXISTS `student_ai_conversation` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `user_id` integer NOT NULL,
  `course_id` integer NOT NULL,
  `title` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE INDEX IF NOT EXISTS `idx_ai_conv_user_course` ON `student_ai_conversation`(`user_id`, `course_id`, `updated_at`);

CREATE TABLE IF NOT EXISTS `student_ai_message` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `conversation_id` integer NOT NULL,
  `role` text NOT NULL,
  `content` text NOT NULL,
  `chapter_key` text,
  `activity_key` text,
  `section_title` text,
  `created_at` datetime NOT NULL
);

CREATE INDEX IF NOT EXISTS `idx_ai_message_conversation` ON `student_ai_message`(`conversation_id`, `id`);
