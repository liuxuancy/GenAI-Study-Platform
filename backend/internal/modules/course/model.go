// course 表、课程进度、套题、题目内容、做题进度和套题记录。查询和写入用 SQL，这里只给 AutoMigrate 建表。

package course

import (
	"time"

	"gorm.io/gorm"
)

const (
	PublishOff = 0
	PublishOn  = 1

	SubmitOff = 0
	SubmitOn  = 1

	// CourseTypeStructured 结构化课程。不加 gorm default，已有课程的空类型不回填。
	CourseTypeStructured = "structured"

	PublishStatusDraft     = "draft"
	PublishStatusPublished = "published"
	PublishStatusArchived  = "archived"

	CreationMethodManual = "manual"
	CreationMethodExcel  = "excel"
	CreationMethodAI     = "ai"

	QuestionTypeSingleChoice = "single_choice"

	AttemptStatusInProgress = "in_progress"
	AttemptStatusSubmitted  = "submitted"

	EventCourseSession  = "course_session"
	EventCourseProgress = "course_progress"
	EventQuizAnswer     = "quiz_answer"
	EventQuizSubmit     = "quiz_submit"
	EventChapterQA      = "chapter_qa"
	EventStorySession   = "story_session"
)

// 库里的 language 为空时，读接口按这个值，不回写。
const DefaultLanguage = "zh-CN"

