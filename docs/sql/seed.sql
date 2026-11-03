-- Demo seed data. Run init.sql first, then this file.
-- Passwords: admin / admin123, teacher1–teacher3 / teacher123, student1–student6 / student123.
-- student4 is disabled (status = 0). teacher3 is not assigned to any class.
-- Questions without a chapter use chapter_key NULL. Do not insert duplicate chapter_key for the same course.

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

-- content_json only on courses with chapters. Student public courses have no course_class rows.
INSERT INTO course (id, title, course_type, major, description, audience, language, content_json, is_publish, owner_id, created_at, updated_at) VALUES
  (1, 'Introduction to Git', 'structured', 'Computer Science', 'A commit saves the files as they were at that moment.', 'CS', 'en-US',
   '{"schemaVersion":"1.0","title":"Introduction to Git","description":"A commit saves the files as they were at that moment.","audience":"CS","language":"en-US","estimatedMinutes":40,"chapters":[{"key":"ch-commit","title":"Commits","summary":"A commit stores a snapshot of the repository at one moment.","order":1,"activities":[{"key":"act-read","type":"rich_text","title":"What is a commit","order":1,"payload":{"markdown":"A commit stores a snapshot of the files at that moment, not a list of diffs.","highlights":["snapshot"]}},{"key":"act-code","type":"coding","title":"Check status","order":2,"payload":{"language":"other","promptMarkdown":"Check the current state of the repository.","starterCode":"git status"}},{"key":"act-quiz","type":"quiz_game","title":"Chapter quiz","order":3,"payload":{"instructionsMarkdown":"Pick the correct answer.","questions":[{"question":"What does a commit store?","options":["A snapshot of the files at that moment","A deleted file"],"correctIndex":0,"explanation":"A commit stores a snapshot at that moment."}]}}]},{"key":"ch-branch","title":"Branches","summary":"A branch is a movable line of commits.","order":2,"activities":[{"key":"act-branch","type":"rich_text","title":"Why branch","order":1,"payload":{"markdown":"A branch lets you try a change without disturbing the main line."}}]}]}',
   1, 2, '2026-08-12 03:00:00', '2026-10-20 03:00:00'),
  (2, 'How a lecture is read', 'structured', 'Computer Science', 'Read a lecture as opening, body, and closing.', 'GENERAL', 'en-US',
   '{"schemaVersion":"1.0","title":"How a lecture is read","description":"Read a lecture as opening, body, and closing.","audience":"GENERAL","language":"en-US","estimatedMinutes":25,"chapters":[{"key":"ch-opening","title":"Opening","summary":"State what this session will solve.","order":1,"activities":[{"key":"act-open","type":"rich_text","title":"Opening section","order":1,"payload":{"markdown":"The opening answers one question: what will you be able to do when this session ends?"}}]}]}',
   1, 2, '2026-08-18 03:00:00', '2026-10-18 03:00:00'),
  (3, 'Asking on a line', 'structured', 'Computer Science', 'When you are stuck on one sentence, ask on that sentence.', 'GENERAL', 'en-US', NULL,
   0, 5, '2026-08-20 03:00:00', '2026-08-20 03:00:00'),
  (4, 'Draft lecture', 'structured', 'Computer Science', 'Draft not assigned to any class yet.', 'CS', 'en-US', NULL,
   0, 2, '2026-09-02 03:00:00', '2026-09-02 03:00:00'),
  (5, 'Data Structures and Algorithms', 'structured', 'Computer Science', 'Arrays, linked lists, and when each access pattern fits.', 'CS', 'en-US',
   '{"schemaVersion":"1.0","title":"Data Structures and Algorithms","description":"Arrays, linked lists, and when each access pattern fits.","audience":"CS","language":"en-US","estimatedMinutes":80,"chapters":[{"key":"ch-array","title":"Arrays","summary":"Index access is fast; inserting in the middle shifts what follows.","order":1,"activities":[{"key":"act-array","type":"rich_text","title":"Contiguous storage","order":1,"payload":{"markdown":"An array keeps elements in contiguous memory, so indexed reads are fast."}},{"key":"act-array-code","type":"coding","title":"Read by index","order":2,"payload":{"language":"python","promptMarkdown":"Read the element at index 2.","starterCode":"items = [10, 20, 30]\nprint(items[2])"}}]},{"key":"ch-list","title":"Linked lists","summary":"Insertion is cheap; indexed access walks pointers.","order":2,"activities":[{"key":"act-list","type":"rich_text","title":"Pointers","order":1,"payload":{"markdown":"The next element is found through a pointer; memory need not be contiguous."}},{"key":"act-interview","type":"mock_interview","title":"Interview: array or list","order":2,"payload":{"role":"Backend engineer","scenarioMarkdown":"The interviewer asks which structure you would use for a log buffer.","questions":[{"question":"Which do you pick when inserts happen often in the middle?","followUpHints":["Compare access and insertion cost"]}],"evaluationRubricMarkdown":"Can contrast indexed access with middle insertion."}}]}]}',
   1, 2, '2026-09-05 03:00:00', '2026-10-22 03:00:00'),
  (6, 'Operating Systems', 'structured', 'Software Engineering', 'Processes, scheduling, and how they differ from programs.', 'CS', 'en-US',
   '{"schemaVersion":"1.0","title":"Operating Systems","description":"Processes, scheduling, and how they differ from programs.","audience":"CS","language":"en-US","estimatedMinutes":60,"chapters":[{"key":"ch-process","title":"Processes","summary":"A process is a running program plus its own address space.","order":1,"activities":[{"key":"act-process","type":"rich_text","title":"Program vs process","order":1,"payload":{"markdown":"A program is a file on disk. A process is that program loaded and executing."}}]}]}',
   1, 3, '2026-09-12 03:00:00', '2026-10-21 03:00:00'),
  (7, 'Human-Computer Interaction', 'structured', 'Design', 'Help people finish tasks before adding decoration.', 'NON_CS', 'en-US', NULL,
   1, 3, '2026-09-20 03:00:00', '2026-09-20 03:00:00'),
  (8, 'Discrete Mathematics', 'structured', 'Computer Science', 'Unpublished for students; class link remains.', 'CS', 'en-US', NULL,
   0, 2, '2026-09-25 03:00:00', '2026-10-25 03:00:00'),
  (9, 'Shared line notes', 'structured', 'Computer Science', 'Public notes from Student One to classmates.', 'GENERAL', 'en-US',
   '{"schemaVersion":"1.0","title":"Shared line notes","description":"Public notes from Student One to classmates.","audience":"GENERAL","language":"en-US","estimatedMinutes":15,"chapters":[{"key":"ch-ask","title":"Ask on the line","summary":"Keep the question beside the sentence you do not understand.","order":1,"activities":[{"key":"act-ask","type":"rich_text","title":"How to ask","order":1,"payload":{"markdown":"Try one step yourself, then write the question on the line where you got stuck."}}]}]}',
   1, 5, '2026-10-01 03:00:00', '2026-10-01 03:00:00'),
  (10, 'Sorting notes', 'structured', 'Computer Science', 'Public from Student Two; visible in Demo Class and Data Structures Class.', 'CS', 'en-US', NULL,
   1, 6, '2026-10-05 03:00:00', '2026-10-05 03:00:00'),
  (11, 'Private algorithms', 'structured', 'Information Systems', 'Private draft owned by Student Three.', 'CS', 'en-US', NULL,
   0, 7, '2026-10-08 03:00:00', '2026-10-08 03:00:00'),
  (12, 'Database Systems', 'structured', 'Computer Science', 'Tables, keys, and why a query uses an index.', 'CS', 'en-US',
   '{"schemaVersion":"1.0","title":"Database Systems","description":"Tables, keys, and why a query uses an index.","audience":"CS","language":"en-US","estimatedMinutes":70,"chapters":[{"key":"ch-sql","title":"Tables","summary":"A row is one fact. A column is one kind of fact.","order":1,"activities":[{"key":"act-sql","type":"rich_text","title":"Rows and columns","order":1,"payload":{"markdown":"A primary key identifies one row. Two rows must not share it."}},{"key":"act-sql-code","type":"coding","title":"Select one row","order":2,"payload":{"language":"other","promptMarkdown":"Read the row whose id is 1.","starterCode":"SELECT id, title FROM course WHERE id = 1;"}}]},{"key":"ch-index","title":"Indexes","summary":"An index avoids scanning every row.","order":2,"activities":[{"key":"act-index","type":"rich_text","title":"When to add one","order":1,"payload":{"markdown":"Add an index on a column you filter or join by often."}}]}]}',
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

INSERT INTO course_question (id, title, course_id, creator_id, question_count, is_publish, difficulty, chapter_key) VALUES
  (1, 'What is a commit?', 1, 2, 2, 1, 'easy', NULL),
  (2, 'Which paragraph comes first?', 2, 2, 2, 1, 'medium', NULL),
  (3, 'Where do you ask?', 3, 5, 2, 0, 'easy', NULL),
  (4, 'Git · Commits', 1, 2, 2, 1, 'easy', 'ch-commit'),
  (5, 'Git · Branches', 1, 2, 2, 1, 'medium', 'ch-branch'),
  (6, 'DSA · Arrays', 5, 2, 2, 1, 'easy', 'ch-array'),
  (7, 'DSA · Linked lists', 5, 2, 2, 1, 'medium', 'ch-list'),
  (8, 'OS · Processes', 6, 3, 2, 1, 'easy', 'ch-process'),
  (9, 'DB · Tables', 12, 2, 2, 1, 'easy', 'ch-sql'),
  (10, 'DB · Indexes', 12, 2, 2, 1, 'hard', 'ch-index'),
  (11, 'Shared notes · Asking', 9, 5, 2, 1, 'easy', 'ch-ask'),
  (12, 'HCI checkpoint', 7, 3, 2, 1, 'medium', NULL),
  (13, 'Sorting checkpoint', 10, 6, 2, 1, 'easy', NULL),
  (14, 'Evening checkpoint', 14, 10, 2, 0, 'easy', NULL);

INSERT INTO course_question_content (id, question_id, content, options, correct_answer, explanation, question_type, sort_order, score) VALUES
  (1, 1, 'What does a commit store?', 'A snapshot of the work|A deleted file', 'A snapshot of the work', 'A commit stores a snapshot of the files at that moment.', 'single_choice', 1, 1),
  (2, 1, 'Who can read a commit after it is made?', 'Anyone with the repository|Only the computer that created it', 'Anyone with the repository', 'A commit stays in the repository history for everyone who can read it.', 'single_choice', 2, 1),
  (3, 2, 'Which paragraph comes first?', 'The opening|The closing', 'The opening', 'A lecture starts with the opening paragraph.', 'single_choice', 3, 1),
  (4, 2, 'What follows the opening?', 'The body|The title', 'The body', 'The body comes after the opening.', 'single_choice', 4, 1),
  (5, 3, 'Where do you ask a question?', 'On the line|In a separate room', 'On the line', 'Ask on the line so the next person can see it.', 'single_choice', 5, 1),
  (6, 3, 'When do you ask?', 'After you have tried|Before reading', 'After you have tried', 'Try the step first, then ask.', 'single_choice', 6, 1),
  (7, 4, 'What does a commit store?', 'A snapshot of the files at that moment|A list of files to delete', 'A snapshot of the files at that moment', 'A commit records the files in the repository at that moment.', 'single_choice', 7, 1),
  (8, 4, 'After you change a file, what happens to older commits?', 'They stay in history|They are replaced by the new commit', 'They stay in history', 'New commits are appended; older commits remain.', 'single_choice', 8, 1),
  (9, 5, 'What are branches typically used for?', 'Trying a change in isolation|Deleting mainline history', 'Trying a change in isolation', 'Branches keep experiments off the main line.', 'single_choice', 9, 1),
  (10, 5, 'Can two branches point to the same commit?', 'Yes|No', 'Yes', 'A branch is just a name for a commit; several names can point to one commit.', 'single_choice', 10, 1),
  (11, 6, 'Why is indexed access fast in an array?', 'Elements sit in contiguous memory|You must walk pointers from the head each time', 'Elements sit in contiguous memory', 'An index maps directly to an address.', 'single_choice', 11, 1),
  (12, 6, 'When inserting in the middle of an array, what usually happens?', 'Elements after the gap are shifted|Only one pointer is updated', 'Elements after the gap are shifted', 'Contiguous storage needs room in the middle.', 'single_choice', 12, 1),
  (13, 7, 'When inserting in the middle of a linked list, what changes?', 'Pointers of neighboring nodes|Indices across the whole block', 'Pointers of neighboring nodes', 'Wire the new node between its neighbors.', 'single_choice', 13, 1),
  (14, 7, 'Why is indexed access slow in a linked list?', 'You walk pointers to reach position n|An index computes the address directly', 'You walk pointers to reach position n', 'You cannot jump to the nth node without walking.', 'single_choice', 14, 1),
  (15, 8, 'How does a process differ from a program?', 'A process is the running instance|They are always the same file name', 'A process is the running instance', 'The program is on disk; the process is executing.', 'single_choice', 15, 1),
  (16, 8, 'Can two processes run the same program at once?', 'Yes|No', 'Yes', 'Each process has its own address space.', 'single_choice', 16, 1),
  (17, 9, 'What is a primary key for?', 'Identifying a unique row|Allowing every column to be null', 'Identifying a unique row', 'Two rows cannot share the same primary key.', 'single_choice', 17, 1),
  (18, 9, 'What does one row usually represent?', 'One complete fact|The whole table', 'One complete fact', 'Columns are parts of that fact.', 'single_choice', 18, 1),
  (19, 10, 'When is an index worth adding?', 'The column is often filtered or joined|The column never appears in conditions', 'The column is often filtered or joined', 'Indexes reduce full table scans.', 'single_choice', 19, 1),
  (20, 10, 'Does an index always make writes faster?', 'Not always; writes must maintain the index|Always faster', 'Not always; writes must maintain the index', 'Reads can be faster; writes update extra structure.', 'single_choice', 20, 1),
  (21, 11, 'Where should you write the question?', 'Beside the sentence you do not understand|At the end of another course', 'Beside the sentence you do not understand', 'Others can see the answer in context.', 'single_choice', 21, 1),
  (22, 11, 'What should you do before asking?', 'Try one step yourself|Mark the whole chapter complete', 'Try one step yourself', 'After trying, your question is more specific.', 'single_choice', 22, 1),
  (23, 12, 'What should the interface serve first?', 'Completing the task|Decorating the screen', 'Completing the task', 'Decoration does not help if the task fails.', 'single_choice', 23, 1),
  (24, 12, 'If a button is unclear, what do you fix first?', 'The label on the button|The shadow on the button', 'The label on the button', 'People need to know what happens when they click.', 'single_choice', 24, 1),
  (25, 13, 'For a small nearly sorted segment, which sort is often a good fit?', 'Insertion sort|Always copy the whole table first', 'Insertion sort', 'Already ordered parts are not reshuffled repeatedly.', 'single_choice', 25, 1),
  (26, 13, 'What is a common lower bound for comparison sorts?', 'n log n|n', 'n log n', 'With comparisons only, linear time is usually unreachable.', 'single_choice', 26, 1),
  (27, 14, 'What should evening review start with?', 'The section that blocked you today|Next week''s material', 'The section that blocked you today', 'Fix today''s gaps before moving on.', 'single_choice', 27, 1),
  (28, 14, 'Who can see public review notes?', 'Classmates in the same class|Everyone on the site', 'Classmates in the same class', 'Student public courses are visible to classmates only.', 'single_choice', 28, 1);

INSERT INTO student_question_progress (user_id, content_id, question_id, user_option) VALUES
  (5, 1, 1, 'A snapshot of the work'),
  (5, 2, 1, 'Anyone with the repository'),
  (5, 3, 2, 'The opening'),
  (5, 7, 4, 'A snapshot of the files at that moment'),
  (5, 8, 4, 'They are replaced by the new commit'),
  (6, 11, 6, 'Elements sit in contiguous memory'),
  (6, 12, 6, 'Elements after the gap are shifted'),
  (7, 15, 8, 'A process is the running instance'),
  (9, 17, 9, 'Identifying a unique row'),
  (10, 23, 12, 'Completing the task');

INSERT INTO student_question_record (user_id, question_id, is_submit) VALUES
  (5, 1, 1),
  (5, 4, 1),
  (6, 6, 1),
  (7, 8, 0),
  (9, 9, 1),
  (10, 12, 0);

INSERT INTO student_course_progress (user_id, course_id, percent) VALUES
  (5, 1, 35),
  (5, 2, 70),
  (5, 3, 10),
  (5, 9, 100),
  (5, 12, 20),
  (6, 1, 80),
  (6, 5, 45),
  (6, 10, 15),
  (7, 6, 55),
  (7, 10, 30),
  (7, 11, 5),
  (9, 1, 15),
  (9, 6, 40),
  (9, 12, 60),
  (9, 14, 25),
  (10, 6, 10),
  (10, 7, 50),
  (10, 14, 90),
  (8, 15, 0);

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
UPDATE sqlite_sequence SET seq = 14 WHERE name = 'course_question';
UPDATE sqlite_sequence SET seq = 28 WHERE name = 'course_question_content';
UPDATE sqlite_sequence SET seq = 18 WHERE name = 'ai_usage';

COMMIT;
