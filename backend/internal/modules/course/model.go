// course 表、课程进度、套题、题目内容、做题进度和套题记录。查询和写入用 SQL，这里只给 AutoMigrate 建表。

package course

import "time"

const (
	PublishOff = 0
	PublishOn  = 1

	SubmitOff = 0
	SubmitOn  = 1

	// CourseTypeStructured 结构化课程。不加 gorm default，已有课程的空类型不回填。
	CourseTypeStructured = "structured"
)

// 库里的 language 为空时，读接口按这个值，不回写。
const DefaultLanguage = "zh-CN"

// Course 对应 course。IsPublish 不加 gorm default 标签，否则写入 0 会被当成未赋值。
// 章节整段放在 ContentJSON，不另建章节表。
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

// StudentCourseProgress 对应 student_course_progress，记录某个学生对某门课的进度。
type StudentCourseProgress struct {
	UserID   int64 `gorm:"primaryKey"`
	CourseID int64 `gorm:"primaryKey"`
	Percent  int   `gorm:"not null"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (StudentCourseProgress) TableName() string { return "student_course_progress" }

// CourseQuestion 对应 course_question，一套题一行。IsPublish 不加 gorm default 标签，否则写入 0 会被当成未赋值。
// 学生创建的套题写入时 IsPublish 为 PublishOff。
// ChapterKey 为空时必须是 NULL，不能是空字符串：同一门课的唯一约束才不会把多套普通题挡在外面。
type CourseQuestion struct {
	ID            int64  `gorm:"primaryKey"`
	Title         string `gorm:"size:200;not null;uniqueIndex"`
	CourseID      int64  `gorm:"not null;index;uniqueIndex:uidx_course_chapter,priority:1"`
	CreatorID     int64  `gorm:"not null;index"`
	QuestionCount int    `gorm:"not null"`
	IsPublish     int    `gorm:"not null"`
	Difficulty    string `gorm:"size:16"` // easy、medium、hard。可空，不加 default。种子只补同标题且为空的值
	ChapterKey    string `gorm:"size:128;uniqueIndex:uidx_course_chapter,priority:2"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (CourseQuestion) TableName() string { return "course_question" }

// CourseQuestionContent 对应 course_question_content，一行是一套题里的一道题。
type CourseQuestionContent struct {
	ID            int64  `gorm:"primaryKey"`
	QuestionID    int64  `gorm:"not null;index"`
	Content       string `gorm:"not null"`
	Options       string `gorm:"not null"`
	CorrectAnswer string `gorm:"size:200;not null"`
	Explanation   string `gorm:"not null"`
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