// Course 对应 course。IsPublish 不加 gorm default 标签，否则写入 0 会被当成未赋值。
// 章节正文仍在 ContentJSON；course_chapter / course_section 由 JSON 派生同步。
type Course struct {
	ID          int64  `gorm:"primaryKey"`
	Title       string `gorm:"size:200;not null;uniqueIndex"`
	CourseType  string `gorm:"size:32"` // 课程类型，目前只有 structured
	Major       string `gorm:"size:64"` // 专业，可空，不加 default
	Description string `gorm:"type:text"`
	Audience    string `gorm:"size:16"` // CS、NON_CS、GENERAL，可空
	Language    string `gorm:"size:16"` // zh-CN、en-US，可空
	ContentJSON string `gorm:"type:text"`
	IsPublish   int    `gorm:"not null"` // 学生课：1 同班可见。教师课：1 还要有 course_class 才对学生可见
	OwnerID     int64  `gorm:"not null;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (Course) TableName() string { return "course" }

// CourseChapter 由 content_json 同步，不作为独立编辑源。
type CourseChapter struct {
	ID          int64  `gorm:"primaryKey"`
	CourseID    int64  `gorm:"not null;index;uniqueIndex:uidx_course_chapter_key,priority:1"`
	ChapterKey  string `gorm:"size:128;not null;uniqueIndex:uidx_course_chapter_key,priority:2"`
	Title       string `gorm:"size:200;not null"`
	Description string `gorm:"type:text"`
	SortOrder   int    `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (CourseChapter) TableName() string { return "course_chapter" }

// CourseSection 由 content_json 同步。
type CourseSection struct {
	ID          int64  `gorm:"primaryKey"`
	CourseID    int64  `gorm:"not null;index"`
	ChapterID   int64  `gorm:"not null;uniqueIndex:uidx_course_section_key,priority:1"`
	SectionKey  string `gorm:"size:128;not null;uniqueIndex:uidx_course_section_key,priority:2"`
	Title       string `gorm:"size:200;not null"`
	Description string `gorm:"type:text"`
	SortOrder   int    `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (CourseSection) TableName() string { return "course_section" }

// StudentCourseProgress 对应 student_course_progress，记录某个学生对某门课的进度。
type StudentCourseProgress struct {
	UserID   int64 `gorm:"primaryKey"`
	CourseID int64 `gorm:"primaryKey"`
	Percent  int   `gorm:"not null"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (StudentCourseProgress) TableName() string { return "student_course_progress" }

// StudentLearningEvent 记录学生学习活动，供趋势与日历聚合。
type StudentLearningEvent struct {
	ID              int64  `gorm:"primaryKey"`
	UserID          int64  `gorm:"not null;index:idx_student_learning_event_user_time,priority:1"`
	EventType       string `gorm:"size:32;not null"`
	CourseID        *int64
	QuizID          *int64
	RefKey          string    `gorm:"size:128"`
	DurationSeconds int       `gorm:"not null"`
	OccurredAt      time.Time `gorm:"not null;index:idx_student_learning_event_user_time,priority:2"`
}

func (StudentLearningEvent) TableName() string { return "student_learning_event" }

// CourseQuestion 对应 course_question，业务上的 Quiz。IsPublish 保留兼容；新逻辑用 PublishStatus。
type CourseQuestion struct {
	ID             int64  `gorm:"primaryKey"`
	Title          string `gorm:"size:200;not null;uniqueIndex"`
	CourseID       int64  `gorm:"not null;index"`
	CreatorID      int64  `gorm:"not null;index"`
	QuestionCount  int    `gorm:"not null"`
	IsPublish      int    `gorm:"not null"`
	Difficulty     string `gorm:"size:16"`
	ChapterKey     string `gorm:"size:128;index"`
	SectionKey     string `gorm:"size:128;index"`
	ChapterID      *int64 `gorm:"index"`
	SectionID      *int64 `gorm:"index"`
	Description    string `gorm:"type:text"`
	CreationMethod string `gorm:"size:16"`
	PublishStatus  string `gorm:"size:16"`
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (CourseQuestion) TableName() string { return "course_question" }

// CourseQuestionContent 对应 course_question_content，业务上的 Question。
type CourseQuestionContent struct {
	ID            int64  `gorm:"primaryKey"`
	QuestionID    int64  `gorm:"not null;index"`
	Content       string `gorm:"not null"`
	Options       string `gorm:"not null"`
	CorrectAnswer string `gorm:"size:200;not null"`
	Explanation   string `gorm:"not null"`
	QuestionType  string `gorm:"size:32;not null;default:single_choice"`
	SortOrder     int    `gorm:"not null;default:0"`
	Score         int    `gorm:"not null;default:1"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (CourseQuestionContent) TableName() string { return "course_question_content" }

// StudentQuestionProgress 对应 student_question_progress，一行是一个学生对一道题的答案。
type StudentQuestionProgress struct {
	UserID     int64  `gorm:"primaryKey"`
	ContentID  int64  `gorm:"primaryKey"`
	QuestionID int64  `gorm:"not null;index"`
	UserOption string `gorm:"size:200"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (StudentQuestionProgress) TableName() string { return "student_question_progress" }

// StudentQuestionRecord 对应 student_question_record，一行是一个学生对一套题是否已提交。
// IsSubmit 不加 gorm default 标签，否则写入 0 会被当成未赋值。没有行表示未提交。
type StudentQuestionRecord struct {
	UserID     int64 `gorm:"primaryKey"`
	QuestionID int64 `gorm:"primaryKey"`
	IsSubmit   int   `gorm:"not null"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (StudentQuestionRecord) TableName() string { return "student_question_record" }

// QuizAttempt 一次测验作答。
type QuizAttempt struct {
	ID                int64  `gorm:"primaryKey"`
	QuizID            int64  `gorm:"not null;uniqueIndex:uidx_quiz_attempt_user,priority:1"`
	StudentID         int64  `gorm:"not null;index;uniqueIndex:uidx_quiz_attempt_user,priority:2"`
	AttemptNo         int    `gorm:"not null;uniqueIndex:uidx_quiz_attempt_user,priority:3"`
	Status            string `gorm:"size:16;not null"`
	TotalQuestions    int    `gorm:"not null"`
	AnsweredQuestions int    `gorm:"not null"`
	Score             *int
	StartedAt         *time.Time
	SubmittedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (QuizAttempt) TableName() string { return "quiz_attempt" }

// QuizAnswer 一次 Attempt 内的一道题答案。
type QuizAnswer struct {
	ID         int64  `gorm:"primaryKey"`
	AttemptID  int64  `gorm:"not null;uniqueIndex:uidx_quiz_answer_attempt_question,priority:1"`
	QuestionID int64  `gorm:"not null;uniqueIndex:uidx_quiz_answer_attempt_question,priority:2"`
	Answer     string `gorm:"type:text;not null"`
	IsCorrect  *int
	Score      *int
	AnsweredAt time.Time
	UpdatedAt  time.Time
}

func (QuizAnswer) TableName() string { return "quiz_answer" }

// MigrationLog 数据迁移差异与告警。
type MigrationLog struct {
	ID        int64  `gorm:"primaryKey"`
	Kind      string `gorm:"size:64;not null"`
	RefKey    string `gorm:"size:256;not null"`
	Detail    string `gorm:"type:text;not null"`
	CreatedAt time.Time
}

func (MigrationLog) TableName() string { return "migration_log" }

// StudentAIConversation 是某个学生对一门课的一次 AI 会话。
type StudentAIConversation struct {
	ID        int64  `gorm:"primaryKey"`
	UserID    int64  `gorm:"not null;index:idx_ai_conv_user_course,priority:1"`
	CourseID  *int64 `gorm:"index:idx_ai_conv_user_course,priority:2"`
	Title     string `gorm:"size:80;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time `gorm:"index:idx_ai_conv_user_course,priority:3"`
}

func (StudentAIConversation) TableName() string { return "student_ai_conversation" }

// StudentAIMessage 是会话中的一条消息，并记下产生时的章节和小节。
type StudentAIMessage struct {
	ID             int64     `gorm:"primaryKey"`
	ConversationID int64     `gorm:"not null;index:idx_ai_message_conversation,priority:1"`
	Role           string    `gorm:"size:16;not null"`
	Content        string    `gorm:"type:text;not null"`
	ChapterKey     string    `gorm:"size:128"`
	ActivityKey    string    `gorm:"size:128"`
	SectionTitle   string    `gorm:"size:200"`
	CreatedAt      time.Time `gorm:"index:idx_ai_message_conversation,priority:2"`
}

func (StudentAIMessage) TableName() string { return "student_ai_message" }

// StudyClass 对应 study_class。表名避开 SQL 的 CLASS。
type StudyClass struct {
	ID        int64  `gorm:"primaryKey"`
	Name      string `gorm:"size:64;not null;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (StudyClass) TableName() string { return "study_class" }

// StudyClassMember 对应 study_class_member。主键是班级加用户。班里不另存角色。
type StudyClassMember struct {
	ClassID int64 `gorm:"primaryKey"`
	UserID  int64 `gorm:"primaryKey;index"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (StudyClassMember) TableName() string { return "study_class_member" }

// CourseClass 对应 course_class。教师把课发到哪些班。学生自己的课不写这张表。
type CourseClass struct {
	CourseID int64 `gorm:"primaryKey"`
	ClassID  int64 `gorm:"primaryKey;index"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (CourseClass) TableName() string { return "course_class" }

// CourseQualityReport 保存每门课最新一次 AI 质量评估。
type CourseQualityReport struct {
	CourseID     int64  `gorm:"primaryKey"`
	OverallScore int    `gorm:"not null"`
	ReportJSON   string `gorm:"type:text;not null"`
	CreatedAt    time.Time
}

func (CourseQualityReport) TableName() string { return "course_quality_report" }

const (
	ResourceScopeCourse  = "course"
	ResourceScopeChapter = "chapter"
	ResourceScopeSection = "section"

	ResourceStorageTencentCOS = "tencent_cos"

	ResourcePreviewPending      = "pending"
	ResourcePreviewProcessing = "processing"
	ResourcePreviewReady        = "ready"
	ResourcePreviewFailed       = "failed"
	ResourcePreviewUnsupported  = "unsupported"
)

// CourseResource 课程资料元数据；文件字节存 COS，Key 见 platform/cos/keys.go。
type CourseResource struct {
	ID               int64  `gorm:"primaryKey"`
	CourseID         int64  `gorm:"not null;index"`
	ChapterKey       string `gorm:"size:128;not null"`
	SectionKey       string `gorm:"size:128;not null"`
	UploaderID       int64  `gorm:"not null;index"`
	OriginalName     string `gorm:"size:512;not null"`
	DisplayName      string `gorm:"size:512;not null"`
	ObjectKey        string `gorm:"size:512;not null;uniqueIndex"`
	FileExtension    string `gorm:"size:16;not null"`
	MimeType         string `gorm:"size:128;not null"`
	FileSize         int64  `gorm:"not null"`
	StorageProvider  string `gorm:"size:32;not null"`
	ScopeType        string `gorm:"size:16;not null"`
	PublishStatus    string `gorm:"size:16;not null"`
	PreviewStatus    string `gorm:"size:16;not null"`
	PreviewObjectKey string `gorm:"size:512;not null"`
	DeletedAt        gorm.DeletedAt
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (CourseResource) TableName() string { return "course_resource" }
