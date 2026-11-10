package course

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
)

func TestFirstAvatarAchievementCreationAndReconciliation(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	other := insertPrincipal(t, database, "avatar-other", access.RoleStudent)

	created, err := service.CreateStudentCourse(ctx, student, CreateCourseInput{Title: "First companion"})
	if err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ? AND achievement = ?", 1, student.UserID, achievementFirstAvatar)
	result, err := service.SyncAchievements(ctx, student)
	if err != nil || len(result.Notify) != 1 || result.Notify[0].Key != achievementFirstAvatar {
		t.Fatalf("first sync = %+v, error %v", result, err)
	}
	result, err = service.SyncAchievements(ctx, student)
	if err != nil || len(result.Notify) != 0 {
		t.Fatalf("repeat sync = %+v, error %v", result, err)
	}
	testutil.MustExec(t, database, "INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, 0, 0)", other.UserID, created.ID)
	result, err = service.SyncAchievements(ctx, other)
	if err != nil || len(result.Unlocked) != 0 {
		t.Fatalf("other student = %+v, error %v", result, err)
	}

	testutil.MustExec(t, database, "DELETE FROM student_achievement WHERE user_id = ?", student.UserID)
	testutil.MustExec(t, database, "DELETE FROM student_course_progress WHERE user_id = ? AND course_id = ?", student.UserID, created.ID)
	result, err = service.SyncAchievements(ctx, student)
	if err != nil || len(result.Unlocked) != 0 {
		t.Fatalf("incomplete creation = %+v, error %v", result, err)
	}
	testutil.MustExec(t, database, "INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, 0, 0)", student.UserID, created.ID)
	result, err = service.SyncAchievements(ctx, student)
	if err != nil || len(result.Notify) != 1 || result.Notify[0].Key != achievementFirstAvatar {
		t.Fatalf("historical avatar = %+v, error %v", result, err)
	}
	if err := service.deleteCreatedCourse(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	result, err = service.SyncAchievements(ctx, student)
	if err != nil || len(result.Unlocked) != 1 || len(result.Notify) != 0 {
		t.Fatalf("after course deletion = %+v, error %v", result, err)
	}
	if _, err := service.CreateStudentCourse(ctx, student, CreateCourseInput{Title: "Another companion"}); err != nil {
		t.Fatal(err)
	}
	result, err = service.SyncAchievements(ctx, student)
	if err != nil || len(result.Unlocked) != 1 || len(result.Notify) != 0 {
		t.Fatalf("subsequent creation = %+v, error %v", result, err)
	}
}

func TestFirstAvatarAchievementFailureRollsBackCreation(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	testutil.MustExec(t, database, `CREATE TRIGGER fail_avatar_achievement BEFORE INSERT ON student_achievement
		BEGIN SELECT RAISE(FAIL, 'achievement failure'); END`)
	if _, err := service.CreateStudentCourse(ctx, student, CreateCourseInput{Title: "Failed companion"}); err == nil {
		t.Fatal("expected achievement insertion failure")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE title = ?", 0, "Failed companion")
	assertCount(t, database, "SELECT COUNT(*) FROM course_pixel_avatar", 0)
	assertCount(t, database, "SELECT COUNT(*) FROM student_course_progress WHERE user_id = ?", 0, student.UserID)
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ?", 0, student.UserID)
}

func TestAchievementsThresholdsVisibilityAndPersistence(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	other := insertPrincipal(t, database, "student2", access.RoleStudent)

	var ids []int64
	for i := 0; i < 5; i++ {
		course := Course{Title: fmt.Sprintf("Achievement course %d", i), OwnerID: student.UserID}
		if err := database.Create(&course).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, course.ID)
	}
	hidden := Course{Title: "Hidden course", OwnerID: other.UserID}
	if err := database.Create(&hidden).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.AdjustAffection(ctx, student.UserID, hidden.ID, 100); err != nil {
		t.Fatal(err)
	}
	if err := service.AdjustAffection(ctx, student.UserID, ids[0], 49); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ?", 0, student.UserID)
	if err := service.AdjustAffection(ctx, student.UserID, ids[0], 1); err != nil {
		t.Fatal(err)
	}
	first, err := service.SyncAchievements(ctx, student)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Unlocked) != 1 || first.Unlocked[0].Key != achievementAffection50 || len(first.Notify) != 1 {
		t.Fatalf("50%% result = %+v", first)
	}
	if err := service.AdjustAffection(ctx, student.UserID, ids[0], 49); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ?", 1, student.UserID)
	if err := service.AdjustAffection(ctx, student.UserID, ids[0], 1); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids[1:4] {
		if err := service.AdjustAffection(ctx, student.UserID, id, 100); err != nil {
			t.Fatal(err)
		}
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ?", 2, student.UserID)
	if err := service.AdjustAffection(ctx, student.UserID, ids[4], 100); err != nil {
		t.Fatal(err)
	}
	result, err := service.SyncAchievements(ctx, student)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unlocked) != 3 || len(result.Notify) != 2 {
		t.Fatalf("100%% result = %+v", result)
	}
	if err := service.AdjustAffection(ctx, student.UserID, ids[0], -100); err != nil {
		t.Fatal(err)
	}
	repeated, err := service.SyncAchievements(ctx, student)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeated.Unlocked) != 3 || len(repeated.Notify) != 0 {
		t.Fatalf("repeated result = %+v", repeated)
	}
	isolated, err := service.SyncAchievements(ctx, other)
	if err != nil || len(isolated.Unlocked) != 0 {
		t.Fatalf("other student result = %+v, error %v", isolated, err)
	}
	if _, err := service.SyncAchievements(ctx, &access.Principal{Role: access.RoleTeacher}); err == nil {
		t.Fatal("teacher should be forbidden")
	}
}

