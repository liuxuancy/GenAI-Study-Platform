package course

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/errs"
)

func TestGetCourseList(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")

	studentItems, err := service.GetCourseList(context.Background(), &access.Principal{
		UserID: studentID,
		Role:   access.RoleStudent,
	})
	if err != nil {
		t.Fatalf("student list: %v", err)
	}
	wantStudent := []CourseListItem{
		{Title: "Introduction to Git", CourseType: CourseTypeStructured, Percent: 0},
		{Title: "How a lecture is read", CourseType: CourseTypeStructured, Percent: 0},
		{Title: "Asking on a line", CourseType: CourseTypeStructured, Percent: 0},
	}
	assertTitles(t, studentItems, wantStudent)

	teacherAsStudent, err := service.GetCourseList(context.Background(), &access.Principal{
		UserID: teacherID,
		Role:   access.RoleStudent,
	})
	if err != nil {
		t.Fatalf("teacher-as-student list: %v", err)
	}
	assertTitles(t, teacherAsStudent, []CourseListItem{
		{Title: "Introduction to Git", CourseType: CourseTypeStructured, Percent: 0},
		{Title: "How a lecture is read", CourseType: CourseTypeStructured, Percent: 0},
		{Title: "Draft lecture", CourseType: CourseTypeStructured, Percent: 0},
	})

	_, err = service.GetCourseList(context.Background(), &access.Principal{
		UserID: teacherID,
		Role:   access.RoleTeacher,
	})
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher list error = %v, want forbidden", err)
	}
}

func TestSeedCourseType(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	assertCourseTypes(t, database, CourseTypeStructured)

	testutil.MustExec(t, database, "UPDATE course SET course_type = '' WHERE title = ?", "Draft lecture")
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed again: %v", err)
	}
	var courseType string
	err := database.Raw("SELECT course_type FROM course WHERE title = ?", "Draft lecture").Scan(&courseType).Error
	if err != nil {
		t.Fatal(err)
	}
	if courseType != "" {
		t.Fatalf("empty course type = %q, want empty", courseType)
	}
}

func TestSeedFillsEmptyMajor(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	assertCourseMajors(t, database, seedMajor)

	studentID := testutil.GetUserID(t, database, "student1")
	testutil.MustExec(t, database, "UPDATE course SET major = ? WHERE title = ?", "Mathematics", "Draft lecture")
	testutil.MustExec(t, database, "UPDATE course SET major = '' WHERE title = ?", "Asking on a line")
	testutil.MustExec(t, database,
		`INSERT INTO student_course_progress (user_id, course_id, percent)
		 VALUES (?, (SELECT id FROM course WHERE title = ?), 25)`,
		studentID, "Introduction to Git",
	)

	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed again: %v", err)
	}

	assertColumn(t, database, "SELECT major FROM course WHERE title = ?", "Draft lecture", "Mathematics")
	assertColumn(t, database, "SELECT COALESCE(major, '') FROM course WHERE title = ?", "Asking on a line", seedMajor)
	var percent int
	err := database.Raw(
		"SELECT percent FROM student_course_progress WHERE user_id = ? AND course_id = (SELECT id FROM course WHERE title = ?)",
		studentID, "Introduction to Git",
	).Scan(&percent).Error
	if err != nil {
		t.Fatal(err)
	}
	if percent != 25 {
		t.Fatalf("progress = %d, want 25", percent)
	}
}

func TestLegacyQuestionSeedFillsEmptyDifficulty(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	seedCoursesAndLegacyQuestions(t, database)
	assertSeedDifficulties(t, database)

	testutil.MustExec(t, database, "UPDATE course_question SET difficulty = ? WHERE title = ?", "hard", "What is a commit?")
	testutil.MustExec(t, database, "UPDATE course_question SET difficulty = '' WHERE title = ?", "Where do you ask?")

	seedLegacyQuestionSetsAgain(t, database)

	assertColumn(t, database, "SELECT difficulty FROM course_question WHERE title = ?", "What is a commit?", "hard")
	assertColumn(t, database, "SELECT COALESCE(difficulty, '') FROM course_question WHERE title = ?", "Where do you ask?", "easy")
	assertColumn(t, database, "SELECT difficulty FROM course_question WHERE title = ?", "Which paragraph comes first?", "medium")
}

