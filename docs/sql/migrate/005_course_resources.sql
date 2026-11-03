-- 课程资料元数据（对象在腾讯云 COS，私有 Bucket）
CREATE TABLE IF NOT EXISTS `course_resource` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `course_id` INTEGER NOT NULL,
  `chapter_key` TEXT NOT NULL DEFAULT '',
  `section_key` TEXT NOT NULL DEFAULT '',
  `uploader_id` INTEGER NOT NULL,
  `original_name` TEXT NOT NULL,
  `display_name` TEXT NOT NULL,
  `object_key` TEXT NOT NULL,
  `file_extension` TEXT NOT NULL,
  `mime_type` TEXT NOT NULL DEFAULT '',
  `file_size` INTEGER NOT NULL DEFAULT 0,
  `storage_provider` TEXT NOT NULL DEFAULT 'tencent_cos',
  `scope_type` TEXT NOT NULL DEFAULT 'course',
  `publish_status` TEXT NOT NULL DEFAULT 'draft',
  `preview_status` TEXT NOT NULL DEFAULT 'unsupported',
  `preview_object_key` TEXT NOT NULL DEFAULT '',
  `deleted_at` DATETIME,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS `uidx_course_resource_object_key` ON `course_resource`(`object_key`);
CREATE INDEX IF NOT EXISTS `idx_course_resource_course_pub` ON `course_resource`(`course_id`, `publish_status`, `deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_course_resource_uploader` ON `course_resource`(`uploader_id`);
