package course

import (
	"context"
	"errors"
	"testing"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/errs"
)

func TestAdminUnpublishKeepsOwnerClassAndProgress(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	list, err := service.GetAdminCourseList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 4 || list.PublishedCount != 2 {
		t.Fatalf("list = total %d published %d", list.Total, list.PublishedCount)
	}

	gitID := courseIDByTitle(t, database, "Introduction to Git")
	studentID := testutil.GetUserID(t, database, "student1")
	testutil.MustExec(t, database,
		"INSERT INTO student_course_progress (user_id, course_id, percent) VALUES (?, ?, 35)",
		studentID, gitID,
	)
	testutil.MustExec(t, database, "UPDATE course SET content_json = ? WHERE id = ?", sampleContent, gitID)
	student := studentPrincipal(t, database, "student1")
	teacher := teacherPrincipal(t, database, "teacher1")
	if _, err := service.GetCourse(context.Background(), student, gitID); err != nil {
		t.Fatalf("student before unpublish: %v", err)
	}

	removed, err := service.SetAdminCoursePublished(context.Background(), gitID, false)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Published || removed.Title != "Introduction to Git" || removed.ClassCount != 1 || removed.OwnerUsername != "teacher1" {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := service.GetCourse(context.Background(), student, gitID); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("student after unpublish: %v", err)
	}
	owned, err := service.GetTeacherCourseList(context.Background(), teacher)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range owned {
		if item.ID == gitID {
			found = !item.Published && len(item.ClassIDs) == 1
		}
	}
	if !found {
		t.Fatalf("teacher list = %+v", owned)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course_class WHERE course_id = ?", 1, gitID)
	assertCount(t, database, "SELECT COUNT(*) FROM student_course_progress WHERE course_id = ? AND percent = 35", 1, gitID)
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE id = ? AND content_json = ?", 1, gitID, sampleContent)

	restored, err := service.SetAdminCoursePublished(context.Background(), gitID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !restored.Published || restored.ClassCount != 1 {
		t.Fatalf("restored = %+v", restored)
	}
	if _, err := service.GetCourse(context.Background(), student, gitID); err != nil {
		t.Fatalf("student after restore: %v", err)
	}
	if _, err := service.SetAdminCoursePublished(context.Background(), 99999, false); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("missing course: %v", err)
	}
}

func TestAdminUnpublishStudentShare(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	owner := studentPrincipal(t, database, "student1")
	classmate := insertPrincipal(t, database, "student2", access.RoleStudent)
	classID := classIDByName(t, database, seedClassName)
	testutil.MustExec(t, database, "INSERT INTO study_class_member (class_id, user_id) VALUES (?, ?)", classID, classmate.UserID)

	created, err := service.CreateStudentCourse(context.Background(), owner, CreateCourseInput{Title: "Shared notes"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetCourseSharing(context.Background(), owner, created.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetCourse(context.Background(), classmate, created.ID); err != nil {
		t.Fatalf("classmate before: %v", err)
	}

	removed, err := service.SetAdminCoursePublished(context.Background(), created.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Published || removed.ClassCount != 0 || removed.OwnerRole != access.RoleStudent {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := service.GetCourse(context.Background(), classmate, created.ID); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("classmate after: %v", err)
	}
	if _, err := service.GetCourse(context.Background(), owner, created.ID); err != nil {
		t.Fatalf("owner after unpublish: %v", err)
	}
	if _, err := service.SetAdminCoursePublished(context.Background(), created.ID, true); err != nil {
		t.Fatal(err)
	}
	seen, err := service.GetCourse(context.Background(), classmate, created.ID)
	if err != nil || seen.Visibility != VisibilityClass {
		t.Fatalf("classmate after restore: %v %+v", err, seen)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course_class WHERE course_id = ?", 0, created.ID)
}