func assertCourseMajors(t *testing.T, database *gorm.DB, want string) {
	t.Helper()
	var majors []string
	err := database.Raw("SELECT COALESCE(major, '') FROM course ORDER BY id").Scan(&majors).Error
	if err != nil {
		t.Fatal(err)
	}
	if len(majors) != 4 {
		t.Fatalf("courses = %d, want 4", len(majors))
	}
	for index, major := range majors {
		if major != want {
			t.Errorf("course %d major = %q, want %q", index, major, want)
		}
	}
}

func assertSeedDifficulties(t *testing.T, database *gorm.DB) {
	t.Helper()
	assertColumn(t, database, "SELECT difficulty FROM course_question WHERE title = ?", "What is a commit?", "easy")
	assertColumn(t, database, "SELECT difficulty FROM course_question WHERE title = ?", "Which paragraph comes first?", "medium")
	assertColumn(t, database, "SELECT difficulty FROM course_question WHERE title = ?", "Where do you ask?", "easy")
}

func assertColumn(t *testing.T, database *gorm.DB, query string, title string, want string) {
	t.Helper()
	var got string
	err := database.Raw(query, title).Scan(&got).Error
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", title, got, want)
	}
}

func assertCourseTypes(t *testing.T, database *gorm.DB, want string) {
	t.Helper()
	var courseTypes []string
	err := database.Raw("SELECT course_type FROM course ORDER BY id").Scan(&courseTypes).Error
	if err != nil {
		t.Fatal(err)
	}
	if len(courseTypes) != 4 {
		t.Fatalf("courses = %d, want 4", len(courseTypes))
	}
	for index, courseType := range courseTypes {
		if courseType != want {
			t.Errorf("course %d type = %q, want %q", index, courseType, want)
		}
	}
}

func TestUpdateCourseProgress(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")
	student := &access.Principal{UserID: studentID, Role: access.RoleStudent}

	_, err := service.UpdateCourseProgress(context.Background(), student, courseIDByTitle(t, database, "Introduction to Git"), 101)
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("high percent: %v", err)
	}
	_, err = service.UpdateCourseProgress(context.Background(), student, courseIDByTitle(t, database, "Introduction to Git"), -1)
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("low percent: %v", err)
	}
	_, err = service.UpdateCourseProgress(context.Background(), &access.Principal{UserID: teacherID, Role: access.RoleTeacher}, courseIDByTitle(t, database, "Introduction to Git"), 8)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher: %v", err)
	}
	_, err = service.UpdateCourseProgress(context.Background(), student, courseIDByTitle(t, database, "Draft lecture"), 8)
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("hidden course: %v", err)
	}

	updated, err := service.UpdateCourseProgress(context.Background(), student, courseIDByTitle(t, database, "Introduction to Git"), 8)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Percent != 8 {
		t.Fatalf("updated = %+v", updated)
	}
	items, err := service.GetCourseList(context.Background(), student)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	assertTitles(t, items, []CourseListItem{
		{Title: "Introduction to Git", CourseType: CourseTypeStructured, Percent: 8},
		{Title: "How a lecture is read", CourseType: CourseTypeStructured, Percent: 0},
		{Title: "Asking on a line", CourseType: CourseTypeStructured, Percent: 0},
	})

	askingID := courseIDByTitle(t, database, "Asking on a line")
	testutil.MustExec(t, database, "DELETE FROM student_course_progress WHERE user_id = ? AND course_id = ?", studentID, askingID)
	inserted, err := service.UpdateCourseProgress(context.Background(), student, askingID, 0)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if inserted.Percent != 0 {
		t.Fatalf("inserted = %+v", inserted)
	}
	var stored int
	err = database.Raw(
		"SELECT percent FROM student_course_progress WHERE user_id = ? AND course_id = ?",
		studentID, askingID,
	).Scan(&stored).Error
	if err != nil {
		t.Fatal(err)
	}
	if stored != 0 {
		t.Fatalf("stored percent = %d", stored)
	}
}

