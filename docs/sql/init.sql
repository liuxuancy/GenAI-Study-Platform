-- CaveClass 本地库结构。SQLite，和 GORM AutoMigrate 当前建出来的表一致。
-- 不建外键。不包含阶段 7 的 course_quality_report、阶段 8 的 study_note。
--
-- 本地重建（在仓库根目录执行，先停掉正在占用这个文件的后端）：
--   rm -f backend/data/caveclass.db backend/data/caveclass.db-wal backend/data/caveclass.db-shm
--   sqlite3 backend/data/caveclass.db < docs/sql/init.sql
--   sqlite3 backend/data/caveclass.db < docs/sql/seed.sql
--
-- 以后改表只改本文件，删掉本地库再跑上面三条。

PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS ai_usage;
DROP TABLE IF EXISTS course_class;
DROP TABLE IF EXISTS course_question_content;
DROP TABLE IF EXISTS course_question;
DROP TABLE IF EXISTS student_course_progress;
DROP TABLE IF EXISTS student_question_progress;
DROP TABLE IF EXISTS student_question_record;
DROP TABLE IF EXISTS study_class_member;
DROP TABLE IF EXISTS study_class;
DROP TABLE IF EXISTS course;
DROP TABLE IF EXISTS sys_role_permission;
DROP TABLE IF EXISTS sys_user;
DROP TABLE IF EXISTS sqlite_sequence;

CREATE TABLE `sys_user` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `username` text NOT NULL,
  `password_hash` text NOT NULL,
  `nickname` text,
  `email` text,
  `major` text,
  `role` text NOT NULL,
  `status` integer NOT NULL,
  `token_version` integer NOT NULL,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `idx_sys_user_username` ON `sys_user`(`username`);
CREATE INDEX `idx_sys_user_role` ON `sys_user`(`role`);

CREATE TABLE `sys_role_permission` (
  `role` text,
  `permission_code` text,
  PRIMARY KEY (`role`, `permission_code`)
);

CREATE INDEX `idx_sys_role_permission_permission_code` ON `sys_role_permission`(`permission_code`);

CREATE TABLE `course` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `title` text NOT NULL,
  `course_type` text,
  `major` text,
  `description` text,
  `audience` text,
  `language` text,
  `content_json` text,
  `is_publish` integer NOT NULL,
  `owner_id` integer NOT NULL,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `idx_course_title` ON `course`(`title`);
CREATE INDEX `idx_course_owner_id` ON `course`(`owner_id`);

CREATE TABLE `student_course_progress` (
  `user_id` integer,
  `course_id` integer,
  `percent` integer NOT NULL,
  PRIMARY KEY (`user_id`, `course_id`)
);

CREATE TABLE `course_question` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `title` text NOT NULL,
  `course_id` integer NOT NULL,
  `creator_id` integer NOT NULL,
  `question_count` integer NOT NULL,
  `is_publish` integer NOT NULL,
  `difficulty` text,
  `chapter_key` text
);

CREATE UNIQUE INDEX `idx_course_question_title` ON `course_question`(`title`);
CREATE INDEX `idx_course_question_course_id` ON `course_question`(`course_id`);
CREATE INDEX `idx_course_question_creator_id` ON `course_question`(`creator_id`);
-- 同一门课、同一个 chapter_key 只能有一套章末题。不挂章节的题把 chapter_key 留空（NULL），不要写成空字符串。
CREATE UNIQUE INDEX `uidx_course_chapter` ON `course_question`(`course_id`, `chapter_key`);

CREATE TABLE `course_question_content` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `question_id` integer NOT NULL,
  `content` text NOT NULL,
  `options` text NOT NULL,
  `correct_answer` text NOT NULL,
  `explanation` text NOT NULL
);

CREATE INDEX `idx_course_question_content_question_id` ON `course_question_content`(`question_id`);

CREATE TABLE `student_question_progress` (
  `user_id` integer,
  `content_id` integer,
  `question_id` integer NOT NULL,
  `user_option` text,
  PRIMARY KEY (`user_id`, `content_id`)
);

CREATE INDEX `idx_student_question_progress_question_id` ON `student_question_progress`(`question_id`);

CREATE TABLE `student_question_record` (
  `user_id` integer,
  `question_id` integer,
  `is_submit` integer NOT NULL,
  PRIMARY KEY (`user_id`, `question_id`)
);

CREATE TABLE `study_class` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `name` text NOT NULL,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `idx_study_class_name` ON `study_class`(`name`);

CREATE TABLE `study_class_member` (
  `class_id` integer,
  `user_id` integer,
  PRIMARY KEY (`class_id`, `user_id`)
);

CREATE INDEX `idx_study_class_member_user_id` ON `study_class_member`(`user_id`);

CREATE TABLE `course_class` (
  `course_id` integer,
  `class_id` integer,
  PRIMARY KEY (`course_id`, `class_id`)
);

CREATE INDEX `idx_course_class_class_id` ON `course_class`(`class_id`);

CREATE TABLE `ai_usage` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `user_id` integer NOT NULL,
  `scene` text NOT NULL,
  `model` text,
  `prompt_tokens` integer NOT NULL,
  `completion_tokens` integer NOT NULL,
  `total_tokens` integer NOT NULL,
  `call_status` text NOT NULL,
  `created_at` datetime
);

CREATE INDEX `idx_ai_usage_user_id` ON `ai_usage`(`user_id`);
CREATE INDEX `idx_ai_usage_created_at` ON `ai_usage`(`created_at`);
