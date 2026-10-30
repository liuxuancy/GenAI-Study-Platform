package router

import (
	"context"
	"fmt"
	"testing"
	"time"

	"caveclass/internal/framework/testutil"
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

	status, result = server.call(t, "GET", "/api/system/courses", adminToken, nil)
	if status != 200 {
		t.Fatalf("admin courses: status=%d msg=%q", status, result.Msg)
	}
	catalog := dataAsMap(result.Data)
	if catalog["total"] != float64(5) || catalog["publishedCount"] != float64(2) {
		t.Fatalf("catalog = %+v", catalog)
	}
	var gitID float64
	for _, item := range dataAsList(catalog["items"]) {
		row := dataAsMap(item)
		if row["title"] == "Introduction to Git" {
			gitID = row["id"].(float64)
		}
	}
	if gitID == 0 {
		t.Fatal("seed course missing")
	}
	status, result = server.call(t, "PUT", fmt.Sprintf("/api/system/courses/%.0f/publish", gitID), adminToken, map[string]any{"published": false})
	if status != 200 || dataAsMap(result.Data)["published"] != false {
		t.Fatalf("unpublish: status=%d data=%v msg=%q", status, result.Data, result.Msg)
	}
	status, result = server.call(t, "GET", "/api/student/courses", studentToken, nil)
	if status != 200 {
		t.Fatalf("student list: status=%d msg=%q", status, result.Msg)
	}
	for _, item := range dataAsList(result.Data) {
		if dataAsMap(item)["title"] == "Introduction to Git" {
			t.Fatal("unpublished course still visible to the student")
		}
	}
	status, result = server.call(t, "GET", "/api/teacher/courses", teacherToken, nil)
	if status != 200 {
		t.Fatalf("teacher list: status=%d msg=%q", status, result.Msg)
	}
	var teacherStillOwns bool
	for _, item := range dataAsList(result.Data) {
		row := dataAsMap(item)
		if row["title"] == "Introduction to Git" {
			classIDs := dataAsList(row["classIds"])
			teacherStillOwns = row["published"] == false && len(classIDs) == 1
		}
	}
	if !teacherStillOwns {
		t.Fatalf("teacher courses = %v", result.Data)
	}
	if status, result = server.call(t, "PUT", "/api/system/courses/99999/publish", adminToken, map[string]any{"published": false}); status != 404 {
		t.Fatalf("missing course: status=%d msg=%q", status, result.Msg)
	}
	if status, result = server.call(t, "PUT", fmt.Sprintf("/api/system/courses/%.0f/publish", gitID), adminToken, map[string]any{}); status != 400 {
		t.Fatalf("missing published: status=%d msg=%q", status, result.Msg)
	}

	testutil.MustExec(t, server.database,
		`INSERT INTO ai_usage (user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at)
		 VALUES (?, 'generate_course', 'mock', 2, 3, 5, 'success', ?)`,
		testutil.GetUserID(t, server.database, "student1"), time.Now().UTC(),
	)
	status, result = server.call(t, "GET", "/api/system/usage", adminToken, nil)
	if status != 200 {
		t.Fatalf("usage: status=%d msg=%q", status, result.Msg)
	}
	usageReport := dataAsMap(result.Data)
	if usageReport["accountCount"] != float64(4) || usageReport["totalTokens"] != float64(5) {
		t.Fatalf("usage = %+v", usageReport)
	}
	var studentCalls float64
	for _, item := range dataAsList(usageReport["users"]) {
		row := dataAsMap(item)
		if row["username"] == "student1" {
			studentCalls = row["callCount"].(float64)
			calls := dataAsList(row["calls"])
			if len(calls) != 1 || dataAsMap(calls[0])["scene"] != "generate_course" {
				t.Fatalf("student calls = %v", row["calls"])
			}
		}
		if row["role"] == "ROLE_ADMIN" {
			t.Fatalf("admin in usage list: %v", row)
		}
	}
	if studentCalls != 1 {
		t.Fatalf("student call count = %v", usageReport["users"])
	}
}