func TestGetQuestionList(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	seedCoursesAndLegacyQuestions(t, database)
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed again: %v", err)
	}
	testutil.MustExec(t, database, "ALTER TABLE course_question_content ADD COLUMN user_option TEXT")
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("drop leftover user_option: %v", err)
	}
	var leftover int64
	if err := database.Raw(
		"SELECT COUNT(*) FROM pragma_table_info('course_question_content') WHERE name = 'user_option'",
	).Scan(&leftover).Error; err != nil {
		t.Fatal(err)
	}
	if leftover != 0 {
		t.Fatal("content user_option column still exists")
	}
	assertQuestionSeed(t, database)

	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")
	otherStudentID := testutil.GetUserID(t, database, "teacher2")

	studentItems, err := service.GetQuestionList(context.Background(), &access.Principal{
		UserID: studentID,
		Role:   access.RoleStudent,
	})
	if err != nil {
		t.Fatalf("student questions: %v", err)
	}
	assertQuestions(t, studentItems, []QuestionListItem{
		{Title: "What is a commit?", CourseTitle: "Introduction to Git", Answered: 2, QuestionCount: 2, Percent: 100},
		{Title: "Which paragraph comes first?", CourseTitle: "How a lecture is read", Answered: 1, QuestionCount: 2, Percent: 50},
		{Title: "Where do you ask?", CourseTitle: "Asking on a line", Answered: 0, QuestionCount: 2, Percent: 0},
	})

	otherStudent, err := service.GetQuestionList(context.Background(), &access.Principal{
		UserID: otherStudentID,
		Role:   access.RoleStudent,
	})
	if err != nil {
		t.Fatalf("other student questions: %v", err)
	}
	if len(otherStudent) != 0 {
		t.Fatalf("user outside the class sees %v", otherStudent)
	}

	_, err = service.GetQuestionList(context.Background(), &access.Principal{
		UserID: teacherID,
		Role:   access.RoleTeacher,
	})
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher questions error = %v, want forbidden", err)
	}
}

func assertQuestionSeed(t *testing.T, database *gorm.DB) {
	t.Helper()
	var setCount int64
	if err := database.Raw("SELECT COUNT(*) FROM course_question").Scan(&setCount).Error; err != nil {
		t.Fatal(err)
	}
	if setCount != 3 {
		t.Fatalf("question sets = %d, want 3", setCount)
	}
	var contentCount int64
	if err := database.Raw("SELECT COUNT(*) FROM course_question_content").Scan(&contentCount).Error; err != nil {
		t.Fatal(err)
	}
	if contentCount != 6 {
		t.Fatalf("question contents = %d, want 6", contentCount)
	}
	var unpublished int64
	err := database.Raw(
		"SELECT COUNT(*) FROM course_question WHERE title = ? AND is_publish = ? AND question_count = ?",
		"Where do you ask?", PublishOff, 2,
	).Scan(&unpublished).Error
	if err != nil {
		t.Fatal(err)
	}
	if unpublished != 1 {
		t.Fatalf("unpublished student set = %d, want 1", unpublished)
	}
	var explanationCount int64
	err = database.Raw("SELECT COUNT(*) FROM course_question_content WHERE explanation <> ''").Scan(&explanationCount).Error
	if err != nil {
		t.Fatal(err)
	}
	if explanationCount != 6 {
		t.Fatalf("explanations = %d, want 6", explanationCount)
	}
}

func TestGetCourse(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")
	student := &access.Principal{UserID: studentID, Role: access.RoleStudent}

	detail, err := service.GetCourse(context.Background(), student, courseIDByTitle(t, database, "Introduction to Git"))
	if err != nil {
		t.Fatalf("published course: %v", err)
	}
	if detail.Title != "Introduction to Git" || detail.Tree != nil || detail.Language != DefaultLanguage || detail.Visibility != VisibilityAssigned {
		t.Fatalf("published course = %+v", detail)
	}

	own, err := service.GetCourse(context.Background(), student, courseIDByTitle(t, database, "Asking on a line"))
	if err != nil {
		t.Fatalf("own unpublished course: %v", err)
	}
	if own.Title != "Asking on a line" {
		t.Fatalf("own title = %q", own.Title)
	}

	_, err = service.GetCourse(context.Background(), student, courseIDByTitle(t, database, "Draft lecture"))
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("hidden course: %v", err)
	}

	_, err = service.GetCourse(context.Background(), &access.Principal{UserID: teacherID, Role: access.RoleTeacher}, detail.ID)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher: %v", err)
	}
}

