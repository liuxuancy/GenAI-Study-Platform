package course

import (
	"context"
	"errors"
	"hash/crc64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/cos"
	"caveclass/internal/platform/errs"
)

type resourceTransport func(*http.Request) (*http.Response, error)

func (transport resourceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestStudentResourceUploadOwnershipAndSharing(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t, Module{})
	if err := Seed(ctx, db); err != nil {
		t.Fatal(err)
	}
	service := NewService(db)
	owner := studentPrincipal(t, db, "student1")
	classmate := insertPrincipal(t, db, "student2", access.RoleStudent)
	teacher := teacherPrincipal(t, db, "teacher1")
	c := Course{Title: "Student upload course", OwnerID: owner.UserID}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, db, `INSERT INTO study_class_member (class_id, user_id) VALUES (?, ?)`, classIDByName(t, db, seedClassName), classmate.UserID)
	client, err := cos.NewClient(cos.Config{Bucket: "test-123", Region: "test", SecretID: "test", SecretKey: "test"})
	if err != nil {
		t.Fatal(err)
	}
	service.UseCOS(client, cos.Limits{DocMaxBytes: 1024, ArchiveMaxBytes: 1024})
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	uploads := 0
	http.DefaultTransport = resourceTransport(func(request *http.Request) (*http.Response, error) {
		response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: request}
		switch request.Method {
		case http.MethodPut:
			uploads++
			data, err := io.ReadAll(request.Body)
			if err != nil || string(data) != "%PDF" {
				t.Errorf("upload body %q, err %v", data, err)
			}
			response.Header.Set("x-cos-hash-crc64ecma", strconv.FormatUint(crc64.Checksum(data, crc64.MakeTable(crc64.ECMA)), 10))
		case http.MethodHead:
			response.Header.Set("Content-Length", strconv.Itoa(4))
			response.ContentLength = 4
		default:
			t.Errorf("unexpected COS method %s", request.Method)
		}
		return response, nil
	})
	upload := func(principal *access.Principal) (*CourseResourceItem, error) {
		return service.UploadStudentCourseResource(ctx, principal, c.ID, "notes.pdf", 4, strings.NewReader("%PDF"), UploadCourseResourceInput{})
	}
	if _, err := upload(classmate); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("classmate upload error = %v", err)
	}
	if _, err := upload(teacher); !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("teacher upload error = %v", err)
	}
	created, err := upload(owner)
	if err != nil || created.PublishStatus != PublishStatusPublished || uploads != 1 {
		t.Fatalf("created = %+v, uploads = %d, error = %v", created, uploads, err)
	}
	ownerList, err := service.ListStudentCourseResources(ctx, owner, c.ID, "", "", "")
	if err != nil || ownerList.Total != 1 {
		t.Fatalf("owner list = %+v, error = %v", ownerList, err)
	}
	if _, err := service.GetCourseResourceDownload(ctx, classmate, c.ID, created.ID); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("private download error = %v", err)
	}
	if _, err := service.SetCourseSharing(ctx, owner, c.ID, true); err != nil {
		t.Fatal(err)
	}
	list, err := service.ListStudentCourseResources(ctx, classmate, c.ID, "", "", "")
	if err != nil || list.Total != 1 {
		t.Fatalf("shared list = %+v, error = %v", list, err)
	}
	if _, err := service.GetCourseResourcePreview(ctx, classmate, c.ID, created.ID); err != nil {
		t.Fatalf("shared preview: %v", err)
	}
	if _, err := service.GetCourseResourceDownload(ctx, classmate, c.ID, created.ID); err != nil {
		t.Fatalf("shared download: %v", err)
	}
	if _, err := upload(classmate); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("classmate upload after share error = %v", err)
	}
	if _, err := service.SetCourseSharing(ctx, owner, c.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetCourseResourcePreview(ctx, classmate, c.ID, created.ID); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("preview after unshare error = %v", err)
	}
}
