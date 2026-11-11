package course

import (
	"path/filepath"
	"testing"

	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/db"
)

func TestEnsurePreAutoMigrateAddsAffectionOnLegacyTable(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(database) })

	testutil.MustExec(t, database, `
		CREATE TABLE student_course_progress (
			user_id INTEGER NOT NULL,
			course_id INTEGER NOT NULL,
			percent INTEGER NOT NULL,
			PRIMARY KEY (user_id, course_id)
		)`)
	testutil.MustExec(t, database, `INSERT INTO student_course_progress (user_id, course_id, percent) VALUES (1, 2, 40)`)

	if err := EnsurePreAutoMigrate(database); err != nil {
		t.Fatalf("EnsurePreAutoMigrate: %v", err)
	}
	if err := db.Migrate(database, &StudentCourseProgress{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	var affection int
	if err := database.Raw(
		`SELECT affection FROM student_course_progress WHERE user_id = 1 AND course_id = 2`,
	).Scan(&affection).Error; err != nil {
		t.Fatalf("query affection: %v", err)
	}
	if affection != 0 {
		t.Fatalf("affection = %d, want 0 before backfill", affection)
	}
}
