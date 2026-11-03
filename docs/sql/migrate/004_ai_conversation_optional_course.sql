-- 课程关联改为可选。不要手工删表。
-- 服务启动时如果发现 course_id 仍是 NOT NULL，会把现有行复制到新表后再替换，历史不会清空。
-- 新库由 AutoMigrate 直接建成可空的 course_id。
CREATE INDEX IF NOT EXISTS `idx_ai_conv_user_course` ON `student_ai_conversation`(`user_id`, `course_id`, `updated_at`);
