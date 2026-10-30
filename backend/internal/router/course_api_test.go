package router

import (
	"context"
	"testing"

	"caveclass/internal/modules/course"
)

func TestCourseRoutes(t *testing.T) {
	server := newTestServer(t, course.Module{})
	if err := course.Seed(context.Background(), server.database); err != nil {
		t.Fatal(err)
	}
	studentToken := server.login(t, "student1", "student123")
	teacherToken := server.login(t, "teacher1", "teacher123")
	adminToken := server.adminToken(t)

	status, result := server.call(t, "POST", "/api/student/courses", studentToken, map[string]string{
		"title": "Private notes", "description": "mine",
	})
	if status != 200 || result.Code != 0 {
		t.Fatalf("create student course: status=%d msg=%q", status, result.Msg)
	}
	if dataAsMap(result.Data)["visibility"] != "private" {
		t.Fatalf("visibility = %v", result.Data)
	}
	if status, result := server.call(t, "POST", "/api/student/courses", teacherToken, map[string]string{"title": "Nope"}); status != 403 {
		t.Fatalf("teacher on student create: status=%d msg=%q", status, result.Msg)
	}

	status, result = server.call(t, "GET", "/api/teacher/courses", teacherToken, nil)
	if status != 200 {
		t.Fatalf("teacher list: status=%d msg=%q", status, result.Msg)
	}
	status, result = server.call(t, "PUT", "/api/teacher/courses/1/classes", teacherToken, map[string]any{})
	if status != 400 {
		t.Fatalf("missing classIds: status=%d msg=%q", status, result.Msg)
	}

	status, result = server.call(t, "GET", "/api/system/classes", adminToken, nil)
	if status != 200 {
		t.Fatalf("class list: status=%d msg=%q", status, result.Msg)
	}
	classes := dataAsList(result.Data)
	if len(classes) != 1 || dataAsMap(classes[0])["name"] != "示例班" {
		t.Fatalf("classes = %v", result.Data)
	}
	if status, _ := server.call(t, "GET", "/api/system/classes", teacherToken, nil); status != 403 {
		t.Fatalf("teacher classes admin route: status = %d", status)
	}
}
