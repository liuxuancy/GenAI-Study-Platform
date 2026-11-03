-- 增量迁移（已有 caveclass.db 时手动执行；新库请直接用 init.sql）。
-- 执行前备份：cp backend/data/caveclass.db backend/data/caveclass.db.bak

PRAGMA foreign_keys = OFF;

CREATE TABLE IF NOT EXISTS course_chapter (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  course_id INTEGER NOT NULL,
  chapter_key TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME,
  updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_course_chapter_key ON course_chapter(course_id, chapter_key);
CREATE INDEX IF NOT EXISTS idx_course_chapter_course_id ON course_chapter(course_id);

CREATE TABLE IF NOT EXISTS course_section (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  course_id INTEGER NOT NULL,
  chapter_id INTEGER NOT NULL,
  section_key TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME,
  updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_course_section_key ON course_section(chapter_id, section_key);
CREATE INDEX IF NOT EXISTS idx_course_section_course_id ON course_section(course_id);

CREATE TABLE IF NOT EXISTS quiz_attempt (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  quiz_id INTEGER NOT NULL,
  student_id INTEGER NOT NULL,
  attempt_no INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL,
  total_questions INTEGER NOT NULL,
  answered_questions INTEGER NOT NULL DEFAULT 0,
  score INTEGER,
  started_at DATETIME,
  submitted_at DATETIME,
  created_at DATETIME,
  updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_quiz_attempt_user ON quiz_attempt(quiz_id, student_id, attempt_no);
CREATE INDEX IF NOT EXISTS idx_quiz_attempt_student ON quiz_attempt(student_id);

CREATE TABLE IF NOT EXISTS quiz_answer (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  attempt_id INTEGER NOT NULL,
  question_id INTEGER NOT NULL,
  answer TEXT NOT NULL,
  is_correct INTEGER,
  score INTEGER,
  answered_at DATETIME,
  updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_quiz_answer_attempt_question ON quiz_answer(attempt_id, question_id);

CREATE TABLE IF NOT EXISTS migration_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL,
  ref_key TEXT NOT NULL,
  detail TEXT NOT NULL,
  created_at DATETIME
);

-- SQLite 3.35+：若列已存在会失败，可忽略或逐条执行
-- ALTER TABLE course_question ADD COLUMN description TEXT;
-- ...