func TestGetQuestion(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	seedCoursesAndLegacyQuestions(t, database)
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")
	student := &access.Principal{UserID: studentID, Role: access.RoleStudent}

	own, err := service.GetQuestion(context.Background(), student, questionIDByTitle(t, database, "Where do you ask?"))
	if err != nil {
		t.Fatalf("own question: %v", err)
	}
	if own.Title != "Where do you ask?" || own.CourseTitle != "Asking on a line" || own.QuestionCount != 2 || own.Percent != 0 {
		t.Fatalf("own detail = %+v", own)
	}
	if len(own.Questions) != 2 || own.Questions[0].Content != "Where do you ask a question?" {
		t.Fatalf("contents = %+v", own.Questions)
	}
	if len(own.Questions[0].Options) != 2 || own.Questions[0].Options[0] != "On the line" || own.Questions[0].Options[1] != "In a separate room" {
		t.Fatalf("options = %#v", own.Questions[0].Options)
	}
	if own.Questions[0].UserOption != "" {
		t.Fatalf("user option = %q", own.Questions[0].UserOption)
	}
	if own.IsSubmit {
		t.Fatal("expected unsubmitted own question")
	}
	if own.Questions[0].CorrectAnswer != "" || own.Questions[0].Explanation != "" {
		t.Fatalf("expected redacted answers before submit, got %+v", own.Questions[0])
	}

	published, err := service.GetQuestion(context.Background(), student, questionIDByTitle(t, database, "What is a commit?"))
	if err != nil {
		t.Fatalf("published question: %v", err)
	}
	if published.Percent != 100 || published.QuestionCount != 2 || len(published.Questions[0].Options) != 2 {
		t.Fatalf("published detail = %+v", published)
	}
	if published.Questions[0].UserOption != "A snapshot of the work" {
		t.Fatalf("published user option = %q", published.Questions[0].UserOption)
	}
	if published.IsSubmit {
		t.Fatal("seed published set should not be marked submitted")
	}
	if published.Questions[0].CorrectAnswer != "" || published.Questions[0].Explanation != "" {
		t.Fatalf("expected redacted answers before submit, got %+v", published.Questions[0])
	}

	_, err = service.GetQuestion(context.Background(), &access.Principal{UserID: teacherID, Role: access.RoleStudent}, own.ID)
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("other student: %v", err)
	}

	_, err = service.GetQuestion(context.Background(), &access.Principal{UserID: teacherID, Role: access.RoleTeacher}, published.ID)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher: %v", err)
	}
}

