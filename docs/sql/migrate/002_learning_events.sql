-- 已有库手工迁移：学习活动事件表
CREATE TABLE IF NOT EXISTS `student_learning_event` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `user_id` integer NOT NULL,
  `event_type` text NOT NULL,
  `course_id` integer,
  `quiz_id` integer,
  `ref_key` text,
  `duration_seconds` integer NOT NULL,
  `occurred_at` datetime NOT NULL
);

CREATE INDEX IF NOT EXISTS `idx_student_learning_event_user_time` ON `student_learning_event`(`user_id`, `occurred_at`);
