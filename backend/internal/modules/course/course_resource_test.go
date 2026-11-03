package course

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/cos"
)

func TestCourseResourcePublishVisibility(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	teacher := teacherPrincipal(t, database, "teacher1")

	draft, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Resource course"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	row := CourseResource{
		CourseID:        draft.ID,
		UploaderID:      teacher.UserID,
		OriginalName:    "notes.pdf",
		DisplayName:     "讲义.pdf",
		ObjectKey:       "courses/1/originals/test-uuid.pdf",
		FileExtension:   "pdf",
		MimeType:        "application/pdf",
		FileSize:        100,
		StorageProvider: ResourceStorageTencentCOS,
		ScopeType:       ResourceScopeCourse,
		PublishStatus:   PublishStatusDraft,
		PreviewStatus:   ResourcePreviewReady,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := database.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	teacherList, err := service.ListTeacherCourseResources(context.Background(), teacher, draft.ID, "draft", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if teacherList.Total != 1 {
		t.Fatalf("teacher list = %+v", teacherList)
	}

	published, err := service.PublishTeacherCourseResource(context.Background(), teacher, draft.ID, row.ID, PublishCourseResourceInput{Published: true})
	if err != nil {
		t.Fatal(err)
	}
	if published.PublishStatus != PublishStatusPublished {
		t.Fatalf("publish = %+v", published)
	}
}

func TestCourseResourceIDOR(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	teacher := teacherPrincipal(t, database, "teacher1")
	other := teacherPrincipal(t, database, "teacher2")

	courseB, err := service.CreateTeacherCourse(context.Background(), other, CreateCourseInput{Title: "Course B resources"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	row := CourseResource{
		CourseID:        courseB.ID,
		UploaderID:      other.UserID,
		OriginalName:    "secret.pdf",
		DisplayName:     "secret.pdf",
		ObjectKey:       "courses/2/originals/other.pdf",
		FileExtension:   "pdf",
		MimeType:        "application/pdf",
		FileSize:        1,
		StorageProvider: ResourceStorageTencentCOS,
		ScopeType:       ResourceScopeCourse,
		PublishStatus:   PublishStatusPublished,
		PreviewStatus:   ResourcePreviewReady,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := database.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	_, err = service.ListTeacherCourseResources(context.Background(), teacher, courseB.ID, "", "", "", "")
	if err == nil {
		t.Fatal("teacher1 must not list teacher2 course resources")
	}

	_, err = service.PublishTeacherCourseResource(context.Background(), teacher, courseB.ID, row.ID, PublishCourseResourceInput{Published: false})
	if err == nil {
		t.Fatal("teacher1 must not publish on foreign course")
	}
}

func TestNormalizeResourceScope(t *testing.T) {
	st, ck, sk, err := normalizeResourceScope("section", "ch1", "act1")
	if err != nil || st != ResourceScopeSection || ck != "ch1" || sk != "act1" {
		t.Fatalf("scope = %s %s %s err=%v", st, ck, sk, err)
	}
	_, _, _, err = normalizeResourceScope("chapter", "", "")
	if err == nil {
		t.Fatal("chapter scope requires chapterKey")
	}
}

func TestValidateResourceFileRejectsOversize(t *testing.T) {
	limits := cos.Limits{DocMaxBytes: 1024, ArchiveMaxBytes: 2048}
	_, _, err := validateResourceFile("big.pdf", 2048, []byte{0x25, 0x50, 0x44, 0x46}, limits)
	if err == nil {
		t.Fatal("expected oversize error")
	}
}

func TestCourseResourceStudentSeesPublishedOnly(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	course, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Student resource course"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i, pub := range []string{PublishStatusDraft, PublishStatusPublished} {
		row := CourseResource{
			CourseID:        course.ID,
			UploaderID:      student.UserID,
			OriginalName:    "file.pdf",
			DisplayName:     pub,
			ObjectKey:       fmt.Sprintf("courses/%d/originals/student-%d.pdf", course.ID, i),
			FileExtension:   "pdf",
			MimeType:        "application/pdf",
			FileSize:        10,
			StorageProvider: ResourceStorageTencentCOS,
			ScopeType:       ResourceScopeCourse,
			PublishStatus:   pub,
			PreviewStatus:   ResourcePreviewReady,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := database.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	list, err := service.ListStudentCourseResources(context.Background(), student, course.ID, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || list.Items[0].PublishStatus != PublishStatusPublished {
		t.Fatalf("student list = %+v", list)
	}
}

func TestCourseResourceSoftDeleteHidden(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	teacher := teacherPrincipal(t, database, "teacher1")
	course, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Delete resource course"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	row := CourseResource{
		CourseID:        course.ID,
		UploaderID:      teacher.UserID,
		OriginalName:    "gone.pdf",
		DisplayName:     "gone.pdf",
		ObjectKey:       "courses/99/originals/gone.pdf",
		FileExtension:   "pdf",
		MimeType:        "application/pdf",
		FileSize:        1,
		StorageProvider: ResourceStorageTencentCOS,
		ScopeType:       ResourceScopeCourse,
		PublishStatus:   PublishStatusPublished,
		PreviewStatus:   ResourcePreviewReady,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := database.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteTeacherCourseResource(context.Background(), teacher, course.ID, row.ID); err != nil {
		t.Fatal(err)
	}
	teacherList, err := service.ListTeacherCourseResources(context.Background(), teacher, course.ID, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if teacherList.Total != 0 {
		t.Fatalf("deleted resource still listed: %+v", teacherList)
	}
}

func TestUploadCourseResourceRequiresCOS(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	teacher := teacherPrincipal(t, database, "teacher1")
	course, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Upload COS test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.UploadTeacherCourseResource(context.Background(), teacher, course.ID, "a.pdf", 4,
		strings.NewReader("%PDF"), UploadCourseResourceInput{})
	if err == nil {
		t.Fatal("upload without COS must fail")
	}
}

func TestGetCourseResourcePreviewOfficeWithoutCI(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	teacher := teacherPrincipal(t, database, "teacher1")
	course, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Preview office"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	row := CourseResource{
		CourseID:        course.ID,
		UploaderID:      teacher.UserID,
		OriginalName:    "slides.pptx",
		DisplayName:     "slides.pptx",
		ObjectKey:       fmt.Sprintf("courses/%d/originals/slides.pptx", course.ID),
		FileExtension:   "pptx",
		MimeType:        "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		FileSize:        100,
		StorageProvider: ResourceStorageTencentCOS,
		ScopeType:       ResourceScopeCourse,
		PublishStatus:   PublishStatusDraft,
		PreviewStatus:   ResourcePreviewUnsupported,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := database.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	preview, err := service.GetCourseResourcePreview(context.Background(), teacher, course.ID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Kind != "unsupported" || preview.Message == "" {
		t.Fatalf("preview = %+v", preview)
	}
}
