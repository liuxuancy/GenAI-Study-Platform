-- CaveClass 本地库结构。SQLite，和 GORM AutoMigrate 当前建出来的表一致。
-- 不建外键。不包含阶段 8 的 study_note。
--
-- 本地重建（在仓库根目录执行，先停掉正在占用这个文件的后端）：
--   rm -f backend/data/caveclass.db backend/data/caveclass.db-wal backend/data/caveclass.db-shm
--   sqlite3 backend/data/caveclass.db < docs/sql/init.sql
--   sqlite3 backend/data/caveclass.db < docs/sql/seed.sql
--
-- 以后改表只改本文件，删掉本地库再跑上面三条。

PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS student_ai_message;
DROP TABLE IF EXISTS student_ai_conversation;
DROP TABLE IF EXISTS migration_log;
DROP TABLE IF EXISTS quiz_answer;
DROP TABLE IF EXISTS quiz_attempt;
DROP TABLE IF EXISTS ai_usage;
DROP TABLE IF EXISTS course_resource;
DROP TABLE IF EXISTS course_quality_report;
DROP TABLE IF EXISTS course_class;
DROP TABLE IF EXISTS course_section;
DROP TABLE IF EXISTS course_chapter;
DROP TABLE IF EXISTS course_question_content;
DROP TABLE IF EXISTS course_question;
DROP TABLE IF EXISTS student_learning_event;
DROP TABLE IF EXISTS course_pixel_avatar;
DROP TABLE IF EXISTS student_course_progress;
DROP TABLE IF EXISTS student_achievement;
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

CREATE TABLE `course_chapter` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `course_id` integer NOT NULL,
  `chapter_key` text NOT NULL,
  `title` text NOT NULL,
  `description` text,
  `sort_order` integer NOT NULL,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `uidx_course_chapter_key` ON `course_chapter`(`course_id`, `chapter_key`);
CREATE INDEX `idx_course_chapter_course_id` ON `course_chapter`(`course_id`);

CREATE TABLE `course_section` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `course_id` integer NOT NULL,
  `chapter_id` integer NOT NULL,
  `section_key` text NOT NULL,
  `title` text NOT NULL,
  `description` text,
  `sort_order` integer NOT NULL,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `uidx_course_section_key` ON `course_section`(`chapter_id`, `section_key`);
CREATE INDEX `idx_course_section_course_id` ON `course_section`(`course_id`);

CREATE TABLE `student_course_progress` (
  `user_id` integer,
  `course_id` integer,
  `percent` integer NOT NULL,
  `affection` integer NOT NULL,
  PRIMARY KEY (`user_id`, `course_id`)
);

CREATE TABLE `student_achievement` (
  `user_id` integer,
  `achievement` text,
  `unlocked_at` datetime NOT NULL,
  `notified_at` datetime,
  PRIMARY KEY (`user_id`, `achievement`)
);

CREATE TABLE `course_pixel_avatar` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `course_id` integer NOT NULL,
  `role_type` text NOT NULL,
  `body_type` text NOT NULL,
  `hair_color` text NOT NULL,
  `hair_style` text NOT NULL,
  `facial_feature` text NOT NULL,
  `accessory` text NOT NULL,
  `silhouette_key` text NOT NULL,
  `personality` text NOT NULL,
  `pixel_json` text NOT NULL,
  `pixel_json_25` text NOT NULL,
  `pixel_json_50` text NOT NULL,
  `pixel_json_75` text NOT NULL,
  `pixel_json_100` text NOT NULL,
  `dialogue_json` text,
  `dialogue_lease` text NOT NULL DEFAULT '',
  `dialogue_retry_at` integer NOT NULL DEFAULT 0,
  `dialogue_attempts` integer NOT NULL DEFAULT 0,
  `created_at` datetime
);

CREATE UNIQUE INDEX `uidx_course_pixel_avatar_course` ON `course_pixel_avatar`(`course_id`);
CREATE UNIQUE INDEX `uidx_pixel_avatar_appearance` ON `course_pixel_avatar`(
  `role_type`, `body_type`, `hair_color`, `hair_style`, `facial_feature`, `accessory`, `silhouette_key`, `personality`
);

CREATE TABLE `student_learning_event` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `user_id` integer NOT NULL,
  `event_type` text NOT NULL,
  `course_id` integer,
  `quiz_id` integer,
  `ref_key` text,
  `duration_seconds` integer NOT NULL,
  `occurred_at` datetime NOT NULL
);

CREATE INDEX `idx_student_learning_event_user_time` ON `student_learning_event`(`user_id`, `occurred_at`);

CREATE TABLE `student_ai_conversation` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `user_id` integer NOT NULL,
  `course_id` integer,
  `title` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE INDEX `idx_ai_conv_user_course` ON `student_ai_conversation`(`user_id`, `course_id`, `updated_at`);

