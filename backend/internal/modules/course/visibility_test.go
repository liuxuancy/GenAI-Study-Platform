package course

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

const sampleContent = `{
  "schemaVersion": "1.0",
  "title": "Stored title",
  "description": "Stored description",
  "audience": "GENERAL",
  "language": "zh-CN",
  "estimatedMinutes": 10,
  "chapters": [
    {
      "key": "ch-1",
      "title": "Opening",
      "summary": "Start here",
      "order": 1,
      "activities": [
        {
          "key": "act-1",
          "type": "rich_text",
          "title": "Read this",
          "order": 1,
          "payload": { "markdown": "Hello chapter" }
        },
        {
          "key": "act-2",
          "type": "coding",
          "title": "Try a command",
          "order": 2,
          "payload": { "language": "other", "promptMarkdown": "Type git status" }
        }
      ]
    }
  ]
}`

func TestSeedDemoClass(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed again: %v", err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM study_class", 1)
	assertCount(t, database, "SELECT COUNT(*) FROM study_class_member", 2)
	assertCount(t, database, "SELECT COUNT(*) FROM course_class", 2)
	assertCount(t, database, "SELECT COUNT(*) FROM course_class WHERE course_id = (SELECT id FROM course WHERE title = 'Asking on a line')", 0)
	assertCount(t, database, "SELECT COUNT(*) FROM course_class WHERE course_id = (SELECT id FROM course WHERE title = 'Draft lecture')", 0)
	if !database.Migrator().HasTable("ai_usage") {
		t.Fatal("ai_usage was not created")
	}
	if !database.Migrator().HasColumn(&Course{}, "ContentJSON") || !database.Migrator().HasColumn(&CourseQuestion{}, "ChapterKey") {
		t.Fatal("new course columns were not created")
	}

	gitID := courseIDByTitle(t, database, "Introduction to Git")
	testutil.MustExec(t, database, "UPDATE course SET is_publish = ? WHERE id = ?", PublishOff, gitID)
	testutil.MustExec(t, database, "DELETE FROM course_class WHERE course_id = ?", gitID)
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed after unpublish: %v", err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE id = ? AND is_publish = 0", 1, gitID)
	assertCount(t, database, "SELECT COUNT(*) FROM course_class WHERE course_id = ?", 0, gitID)
}

