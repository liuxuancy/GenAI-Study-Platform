package router

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"caveclass/internal/modules/course"
)

func TestStudentResourceUploadRouteAccess(t *testing.T) {
	t.Setenv("COS_SECRET_ID", "")
	t.Setenv("COS_SECRET_KEY", "")
	server := newTestServer(t, course.Module{})
	if err := course.Seed(context.Background(), server.database); err != nil {
		t.Fatal(err)
	}
	studentToken := server.login(t, "student1", "student123")
	teacherToken := server.login(t, "teacher1", "teacher123")
	_, result := server.call(t, "POST", "/api/student/courses", studentToken, map[string]string{"title": "Resource upload route"})
	courseID := int64(dataAsMap(result.Data)["id"].(float64))
	path := fmt.Sprintf("/api/student/courses/%d/resources", courseID)
	for _, test := range []struct {
		name  string
		token string
		want  int
	}{
		{"anonymous", "", 401},
		{"teacher", teacherToken, 403},
		{"owner without storage", studentToken, 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			file, err := writer.CreateFormFile("file", "notes.pdf")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("%PDF")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("POST", path, &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}
			recorder := httptest.NewRecorder()
			server.httpHandler.ServeHTTP(recorder, req)
			if recorder.Code != test.want {
				t.Fatalf("status = %d, want %d, body = %s", recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
}