func TestAnswerQuestion(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	seedCoursesAndLegacyQuestions(t, database)
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")
	student := &access.Principal{UserID: studentID, Role: access.RoleStudent}
	questionID := questionIDByTitle(t, database, "Where do you ask?")

	detail, err := service.GetQuestion(context.Background(), student, questionID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	first := detail.Questions[0]
	second := detail.Questions[1]

	_, err = service.AnswerQuestion(context.Background(), student, questionID, first.ID, "Nowhere", false)
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("invalid option: %v", err)
	}

	_, err = service.AnswerQuestion(context.Background(), &access.Principal{UserID: teacherID, Role: access.RoleStudent}, questionID, first.ID, "On the line", false)
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("other student: %v", err)
	}

	published, err := service.GetQuestion(context.Background(), student, questionIDByTitle(t, database, "What is a commit?"))
	if err != nil {
		t.Fatalf("published: %v", err)
	}
	_, err = service.AnswerQuestion(context.Background(), &access.Principal{UserID: teacherID, Role: access.RoleTeacher}, published.ID, published.Questions[0].ID, published.Questions[0].Options[0], false)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher: %v", err)
	}

	_, err = service.AnswerQuestion(context.Background(), student, questionID, published.Questions[0].ID, "On the line", false)
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("other set: %v", err)
	}

	replaced, err := service.AnswerQuestion(context.Background(), student, questionID, first.ID, "In a separate room", false)
	if err != nil {
		t.Fatalf("first answer: %v", err)
	}
	if replaced.Percent != 50 || replaced.Questions[0].UserOption != "In a separate room" {
		t.Fatalf("after first = %+v", replaced)
	}
	replaced, err = service.AnswerQuestion(context.Background(), student, questionID, first.ID, "On the line", false)
	if err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if replaced.Percent != 50 || replaced.Questions[0].UserOption != "On the line" {
		t.Fatalf("after overwrite = %+v", replaced)
	}

	done, err := service.AnswerQuestion(context.Background(), student, questionID, second.ID, second.Options[0], false)
	if err != nil {
		t.Fatalf("second answer: %v", err)
	}
	if done.Percent != 100 || done.Questions[1].UserOption != second.Options[0] {
		t.Fatalf("after second = %+v", done)
	}
	if submittedCount(t, database, studentID, questionID) != 0 {
		t.Fatal("save marked the set submitted")
	}

	_, err = service.AnswerQuestion(context.Background(), student, questionID, second.ID, second.Options[0], true)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if submittedCount(t, database, studentID, questionID) != 1 {
		t.Fatal("submit did not mark the set")
	}
	if submittedCount(t, database, teacherID, questionID) != 0 {
		t.Fatal("submit marked another user")
	}

	final, err := service.GetQuestion(context.Background(), student, questionID)
	if err != nil {
		t.Fatalf("after submit detail: %v", err)
	}
	if !final.IsSubmit || final.Score == nil || *final.Score != 50 {
		t.Fatalf("after submit = isSubmit %v score %v", final.IsSubmit, final.Score)
	}
	if final.Questions[0].CorrectAnswer != "On the line" || final.Questions[0].Explanation == "" {
		t.Fatalf("expected revealed answer after submit, got %+v", final.Questions[0])
	}
	if !final.Questions[0].IsCorrect || final.Questions[1].IsCorrect {
		t.Fatalf("isCorrect = %v %v", final.Questions[0].IsCorrect, final.Questions[1].IsCorrect)
	}
}