func TestAchievementsConcurrentNotificationClaim(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	course := Course{Title: "Concurrent achievement", OwnerID: student.UserID}
	if err := database.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.AdjustAffection(ctx, student.UserID, course.ID, 50); err != nil {
		t.Fatal(err)
	}
	results := make([]*AchievementSync, 2)
	errors := make([]error, 2)
	var wait sync.WaitGroup
	for i := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			results[index], errors[index] = service.SyncAchievements(ctx, student)
		}(i)
	}
	wait.Wait()
	total := 0
	for i, result := range results {
		if errors[i] != nil {
			t.Fatalf("sync error: %v", errors[i])
		}
		total += len(result.Notify)
	}
	if total != 1 {
		t.Fatalf("notification claims = %d, want 1", total)
	}
}

func TestAchievementsReconcileAndAtomicUpdate(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	for i := 0; i < 5; i++ {
		course := Course{Title: fmt.Sprintf("Existing course %d", i), OwnerID: student.UserID}
		if err := database.Create(&course).Error; err != nil {
			t.Fatal(err)
		}

		if err := database.Create(&StudentCourseProgress{UserID: student.UserID, CourseID: course.ID, Affection: 100}).Error; err != nil {
			t.Fatal(err)
		}
	}
	result, err := service.SyncAchievements(ctx, student)
	if err != nil || len(result.Notify) != 3 || len(result.Unlocked) != 3 {
		t.Fatalf("reconcile = %+v, error %v", result, err)
	}
	expected := errors.New("forced failure")
	err = service.withAffectionAchievements(ctx, student.UserID, func(txService *Service) error {
		if err := txService.database.Exec("UPDATE student_course_progress SET affection = 0 WHERE user_id = ?", student.UserID).Error; err != nil {
			return err
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v", err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_course_progress WHERE user_id = ? AND affection = 100", 5, student.UserID)
}

func TestAchievementsUnlockOnProgressAndAnswer(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	course := Course{Title: "Progress and answer achievements", OwnerID: student.UserID}
	if err := database.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	question := CourseQuestion{Title: "Achievement question", CourseID: course.ID, CreatorID: student.UserID, QuestionCount: 1}
	if err := database.Create(&question).Error; err != nil {
		t.Fatal(err)
	}
	content := CourseQuestionContent{QuestionID: question.ID, Content: "Question", Options: "A|B", CorrectAnswer: "A", Explanation: "Answer"}
	if err := database.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateCourseProgress(ctx, student, course.ID, 62); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ?", 0, student.UserID)
	if _, err := service.UpdateCourseProgress(ctx, student, course.ID, 63); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ? AND achievement = ?", 1, student.UserID, achievementAffection50)
	if _, err := service.UpdateCourseProgress(ctx, student, course.ID, 100); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ? AND achievement = ?", 1, student.UserID, achievementAffection50)
	if _, err := service.AnswerQuestion(ctx, student, question.ID, content.ID, "A", false); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_achievement WHERE user_id = ? AND achievement = ?", 1, student.UserID, achievementAffection100)
}

func TestAnswerRollsBackWhenAchievementCannotBeRecorded(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	course := Course{Title: "Achievement failure", OwnerID: student.UserID}
	if err := database.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	question := CourseQuestion{Title: "Failure question", CourseID: course.ID, CreatorID: student.UserID, QuestionCount: 1}
	if err := database.Create(&question).Error; err != nil {
		t.Fatal(err)
	}
	content := CourseQuestionContent{QuestionID: question.ID, Content: "Question", Options: "A|B", CorrectAnswer: "A", Explanation: "Answer"}
	if err := database.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateCourseProgress(ctx, student, course.ID, 40); err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, database, `CREATE TRIGGER reject_achievement BEFORE INSERT ON student_achievement
		BEGIN SELECT RAISE(ABORT, 'achievement write failed'); END`)
	if _, err := service.AnswerQuestion(ctx, student, question.ID, content.ID, "A", false); err == nil {
		t.Fatal("expected achievement write failure")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_question_progress WHERE user_id = ? AND content_id = ?", 0, student.UserID, content.ID)
	assertAffection(t, database, student.UserID, course.ID, 32)
}

func TestAchievementsReconcileAffectionWithoutProgressRow(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	course := Course{Title: "Legacy quiz affection", OwnerID: student.UserID}
	if err := database.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	question := CourseQuestion{Title: "Legacy quiz", CourseID: course.ID, CreatorID: student.UserID, QuestionCount: 1}
	if err := database.Create(&question).Error; err != nil {
		t.Fatal(err)
	}
	content := CourseQuestionContent{QuestionID: question.ID, Content: "Question", Options: "A|B", CorrectAnswer: "A", Explanation: "Answer"}
	if err := database.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&StudentQuestionProgress{UserID: student.UserID, QuestionID: question.ID, ContentID: content.ID, UserOption: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := service.SyncAchievements(ctx, student)
	if err != nil || len(result.Unlocked) != 0 {
		t.Fatalf("reconcile = %+v, error %v", result, err)
	}
	affection, err := service.studentAffection(ctx, student.UserID, course.ID)
	if err != nil || affection != 20 {
		t.Fatalf("affection = %d, error %v", affection, err)
	}
}