CREATE TABLE `student_ai_message` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `conversation_id` integer NOT NULL,
  `role` text NOT NULL,
  `content` text NOT NULL,
  `chapter_key` text,
  `activity_key` text,
  `section_title` text,
  `created_at` datetime NOT NULL
);

CREATE INDEX `idx_ai_message_conversation` ON `student_ai_message`(`conversation_id`, `id`);

CREATE TABLE `course_question` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `title` text NOT NULL,
  `course_id` integer NOT NULL,
  `creator_id` integer NOT NULL,
  `question_count` integer NOT NULL,
  `is_publish` integer NOT NULL,
  `difficulty` text,
  `chapter_key` text,
  `section_key` text,
  `chapter_id` integer,
  `section_id` integer,
  `description` text,
  `creation_method` text,
  `publish_status` text,
  `published_at` datetime,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `idx_course_question_title` ON `course_question`(`title`);
CREATE INDEX `idx_course_question_course_id` ON `course_question`(`course_id`);
CREATE INDEX `idx_course_question_creator_id` ON `course_question`(`creator_id`);
-- 同一门课、同一个章节只能有一套未挂小节的章末题。测验小节各自一套，用 section_key 区分。
CREATE UNIQUE INDEX `uidx_course_chapter_only` ON `course_question`(`course_id`, `chapter_key`) WHERE `chapter_key` IS NOT NULL AND `chapter_key` <> '' AND (`section_key` IS NULL OR `section_key` = '');
CREATE UNIQUE INDEX `uidx_course_section_quiz` ON `course_question`(`course_id`, `chapter_key`, `section_key`) WHERE `section_key` IS NOT NULL AND `section_key` <> '';

CREATE TABLE `course_question_content` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `question_id` integer NOT NULL,
  `content` text NOT NULL,
  `options` text NOT NULL,
  `correct_answer` text NOT NULL,
  `explanation` text NOT NULL,
  `question_type` text NOT NULL,
  `sort_order` integer NOT NULL,
  `score` integer NOT NULL,
  `created_at` datetime,
  `updated_at` datetime
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

CREATE TABLE `quiz_attempt` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `quiz_id` integer NOT NULL,
  `student_id` integer NOT NULL,
  `attempt_no` integer NOT NULL,
  `status` text NOT NULL,
  `total_questions` integer NOT NULL,
  `answered_questions` integer NOT NULL,
  `score` integer,
  `started_at` datetime,
  `submitted_at` datetime,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `uidx_quiz_attempt_user` ON `quiz_attempt`(`quiz_id`, `student_id`, `attempt_no`);
CREATE INDEX `idx_quiz_attempt_student` ON `quiz_attempt`(`student_id`);

CREATE TABLE `quiz_answer` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `attempt_id` integer NOT NULL,
  `question_id` integer NOT NULL,
  `answer` text NOT NULL,
  `is_correct` integer,
  `score` integer,
  `answered_at` datetime,
  `updated_at` datetime
);

CREATE UNIQUE INDEX `uidx_quiz_answer_attempt_question` ON `quiz_answer`(`attempt_id`, `question_id`);

CREATE TABLE `migration_log` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `kind` text NOT NULL,
  `ref_key` text NOT NULL,
  `detail` text NOT NULL,
  `created_at` datetime
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

CREATE TABLE `course_quality_report` (
  `course_id` integer PRIMARY KEY,
  `overall_score` integer NOT NULL,
  `report_json` text NOT NULL,
  `created_at` datetime
);

CREATE TABLE `course_resource` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `course_id` integer NOT NULL,
  `chapter_key` text NOT NULL DEFAULT '',
  `section_key` text NOT NULL DEFAULT '',
  `uploader_id` integer NOT NULL,
  `original_name` text NOT NULL,
  `display_name` text NOT NULL,
  `object_key` text NOT NULL,
  `file_extension` text NOT NULL,
  `mime_type` text NOT NULL DEFAULT '',
  `file_size` integer NOT NULL DEFAULT 0,
  `storage_provider` text NOT NULL DEFAULT 'tencent_cos',
  `scope_type` text NOT NULL DEFAULT 'course',
  `publish_status` text NOT NULL DEFAULT 'draft',
  `preview_status` text NOT NULL DEFAULT 'unsupported',
  `preview_object_key` text NOT NULL DEFAULT '',
  `deleted_at` datetime,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL
);

CREATE UNIQUE INDEX `uidx_course_resource_object_key` ON `course_resource`(`object_key`);
CREATE INDEX `idx_course_resource_course_pub` ON `course_resource`(`course_id`, `publish_status`, `deleted_at`);
CREATE INDEX `idx_course_resource_uploader` ON `course_resource`(`uploader_id`);

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
