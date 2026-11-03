-- Demo seed data. Run init.sql first, then this file.
-- Passwords: admin / admin123, teacher1–teacher3 / teacher123, student1–student6 / student123.
-- Courses: metadata only (title, description, …). No content_json chapters or pre-built quizzes.
-- student4 is disabled (status = 0). teacher3 is not assigned to any class.

BEGIN;

INSERT INTO sys_user (id, username, password_hash, nickname, email, major, role, status, token_version, created_at, updated_at) VALUES
  (1, 'admin', '$2a$10$mn3Udg.1cRzgXc8Ag.1jP./uF26Z.WxaSlc6iEASSzt7OjjZm0DMG', 'Administrator', 'admin@caveclass.local', NULL, 'ROLE_ADMIN', 1, 0, '2026-08-01 01:00:00', '2026-08-01 01:00:00'),
  (2, 'teacher1', '$2a$10$aReILaQox08xE8hynXY34OdtEtBLen7Z2pF34u35LoLkdy9VGPHia', 'Teacher One', 'teacher1@caveclass.local', 'Computer Science', 'ROLE_TEACHER', 1, 0, '2026-08-01 01:10:00', '2026-08-01 01:10:00'),
  (3, 'teacher2', '$2a$10$aReILaQox08xE8hynXY34OdtEtBLen7Z2pF34u35LoLkdy9VGPHia', 'Teacher Two', 'teacher2@caveclass.local', 'Software Engineering', 'ROLE_TEACHER', 1, 0, '2026-08-01 01:20:00', '2026-08-01 01:20:00'),
  (4, 'teacher3', '$2a$10$aReILaQox08xE8hynXY34OdtEtBLen7Z2pF34u35LoLkdy9VGPHia', 'Teacher Three', 'teacher3@caveclass.local', 'Data Science', 'ROLE_TEACHER', 1, 0, '2026-08-02 01:00:00', '2026-08-02 01:00:00'),
  (5, 'student1', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', 'Student One', 'student1@caveclass.local', 'Computer Science', 'ROLE_STUDENT', 1, 0, '2026-08-03 01:00:00', '2026-08-03 01:00:00'),
  (6, 'student2', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', 'Student Two', 'student2@caveclass.local', 'Computer Science', 'ROLE_STUDENT', 1, 0, '2026-08-03 02:00:00', '2026-08-03 02:00:00'),
  (7, 'student3', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', 'Student Three', 'student3@caveclass.local', 'Information Systems', 'ROLE_STUDENT', 1, 0, '2026-08-03 03:00:00', '2026-08-03 03:00:00'),
  (8, 'student4', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', 'Student Four', 'student4@caveclass.local', 'Computing', 'ROLE_STUDENT', 0, 1, '2026-08-04 01:00:00', '2026-09-01 01:00:00'),
  (9, 'student5', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', 'Student Five', 'student5@caveclass.local', 'Computer Science', 'ROLE_STUDENT', 1, 0, '2026-08-05 01:00:00', '2026-08-05 01:00:00'),
  (10, 'student6', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', 'Student Six', 'student6@caveclass.local', 'Design', 'ROLE_STUDENT', 1, 0, '2026-08-06 01:00:00', '2026-08-06 01:00:00');

INSERT INTO study_class (id, name, created_at, updated_at) VALUES
  (1, 'Demo Class', '2026-08-10 02:00:00', '2026-08-10 02:00:00'),
  (2, 'Data Structures Class', '2026-09-01 02:00:00', '2026-09-01 02:00:00'),
  (3, 'Evening Class', '2026-09-15 02:00:00', '2026-09-15 02:00:00');

INSERT INTO study_class_member (class_id, user_id) VALUES
  (1, 2), (1, 5), (1, 6), (1, 9),
  (2, 2), (2, 3), (2, 6), (2, 7),
  (3, 3), (3, 9), (3, 10);

-- Courses ship metadata only (no content_json chapters). Chapters and quizzes are created via AI after deploy.
INSERT INTO course (id, title, course_type, major, description, audience, language, content_json, is_publish, owner_id, created_at, updated_at) VALUES
  (1, 'Introduction to Git', 'structured', 'Computer Science', 'A commit saves the files as they were at that moment.', 'CS', 'en-US', NULL,
   1, 2, '2026-08-12 03:00:00', '2026-10-20 03:00:00'),
  (2, 'How a lecture is read', 'structured', 'Computer Science', 'Read a lecture as opening, body, and closing.', 'GENERAL', 'en-US', NULL,
   1, 2, '2026-08-18 03:00:00', '2026-10-18 03:00:00'),
  (3, 'Asking on a line', 'structured', 'Computer Science', 'When you are stuck on one sentence, ask on that sentence.', 'GENERAL', 'en-US', NULL,
   0, 5, '2026-08-20 03:00:00', '2026-08-20 03:00:00'),
  (4, 'Draft lecture', 'structured', 'Computer Science', 'Draft not assigned to any class yet.', 'CS', 'en-US', NULL,
   0, 2, '2026-09-02 03:00:00', '2026-09-02 03:00:00'),
  (5, 'Data Structures and Algorithms', 'structured', 'Computer Science', 'Arrays, linked lists, and when each access pattern fits.', 'CS', 'en-US', NULL,
   1, 2, '2026-09-05 03:00:00', '2026-10-22 03:00:00'),
  (6, 'Operating Systems', 'structured', 'Computer Science', 'Processes, scheduling, and how they differ from programs.', 'CS', 'en-US', NULL,
   1, 3, '2026-09-12 03:00:00', '2026-10-21 03:00:00'),
  (7, 'Human-Computer Interaction', 'structured', 'Design', 'Help people finish tasks before adding decoration.', 'NON_CS', 'en-US', NULL,
   1, 3, '2026-09-20 03:00:00', '2026-09-20 03:00:00'),
  (8, 'Discrete Mathematics', 'structured', 'Computer Science', 'Unpublished for students; class link remains.', 'CS', 'en-US', NULL,
   0, 2, '2026-09-25 03:00:00', '2026-10-25 03:00:00'),
  (9, 'Shared line notes', 'structured', 'Computer Science', 'Public notes from Student One to classmates.', 'GENERAL', 'en-US', NULL,
   1, 5, '2026-10-01 03:00:00', '2026-10-01 03:00:00'),
  (10, 'Sorting notes', 'structured', 'Computer Science', 'Public from Student Two; visible in Demo Class and Data Structures Class.', 'CS', 'en-US', NULL,
   1, 6, '2026-10-05 03:00:00', '2026-10-05 03:00:00'),
  (11, 'Private algorithms', 'structured', 'Information Systems', 'Private draft owned by Student Three.', 'CS', 'en-US', NULL,
   0, 7, '2026-10-08 03:00:00', '2026-10-08 03:00:00'),
  (12, 'Database Systems', 'structured', 'Computer Science', 'Tables, keys, and why a query uses an index.', 'CS', 'en-US', NULL,
   1, 2, '2026-10-12 03:00:00', '2026-10-26 03:00:00'),
  (13, 'Computer Networks', 'structured', 'Data Science', 'Teacher Three is in no class, so this course cannot be assigned.', 'CS', 'en-US', NULL,
   0, 4, '2026-10-15 03:00:00', '2026-10-15 03:00:00'),
  (14, 'Evening review', 'structured', 'Design', 'Public to Evening Class from Student Six.', 'NON_CS', 'en-US', NULL,
   1, 10, '2026-10-18 03:00:00', '2026-10-18 03:00:00'),
  (15, 'Disabled draft', 'structured', 'Computing', 'Private course under a disabled account; still there after re-enable.', 'GENERAL', 'en-US', NULL,
   0, 8, '2026-10-28 03:00:00', '2026-10-28 03:00:00');

INSERT INTO course_class (course_id, class_id) VALUES
  (1, 1),
  (2, 1),
  (5, 1), (5, 2),
  (6, 2), (6, 3),
  (7, 3),
  (8, 1),
  (12, 1);

-- No pre-filled learning progress; COALESCE(..., 0) in API until students learn.

INSERT INTO ai_usage (id, user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at) VALUES
  (1, 2, 'generate_course', 'deepseek-chat', 820, 1460, 2280, 'success', '2026-10-20 03:10:00'),
  (2, 2, 'generate_activity', 'deepseek-chat', 210, 180, 390, 'success', '2026-10-20 03:12:00'),
  (3, 2, 'generate_activity', 'deepseek-chat', 0, 0, 0, 'failed', '2026-10-21 01:00:00'),
  (4, 2, 'generate_course', 'teamily', 640, 980, 1620, 'success', '2026-10-22 04:00:00'),
  (5, 3, 'generate_course', 'deepseek-chat', 700, 1100, 1800, 'success', '2026-10-21 06:00:00'),
  (6, 3, 'generate_activity', 'deepseek-chat', 180, 240, 420, 'success', '2026-10-21 06:05:00'),
  (7, 3, 'generate_course', 'mock', 0, 0, 0, 'failed', '2026-10-23 02:00:00'),
  (8, 5, 'generate_course', 'deepseek-chat', 300, 520, 820, 'success', '2026-10-01 04:00:00'),
  (9, 5, 'generate_activity', 'deepseek-chat', 90, 110, 200, 'success', '2026-10-01 04:03:00'),
  (10, 6, 'generate_course', 'teamily', 280, 0, 280, 'failed', '2026-10-05 05:00:00'),
  (11, 6, 'generate_course', 'teamily', 260, 410, 670, 'success', '2026-10-05 05:20:00'),
  (12, 7, 'generate_activity', 'mock', 40, 20, 60, 'success', '2026-10-08 02:00:00'),
  (13, 9, 'generate_course', 'deepseek-chat', 150, 260, 410, 'success', '2026-10-16 08:00:00'),
  (14, 10, 'generate_course', 'deepseek-chat', 120, 200, 320, 'success', '2026-10-18 09:00:00'),
  (15, 10, 'generate_activity', 'deepseek-chat', 0, 0, 0, 'failed', '2026-10-18 09:02:00'),
  (16, 4, 'generate_course', 'mock', 50, 10, 60, 'failed', '2026-10-15 07:00:00'),
  (17, 8, 'generate_course', 'mock', 30, 0, 30, 'failed', '2026-10-28 04:00:00'),
  (18, 1, 'generate_course', 'deepseek-chat', 10, 5, 15, 'success', '2026-10-19 01:00:00');

-- After explicit ids, bump AUTOINCREMENT so the next insert does not collide.
UPDATE sqlite_sequence SET seq = 10 WHERE name = 'sys_user';
UPDATE sqlite_sequence SET seq = 3 WHERE name = 'study_class';
UPDATE sqlite_sequence SET seq = 15 WHERE name = 'course';
UPDATE sqlite_sequence SET seq = 18 WHERE name = 'ai_usage';

COMMIT;
