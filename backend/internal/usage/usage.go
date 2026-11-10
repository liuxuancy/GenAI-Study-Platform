// Package usage 是 AI 调用用量。course 模块负责把表交给 AutoMigrate。
// system 模块以后只来这里查询。course 与 system 不互相 import，本包也不访问它们。
// internal/ai 不写这张表。
package usage

import "time"

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"

	SceneGenerateCourse        = "generate_course"
	SceneGenerateChapter       = "generate_chapter"
	SceneGenerateActivity      = "generate_activity"
	SceneEvaluateCourse        = "evaluate_course"
	SceneChapterQA             = "chapter_qa"
	SceneStudyNote             = "study_note"
	SceneLearningInsight       = "learning_insight"
	SceneGeneratePixelSprite   = "generate_pixel_sprite"
	SceneGeneratePixelDialogue = "generate_pixel_dialogue"
)

// Usage 对应 ai_usage。一次模型调用一行。不存提示词，不存密钥。
// token 列不加 gorm default，否则写入 0 会被当成未赋值。
type Usage struct {
	ID               int64     `gorm:"primaryKey"`
	UserID           int64     `gorm:"not null;index"`
	Scene            string    `gorm:"size:32;not null"`
	Model            string    `gorm:"size:64"`
	PromptTokens     int       `gorm:"not null"`
	CompletionTokens int       `gorm:"not null"`
	TotalTokens      int       `gorm:"not null"`
	CallStatus       string    `gorm:"size:16;not null"`
	CreatedAt        time.Time `gorm:"index"`
}

// TableName 方法名是 GORM 规定的，不能改成 GetTableName。
func (Usage) TableName() string { return "ai_usage" }
