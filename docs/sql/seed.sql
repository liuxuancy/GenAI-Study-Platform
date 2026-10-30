-- 本地模拟数据。先执行 init.sql，再执行本文件。
-- 密码：admin / admin123，teacher1–teacher3 / teacher123，student1–student6 / student123。
-- student4 是禁用账号（status = 0）。teacher3 没有分班。
-- 不挂章节的题 chapter_key 为 NULL。同一门课不要插入两条相同的 chapter_key。

BEGIN;

INSERT INTO sys_user (id, username, password_hash, nickname, email, major, role, status, token_version, created_at, updated_at) VALUES
  (1, 'admin', '$2a$10$mn3Udg.1cRzgXc8Ag.1jP./uF26Z.WxaSlc6iEASSzt7OjjZm0DMG', '管理员', 'admin@caveclass.local', NULL, 'ROLE_ADMIN', 1, 0, '2026-08-01 01:00:00', '2026-08-01 01:00:00'),
  (2, 'teacher1', '$2a$10$aReILaQox08xE8hynXY34OdtEtBLen7Z2pF34u35LoLkdy9VGPHia', '讲师一号', 'teacher1@caveclass.local', 'Computer Science', 'ROLE_TEACHER', 1, 0, '2026-08-01 01:10:00', '2026-08-01 01:10:00'),
  (3, 'teacher2', '$2a$10$aReILaQox08xE8hynXY34OdtEtBLen7Z2pF34u35LoLkdy9VGPHia', '讲师二号', 'teacher2@caveclass.local', 'Software Engineering', 'ROLE_TEACHER', 1, 0, '2026-08-01 01:20:00', '2026-08-01 01:20:00'),
  (4, 'teacher3', '$2a$10$aReILaQox08xE8hynXY34OdtEtBLen7Z2pF34u35LoLkdy9VGPHia', '讲师三号', 'teacher3@caveclass.local', 'Data Science', 'ROLE_TEACHER', 1, 0, '2026-08-02 01:00:00', '2026-08-02 01:00:00'),
  (5, 'student1', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', '学生一号', 'student1@caveclass.local', 'Computer Science', 'ROLE_STUDENT', 1, 0, '2026-08-03 01:00:00', '2026-08-03 01:00:00'),
  (6, 'student2', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', '学生二号', 'student2@caveclass.local', 'Computer Science', 'ROLE_STUDENT', 1, 0, '2026-08-03 02:00:00', '2026-08-03 02:00:00'),
  (7, 'student3', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', '学生三号', 'student3@caveclass.local', 'Information Systems', 'ROLE_STUDENT', 1, 0, '2026-08-03 03:00:00', '2026-08-03 03:00:00'),
  (8, 'student4', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', '学生四号', 'student4@caveclass.local', 'Computing', 'ROLE_STUDENT', 0, 1, '2026-08-04 01:00:00', '2026-09-01 01:00:00'),
  (9, 'student5', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', '学生五号', 'student5@caveclass.local', 'Computer Science', 'ROLE_STUDENT', 1, 0, '2026-08-05 01:00:00', '2026-08-05 01:00:00'),
  (10, 'student6', '$2a$10$iQcX0aSapmo7IfwcNxRMA.d8ygEgV2sJ649woR3xKOn5dbw30mBSa', '学生六号', 'student6@caveclass.local', 'Design', 'ROLE_STUDENT', 1, 0, '2026-08-06 01:00:00', '2026-08-06 01:00:00');

INSERT INTO study_class (id, name, created_at, updated_at) VALUES
  (1, '示例班', '2026-08-10 02:00:00', '2026-08-10 02:00:00'),
  (2, '数据结构班', '2026-09-01 02:00:00', '2026-09-01 02:00:00'),
  (3, '晚班', '2026-09-15 02:00:00', '2026-09-15 02:00:00');

INSERT INTO study_class_member (class_id, user_id) VALUES
  (1, 2), (1, 5), (1, 6), (1, 9),
  (2, 2), (2, 3), (2, 6), (2, 7),
  (3, 3), (3, 9), (3, 10);

-- content_json 只放在有章节的课上。学生公开课不写 course_class。
INSERT INTO course (id, title, course_type, major, description, audience, language, content_json, is_publish, owner_id, created_at, updated_at) VALUES
  (1, 'Introduction to Git', 'structured', 'Computer Science', '一次提交保存当时的文件。', 'CS', 'zh-CN',
   '{"schemaVersion":"1.0","title":"Introduction to Git","description":"一次提交保存当时的文件。","audience":"CS","language":"zh-CN","estimatedMinutes":40,"chapters":[{"key":"ch-commit","title":"提交","summary":"提交保存的是那一刻的快照。","order":1,"activities":[{"key":"act-read","type":"rich_text","title":"什么是提交","order":1,"payload":{"markdown":"提交保存的是当时的文件快照，不是一份差异清单。","highlights":["快照"]}},{"key":"act-code","type":"coding","title":"查看状态","order":2,"payload":{"language":"other","promptMarkdown":"在仓库里查看当前状态。","starterCode":"git status"}},{"key":"act-quiz","type":"quiz_game","title":"章末小测","order":3,"payload":{"instructionsMarkdown":"选出正确的一项。","questions":[{"question":"提交保存的是什么？","options":["当时的文件快照","被删掉的文件"],"correctIndex":0,"explanation":"提交保存的是那一刻的快照。"}]}}]},{"key":"ch-branch","title":"分支","summary":"分支是一条可移动的提交线。","order":2,"activities":[{"key":"act-branch","type":"rich_text","title":"为什么要分支","order":1,"payload":{"markdown":"分支让你在不影响主线的情况下试一个改动。"}}]}]}',
   1, 2, '2026-08-12 03:00:00', '2026-10-20 03:00:00'),
  (2, 'How a lecture is read', 'structured', 'Computer Science', '按开场、正文、收束来读一节课。', 'GENERAL', 'zh-CN',
   '{"schemaVersion":"1.0","title":"How a lecture is read","description":"按开场、正文、收束来读一节课。","audience":"GENERAL","language":"zh-CN","estimatedMinutes":25,"chapters":[{"key":"ch-opening","title":"开场","summary":"先说明这节课要解决什么。","order":1,"activities":[{"key":"act-open","type":"rich_text","title":"开场段","order":1,"payload":{"markdown":"开场只回答：这节课结束时你能做什么。"}}]}]}',
   1, 2, '2026-08-18 03:00:00', '2026-10-18 03:00:00'),
  (3, 'Asking on a line', 'structured', 'Computer Science', '卡在某一句时，就在那一句上提问。', 'GENERAL', 'zh-CN', NULL,
   0, 5, '2026-08-20 03:00:00', '2026-08-20 03:00:00'),
  (4, 'Draft lecture', 'structured', 'Computer Science', '还没发给任何班的草稿。', 'CS', 'zh-CN', NULL,
   0, 2, '2026-09-02 03:00:00', '2026-09-02 03:00:00'),
  (5, 'Data Structures and Algorithms', 'structured', 'Computer Science', '数组、链表，以及它们各自适合的访问方式。', 'CS', 'zh-CN',
   '{"schemaVersion":"1.0","title":"Data Structures and Algorithms","description":"数组、链表，以及它们各自适合的访问方式。","audience":"CS","language":"zh-CN","estimatedMinutes":80,"chapters":[{"key":"ch-array","title":"数组","summary":"按下标访问，中间插入要搬动后面的元素。","order":1,"activities":[{"key":"act-array","type":"rich_text","title":"连续存放","order":1,"payload":{"markdown":"数组把元素放在连续内存里，按下标读取很快。"}},{"key":"act-array-code","type":"coding","title":"按下标读取","order":2,"payload":{"language":"python","promptMarkdown":"读取下标 2 的元素。","starterCode":"items = [10, 20, 30]\nprint(items[2])"}}]},{"key":"ch-list","title":"链表","summary":"插入便宜，按下标访问要顺着指针走。","order":2,"activities":[{"key":"act-list","type":"rich_text","title":"指针","order":1,"payload":{"markdown":"链表的下一个元素靠指针找到，不要求内存连续。"}},{"key":"act-interview","type":"mock_interview","title":"面试：数组还是链表","order":2,"payload":{"role":"后端工程师","scenarioMarkdown":"面试官问你日志缓冲该用哪种结构。","questions":[{"question":"频繁在中间插入时你选哪一种？","followUpHints":["说清访问和插入的代价"]}],"evaluationRubricMarkdown":"能对比按下标访问和中间插入。"}}]}]}',
   1, 2, '2026-09-05 03:00:00', '2026-10-22 03:00:00'),
  (6, 'Operating Systems', 'structured', 'Software Engineering', '进程、调度，以及它们和程序的差别。', 'CS', 'zh-CN',
   '{"schemaVersion":"1.0","title":"Operating Systems","description":"进程、调度，以及它们和程序的差别。","audience":"CS","language":"zh-CN","estimatedMinutes":60,"chapters":[{"key":"ch-process","title":"进程","summary":"进程是正在运行的程序，加上它自己的地址空间。","order":1,"activities":[{"key":"act-process","type":"rich_text","title":"程序和进程","order":1,"payload":{"markdown":"程序是磁盘上的文件。进程是它被加载之后正在执行的那一份。"}}]}]}',
   1, 3, '2026-09-12 03:00:00', '2026-10-21 03:00:00'),
  (7, 'Human-Computer Interaction', 'structured', 'Design', '界面先让人完成任务，再谈装饰。', 'NON_CS', 'zh-CN', NULL,
   1, 3, '2026-09-20 03:00:00', '2026-09-20 03:00:00'),
  (8, 'Discrete Mathematics', 'structured', 'Computer Science', '已从学生列表拿下，班级关系还留着。', 'CS', 'zh-CN', NULL,
   0, 2, '2026-09-25 03:00:00', '2026-10-25 03:00:00'),
  (9, 'Shared line notes', 'structured', 'Computer Science', '学生一号对同班公开的笔记。', 'GENERAL', 'zh-CN',
   '{"schemaVersion":"1.0","title":"Shared line notes","description":"学生一号对同班公开的笔记。","audience":"GENERAL","language":"zh-CN","estimatedMinutes":15,"chapters":[{"key":"ch-ask","title":"在原句上问","summary":"问题留在看不懂的那一句旁边。","order":1,"activities":[{"key":"act-ask","type":"rich_text","title":"怎么问","order":1,"payload":{"markdown":"先自己试一步，再把问题写在卡住的那一句上。"}}]}]}',
   1, 5, '2026-10-01 03:00:00', '2026-10-01 03:00:00'),
  (10, 'Sorting notes', 'structured', 'Computer Science', '学生二号对同班公开。示例班和数据结构班都能看见。', 'CS', 'zh-CN', NULL,
   1, 6, '2026-10-05 03:00:00', '2026-10-05 03:00:00'),
  (11, 'Private algorithms', 'structured', 'Information Systems', '学生三号的私有草稿。', 'CS', 'zh-CN', NULL,
   0, 7, '2026-10-08 03:00:00', '2026-10-08 03:00:00'),
  (12, 'Database Systems', 'structured', 'Computer Science', '表、主键，以及为什么查询要走索引。', 'CS', 'en-US',
   '{"schemaVersion":"1.0","title":"Database Systems","description":"Tables, keys, and why a query uses an index.","audience":"CS","language":"en-US","estimatedMinutes":70,"chapters":[{"key":"ch-sql","title":"Tables","summary":"A row is one fact. A column is one kind of fact.","order":1,"activities":[{"key":"act-sql","type":"rich_text","title":"Rows and columns","order":1,"payload":{"markdown":"A primary key identifies one row. Two rows must not share it."}},{"key":"act-sql-code","type":"coding","title":"Select one row","order":2,"payload":{"language":"other","promptMarkdown":"Read the row whose id is 1.","starterCode":"SELECT id, title FROM course WHERE id = 1;"}}]},{"key":"ch-index","title":"Indexes","summary":"An index avoids scanning every row.","order":2,"activities":[{"key":"act-index","type":"rich_text","title":"When to add one","order":1,"payload":{"markdown":"Add an index on a column you filter or join by often."}}]}]}',
   1, 2, '2026-10-12 03:00:00', '2026-10-26 03:00:00'),
  (13, 'Computer Networks', 'structured', 'Data Science', '讲师三号还没进任何班，这门课发不出去。', 'CS', 'zh-CN', NULL,
   0, 4, '2026-10-15 03:00:00', '2026-10-15 03:00:00'),
  (14, 'Evening review', 'structured', 'Design', '学生六号对晚班同学公开。', 'NON_CS', 'zh-CN', NULL,
   1, 10, '2026-10-18 03:00:00', '2026-10-18 03:00:00'),
  (15, 'Disabled draft', 'structured', 'Computing', '禁用账号名下的私有课，重新启用后还在。', 'GENERAL', 'zh-CN', NULL,
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
  (4, 'Git · 提交', 1, 2, 2, 1, 'easy', 'ch-commit'),
  (5, 'Git · 分支', 1, 2, 2, 1, 'medium', 'ch-branch'),
  (6, 'DSA · 数组', 5, 2, 2, 1, 'easy', 'ch-array'),
  (7, 'DSA · 链表', 5, 2, 2, 1, 'medium', 'ch-list'),
  (8, 'OS · 进程', 6, 3, 2, 1, 'easy', 'ch-process'),
  (9, 'DB · Tables', 12, 2, 2, 1, 'easy', 'ch-sql'),
  (10, 'DB · Indexes', 12, 2, 2, 1, 'hard', 'ch-index'),
  (11, 'Shared notes · 提问', 9, 5, 2, 1, 'easy', 'ch-ask'),
  (12, 'HCI checkpoint', 7, 3, 2, 1, 'medium', NULL),
  (13, 'Sorting checkpoint', 10, 6, 2, 1, 'easy', NULL),
  (14, 'Evening checkpoint', 14, 10, 2, 0, 'easy', NULL);

INSERT INTO course_question_content (id, question_id, content, options, correct_answer, explanation) VALUES
  (1, 1, 'What does a commit store?', 'A snapshot of the work|A deleted file', 'A snapshot of the work', 'A commit stores a snapshot of the files at that moment.'),
  (2, 1, 'Who can read a commit after it is made?', 'Anyone with the repository|Only the computer that created it', 'Anyone with the repository', 'A commit stays in the repository history for everyone who can read it.'),
  (3, 2, 'Which paragraph comes first?', 'The opening|The closing', 'The opening', 'A lecture starts with the opening paragraph.'),
  (4, 2, 'What follows the opening?', 'The body|The title', 'The body', 'The body comes after the opening.'),
  (5, 3, 'Where do you ask a question?', 'On the line|In a separate room', 'On the line', 'Ask on the line so the next person can see it.'),
  (6, 3, 'When do you ask?', 'After you have tried|Before reading', 'After you have tried', 'Try the step first, then ask.'),
  (7, 4, '提交保存的是什么？', '当时的文件快照|一份将被删除的清单', '当时的文件快照', '提交记下的是那一刻仓库里的文件。'),
  (8, 4, '改了一个文件之后，旧提交会怎样？', '仍留在历史里|被新提交覆盖掉', '仍留在历史里', '新提交追加在后面，旧提交还在。'),
  (9, 5, '分支通常用来做什么？', '单独试一个改动|删掉主线历史', '单独试一个改动', '分支让试验离开主线。'),
  (10, 5, '两条分支可以指向同一次提交吗？', '可以|不可以', '可以', '分支只是指向提交的名字，多个名字可以指向同一次提交。'),
  (11, 6, '数组按下标读取为什么快？', '元素排在连续内存里|每次都要从头找指针', '元素排在连续内存里', '下标可以直接算出地址。'),
  (12, 6, '在数组中间插入一个元素，通常要做什么？', '把后面的元素往后移|只改一个指针', '把后面的元素往后移', '连续存放时，中间要腾出位置。'),
  (13, 7, '链表中间插入，主要改什么？', '相邻节点的指针|整段内存的下标', '相邻节点的指针', '新节点接上前后指针即可。'),
  (14, 7, '链表按下标访问为什么慢？', '要顺着指针走到那个位置|下标会直接算出地址', '要顺着指针走到那个位置', '第 n 个节点不能跳着读。'),
  (15, 8, '进程和程序的差别是什么？', '进程是正在执行的那一份|两者是同一个文件名', '进程是正在执行的那一份', '程序在磁盘上，进程在运行中。'),
  (16, 8, '两个进程可以同时跑同一个程序吗？', '可以|不可以', '可以', '每个进程有自己的地址空间。'),
  (17, 9, '主键的作用是什么？', '标识唯一的一行|让每一列都可以为空', '标识唯一的一行', '两行不能共用同一个主键。'),
  (18, 9, '一行通常表示什么？', '一件完整的事实|一整张表', '一件完整的事实', '列是事实的各个部分。'),
  (19, 10, '什么时候值得加索引？', '这一列经常被筛选或连接|这一列几乎从不出现在条件里', '这一列经常被筛选或连接', '索引用来少做全表扫描。'),
  (20, 10, '索引一定让写入更快吗？', '不一定，写入时还要维护索引|一定更快', '不一定，写入时还要维护索引', '读得快，写的时候要多更新一份结构。'),
  (21, 11, '问题应该写在哪里？', '看不懂的那一句旁边|另一门课的末尾', '看不懂的那一句旁边', '后来的人能对着原句看答案。'),
  (22, 11, '提问前最好先做什么？', '自己试一步|先把整章标成已完成', '自己试一步', '试过之后，问题会更具体。'),
  (23, 12, '界面首先要服务于什么？', '把任务做完|把装饰做满', '把任务做完', '完不成任务的界面，装饰没有用。'),
  (24, 12, '一个按钮看不懂时，优先改什么？', '按钮上的文字|按钮的阴影', '按钮上的文字', '先让人知道按下之后会发生什么。'),
  (25, 13, '对一小段几乎有序的数据，哪种排序往往更合适？', '插入排序|总是先做全表复制', '插入排序', '已有序的部分不用反复打散。'),
  (26, 13, '比较排序的常见下界是什么量级？', 'n log n|n', 'n log n', '只靠比较时，通常到不了线性。'),
  (27, 14, '晚间复习先看什么？', '当天卡住的那一节|还没上的下一周', '当天卡住的那一节', '先补上今天没懂的，再往后走。'),
  (28, 14, '复习笔记公开给谁？', '同一个班里的同学|全站所有人', '同一个班里的同学', '学生公开课只给同班的人看。');

INSERT INTO student_question_progress (user_id, content_id, question_id, user_option) VALUES
  (5, 1, 1, 'A snapshot of the work'),
  (5, 2, 1, 'Anyone with the repository'),
  (5, 3, 2, 'The opening'),
  (5, 7, 4, '当时的文件快照'),
  (5, 8, 4, '被新提交覆盖掉'),
  (6, 11, 6, '元素排在连续内存里'),
  (6, 12, 6, '把后面的元素往后移'),
  (7, 15, 8, '进程是正在执行的那一份'),
  (9, 17, 9, '标识唯一的一行'),
  (10, 23, 12, '把任务做完');

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

-- 显式写过 id 之后，把自增序号推到当前最大值，避免下次插入撞主键。
UPDATE sqlite_sequence SET seq = 10 WHERE name = 'sys_user';
UPDATE sqlite_sequence SET seq = 3 WHERE name = 'study_class';
UPDATE sqlite_sequence SET seq = 15 WHERE name = 'course';
UPDATE sqlite_sequence SET seq = 14 WHERE name = 'course_question';
UPDATE sqlite_sequence SET seq = 28 WHERE name = 'course_question_content';
UPDATE sqlite_sequence SET seq = 18 WHERE name = 'ai_usage';

COMMIT;