func submittedCount(t *testing.T, database *gorm.DB, userID int64, questionID int64) int64 {
	t.Helper()
	var count int64
	err := database.Raw(
		"SELECT COUNT(*) FROM student_question_record WHERE user_id = ? AND question_id = ? AND is_submit = ?",
		userID, questionID, SubmitOn,
	).Scan(&count).Error
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func TestCreateQuestion(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed questions: %v", err)
	}
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")
	student := &access.Principal{UserID: studentID, Role: access.RoleStudent}
	courseID := courseIDByTitle(t, database, "Introduction to Git")
	input := CreateQuestionInput{
		CourseID:   courseID,
		Difficulty: "easy",
		Questions:  sampleQuestions(2),
	}

	_, err := service.CreateQuestion(context.Background(), student, CreateQuestionInput{
		CourseID: courseID, Difficulty: "easy", Questions: sampleQuestions(3),
	})
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("bad count: %v", err)
	}
	_, err = service.CreateQuestion(context.Background(), student, CreateQuestionInput{
		CourseID: courseID, Difficulty: "expert", Questions: sampleQuestions(2),
	})
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("bad difficulty: %v", err)
	}
	wrong := sampleQuestions(2)
	wrong[0].CorrectAnswer = "neither"
	_, err = service.CreateQuestion(context.Background(), student, CreateQuestionInput{
		CourseID: courseID, Difficulty: "easy", Questions: wrong,
	})
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Fatalf("bad answer: %v", err)
	}
	_, err = service.CreateQuestion(context.Background(), student, CreateQuestionInput{
		CourseID: courseIDByTitle(t, database, "Draft lecture"), Difficulty: "easy", Questions: sampleQuestions(2),
	})
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("hidden course: %v", err)
	}
	_, err = service.CreateQuestion(context.Background(), &access.Principal{UserID: teacherID, Role: access.RoleTeacher}, input)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher: %v", err)
	}

	created, err := service.CreateQuestion(context.Background(), student, input)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.CourseTitle != "Introduction to Git" || created.Difficulty != "easy" || created.Answered != 0 || created.QuestionCount != 2 || created.Percent != 0 || created.ID == 0 {
		t.Fatalf("created = %+v", created)
	}

	detail, err := service.GetQuestion(context.Background(), student, created.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.QuestionCount != 2 || detail.Percent != 0 || detail.Difficulty != "easy" || len(detail.Questions) != 2 {
		t.Fatalf("stored = %+v", detail)
	}
	if detail.Questions[0].Content != "Introduction to Git: Easy question 1" || detail.Questions[0].UserOption != "" {
		t.Fatalf("first = %+v", detail.Questions[0])
	}
	var unpublished int64
	err = database.Raw(
		"SELECT COUNT(*) FROM course_question WHERE id = ? AND is_publish = ? AND creator_id = ? AND difficulty = ?",
		created.ID, PublishOff, studentID, "easy",
	).Scan(&unpublished).Error
	if err != nil {
		t.Fatal(err)
	}
	if unpublished != 1 {
		t.Fatalf("unpublished = %d", unpublished)
	}
	var seedDifficulty string
	err = database.Raw(
		"SELECT COALESCE(difficulty, '') FROM course_question WHERE title = ?",
		"What is a commit?",
	).Scan(&seedDifficulty).Error
	if err != nil {
		t.Fatal(err)
	}
	if seedDifficulty != "easy" {
		t.Fatalf("seed difficulty = %q", seedDifficulty)
	}

	items, err := service.GetQuestionList(context.Background(), student)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, item := range items {
		if item.ID == created.ID && item.Difficulty == "easy" && item.Answered == 0 && item.QuestionCount == 2 && item.Percent == 0 && item.CourseTitle == "Introduction to Git" {
			found = true
		}
	}
	if !found {
		t.Fatalf("list missing created set: %+v", items)
	}
}

func sampleQuestions(count int) []CreateQuestionItem {
	items := make([]CreateQuestionItem, 0, count)
	for index := 1; index <= count; index++ {
		first := "Choice A for question " + itoa(index)
		items = append(items, CreateQuestionItem{
			Content:       "Introduction to Git: Easy question " + itoa(index),
			Options:       []string{first, "Choice B for question " + itoa(index)},
			CorrectAnswer: first,
		})
	}
	return items
}

func itoa(value int) string {
	return string(rune('0' + value))
}

func courseIDByTitle(t *testing.T, database *gorm.DB, title string) int64 {
	t.Helper()
	return idByTitle(t, database, "SELECT id FROM course WHERE title = ?", title)
}

func questionIDByTitle(t *testing.T, database *gorm.DB, title string) int64 {
	t.Helper()
	return idByTitle(t, database, "SELECT id FROM course_question WHERE title = ?", title)
}

func idByTitle(t *testing.T, database *gorm.DB, query string, title string) int64 {
	t.Helper()
	var id int64
	if err := database.Raw(query, title).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatalf("%q has no id", title)
	}
	return id
}

func assertQuestions(t *testing.T, got []QuestionListItem, want []QuestionListItem) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for index := range want {
		if got[index].Title != want[index].Title || got[index].CourseTitle != want[index].CourseTitle || got[index].Answered != want[index].Answered || got[index].QuestionCount != want[index].QuestionCount || got[index].Percent != want[index].Percent {
			t.Errorf("item %d = %+v, want %+v", index, got[index], want[index])
		}
		if got[index].ID == 0 {
			t.Errorf("item %d id is 0", index)
		}
	}
}

func assertTitles(t *testing.T, got []CourseListItem, want []CourseListItem) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for index := range want {
		if got[index].Title != want[index].Title || got[index].CourseType != want[index].CourseType || got[index].Percent != want[index].Percent {
			t.Errorf("item %d = %+v, want %+v", index, got[index], want[index])
		}
		if got[index].ID == 0 {
			t.Errorf("item %d id is 0", index)
		}
	}
}