func TestClassVisibilityAndContent(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	teacher := teacherPrincipal(t, database, "teacher1")
	student2 := insertPrincipal(t, database, "student2", access.RoleStudent)
	outsider := student2

	outsiderCourses, err := service.GetCourseList(context.Background(), outsider)
	if err != nil {
		t.Fatalf("outsider list: %v", err)
	}
	if len(outsiderCourses) != 0 {
		t.Fatalf("outsider sees %v", outsiderCourses)
	}

	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{
		Title: "  Line notes  ", Description: "only me",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Visibility != VisibilityPrivate || created.Title != "Line notes" || created.Description != "only me" {
		t.Fatalf("created = %+v", created)
	}
	_, err = service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Introduction to Git"})
	if !errors.Is(err, errs.ErrConflict) {
		t.Fatalf("duplicate title: %v", err)
	}
	_, err = service.CreateStudentCourse(context.Background(), teacher, CreateCourseInput{Title: "Nope"})
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher create as student: %v", err)
	}

	shared, err := service.SetCourseSharing(context.Background(), student, created.ID, true)
	if err != nil {
		t.Fatalf("share: %v", err)
	}
	if !shared.Shared || shared.ClassmateCount != 0 {
		t.Fatalf("share result = %+v, want no other students", shared)
	}
	stillHidden, err := service.GetCourse(context.Background(), outsider, created.ID)
	if !errors.Is(err, errs.ErrNotFound) || stillHidden != nil {
		t.Fatalf("outsider after share: %v %+v", err, stillHidden)
	}

	classID := classIDByName(t, database, seedClassName)
	testutil.MustExec(t, database, "INSERT INTO study_class_member (class_id, user_id) VALUES (?, ?)", classID, outsider.UserID)
	visible, err := service.GetCourse(context.Background(), outsider, created.ID)
	if err != nil {
		t.Fatalf("classmate course: %v", err)
	}
	if visible.Visibility != VisibilityClass || !visible.Shared || visible.Tree != nil {
		t.Fatalf("shared course = %+v", visible)
	}
	if _, err := service.SetCourseSharing(context.Background(), student, created.ID, false); err != nil {
		t.Fatalf("unshare: %v", err)
	}
	if _, err := service.GetCourse(context.Background(), outsider, created.ID); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("after unshare: %v", err)
	}

	draft, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Lab notes", Description: "for one class"})
	if err != nil {
		t.Fatalf("teacher create: %v", err)
	}
	if draft.Published || len(draft.ClassIDs) != 0 || draft.Tree != nil {
		t.Fatalf("draft = %+v", draft)
	}
	if _, err := service.GetCourse(context.Background(), student, draft.ID); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("draft visible: %v", err)
	}
	if _, err := service.PublishCourseToClasses(context.Background(), teacherPrincipal(t, database, "teacher2"), draft.ID, []int64{classID}); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("other teacher publish: %v", err)
	}
	secondClass, err := service.CreateClass(context.Background(), "另一班")
	if err != nil {
		t.Fatalf("second class: %v", err)
	}
	if _, err := service.PublishCourseToClasses(context.Background(), teacher, draft.ID, []int64{secondClass.ID}); !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("publish outside own class: %v", err)
	}

	published, err := service.PublishCourseToClasses(context.Background(), teacher, draft.ID, []int64{classID, classID})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !published.Published || len(published.ClassIDs) != 1 || published.ClassIDs[0] != classID {
		t.Fatalf("published = %+v", published)
	}
	if _, err := service.GetCourse(context.Background(), student, draft.ID); err != nil {
		t.Fatalf("class student: %v", err)
	}
	progress, err := service.GetTeacherCourseProgress(context.Background(), teacher, draft.ID)
	if err != nil {
		t.Fatalf("progress: %v", err)
	}
	if len(progress.Students) != 2 {
		t.Fatalf("progress = %+v", progress.Students)
	}
	if _, err := service.PublishCourseToClasses(context.Background(), teacher, draft.ID, []int64{}); err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if _, err := service.GetCourse(context.Background(), student, draft.ID); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("after withdraw: %v", err)
	}
	withdrawn, err := service.GetTeacherCourseProgress(context.Background(), teacher, draft.ID)
	if err != nil {
		t.Fatalf("withdrawn progress: %v", err)
	}
	if len(withdrawn.Students) != 0 {
		t.Fatalf("withdrawn students = %+v", withdrawn.Students)
	}

	if err := service.SaveCourseContent(context.Background(), student, created.ID, "{"); !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("bad content: %v", err)
	}
	if err := service.SaveCourseContent(context.Background(), student, created.ID, sampleContent); err != nil {
		t.Fatalf("save content: %v", err)
	}
	detail, err := service.GetCourse(context.Background(), student, created.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Tree == nil || detail.Tree.Label != "Line notes" || len(detail.Tree.Children) != 1 {
		t.Fatalf("tree = %+v", detail.Tree)
	}
	chapter := detail.Tree.Children[0]
	if chapter.Key != "ch-1" || chapter.Label != "Opening" || len(chapter.Children) != 2 || chapter.Children[0].Note != "Hello chapter" {
		t.Fatalf("chapter = %+v", chapter)
	}
	if chapter.Children[1].ActivityType != "coding" || chapter.Children[1].Note != "Type git status" {
		t.Fatalf("coding = %+v", chapter.Children[1])
	}
	testutil.MustExec(t, database, "UPDATE course_question SET chapter_key = ? WHERE title = ?", "ch-1", "Where do you ask?")
	// 章末题挂在另一门课上时，这门课的气泡不应带上它。
	if detail, err = service.GetCourse(context.Background(), student, created.ID); err != nil {
		t.Fatal(err)
	}
	if detail.Tree.Children[0].QuestionID != 0 {
		t.Fatalf("question id = %d", detail.Tree.Children[0].QuestionID)
	}
	testutil.MustExec(t, database, "UPDATE course_question SET course_id = ?, chapter_key = ? WHERE title = ?", created.ID, "ch-1", "Where do you ask?")
	detail, err = service.GetCourse(context.Background(), student, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Tree.Children[0].QuestionID == 0 {
		t.Fatal("chapter question was not attached")
	}
	err = database.Exec(
		"UPDATE course_question SET chapter_key = ? WHERE title = ?",
		"ch-1", "What is a commit?",
	).Error
	if err != nil {
		t.Fatal(err)
	}
	moved := database.Exec(
		"UPDATE course_question SET course_id = ? WHERE title = ?",
		created.ID, "What is a commit?",
	)
	if moved.Error == nil {
		t.Fatal("duplicate chapter key was accepted")
	}
}

func TestClassMembership(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	if err := (Module{}).CheckUserDeletion(context.Background(), database, studentID); !errors.Is(err, errs.ErrConflict) {
		t.Fatalf("delete member: %v", err)
	}

	created, err := service.CreateClass(context.Background(), "  晚班  ")
	if err != nil {
		t.Fatalf("create class: %v", err)
	}
	if created.Name != "晚班" || len(created.Members) != 0 {
		t.Fatalf("class = %+v", created)
	}
	_, err = service.CreateClass(context.Background(), "晚班")
	if !errors.Is(err, errs.ErrConflict) {
		t.Fatalf("duplicate class: %v", err)
	}
	_, err = service.CreateClass(context.Background(), strings.Repeat("班", maxClassNameRunes+1))
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("long name: %v", err)
	}
	adminID := testutil.GetUserID(t, database, "admin")
	_, err = service.SetClassMembers(context.Background(), created.ID, []int64{adminID})
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("admin member: %v", err)
	}
	_, err = service.SetClassMembers(context.Background(), created.ID, []int64{99999})
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}
	teacherID := testutil.GetUserID(t, database, "teacher1")
	updated, err := service.SetClassMembers(context.Background(), created.ID, []int64{teacherID, studentID, teacherID})
	if err != nil {
		t.Fatalf("set members: %v", err)
	}
	if len(updated.Members) != 2 {
		t.Fatalf("members = %+v", updated.Members)
	}
	cleared, err := service.SetClassMembers(context.Background(), created.ID, []int64{})
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if len(cleared.Members) != 0 {
		t.Fatalf("cleared = %+v", cleared.Members)
	}

	classes, err := service.GetTeacherClasses(context.Background(), teacherPrincipal(t, database, "teacher1"))
	if err != nil {
		t.Fatalf("teacher classes: %v", err)
	}
	if len(classes) != 1 || classes[0].Name != seedClassName {
		t.Fatalf("teacher classes = %+v", classes)
	}
	gitID := courseIDByTitle(t, database, "Introduction to Git")
	progress, err := service.GetTeacherCourseProgress(context.Background(), teacherPrincipal(t, database, "teacher1"), gitID)
	if err != nil {
		t.Fatalf("seed progress: %v", err)
	}
	if len(progress.Students) != 1 || progress.Students[0].Username != "student1" || progress.Students[0].Percent != 0 {
		t.Fatalf("seed progress = %+v", progress.Students)
	}
}

func TestUsageColumnsAcceptZero(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	userID := testutil.GetUserID(t, database, "student1")
	now := time.Now().UTC()
	testutil.MustExec(t, database,
		`INSERT INTO ai_usage (user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at)
		 VALUES (?, ?, '', 0, 0, 0, ?, ?)`,
		userID, usage.SceneGenerateCourse, usage.StatusFailed, now,
	)
	var total int
	if err := database.Raw("SELECT total_tokens FROM ai_usage WHERE user_id = ?", userID).Scan(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("total tokens = %d", total)
	}
}

func studentPrincipal(t *testing.T, database *gorm.DB, username string) *access.Principal {
	t.Helper()
	return &access.Principal{UserID: testutil.GetUserID(t, database, username), Role: access.RoleStudent}
}

func teacherPrincipal(t *testing.T, database *gorm.DB, username string) *access.Principal {
	t.Helper()
	return &access.Principal{UserID: testutil.GetUserID(t, database, username), Role: access.RoleTeacher}
}

func insertPrincipal(t *testing.T, database *gorm.DB, username string, role string) *access.Principal {
	t.Helper()
	now := time.Now().UTC()
	testutil.MustExec(t, database,
		`INSERT INTO sys_user (username, password_hash, nickname, email, role, status, token_version, created_at, updated_at)
		 VALUES (?, 'x', ?, '', ?, 1, 0, ?, ?)`,
		username, username, role, now, now,
	)
	return &access.Principal{UserID: testutil.GetUserID(t, database, username), Role: role}
}

func classIDByName(t *testing.T, database *gorm.DB, name string) int64 {
	t.Helper()
	var classID int64
	result := database.Raw("SELECT id FROM study_class WHERE name = ?", name).Scan(&classID)
	if result.Error != nil || result.RowsAffected == 0 || classID == 0 {
		t.Fatalf("class %q: %v", name, result.Error)
	}
	return classID
}

func assertCount(t *testing.T, database *gorm.DB, query string, want int64, args ...any) {
	t.Helper()
	var got int64
	if err := database.Raw(query, args...).Scan(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", query, got, want)
	}
}
