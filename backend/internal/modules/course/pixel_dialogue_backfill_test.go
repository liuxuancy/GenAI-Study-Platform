package course

import (
	"context"
	"path/filepath"
	"testing"

	"caveclass/internal/ai"
	"caveclass/internal/ai/llm"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/db"
	"caveclass/internal/usage"
)

type blockingDialogueClient struct {
	started chan struct{}
	release chan struct{}
}

func (client blockingDialogueClient) Complete(ctx context.Context, _ llm.Request) (llm.Response, error) {
	close(client.started)
	select {
	case <-client.release:
		return llm.Response{Text: `{"zh":{"touch":"请别碰我","rest":"休息一下"},"en":{"touch":"Please stop","rest":"I need a rest"}}`}, nil
	case <-ctx.Done():
		return llm.Response{}, ctx.Err()
	}
}

func TestPixelDialogueLeasePreventsDuplicateWork(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Lease avatar"})
	if err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, database, `UPDATE course_pixel_avatar SET dialogue_json = NULL WHERE course_id = ?`, created.ID)
	client := blockingDialogueClient{started: make(chan struct{}), release: make(chan struct{})}
	model, err := ai.New(llm.Config{}, client)
	if err != nil {
		t.Fatal(err)
	}
	service.UseAI(model)
	finished := make(chan error, 1)
	go func() {
		_, err := service.backfillPixelDialogue(context.Background())
		finished <- err
	}()
	<-client.started
	other := NewService(database)
	useMockAI(t, other)
	found, err := other.backfillPixelDialogue(context.Background())
	if found || err != nil {
		close(client.release)
		t.Fatalf("leased row was claimed: found %v error %v", found, err)
	}
	close(client.release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, `SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ?`, 1, student.UserID, usage.SceneGeneratePixelDialogue)
}

func TestPixelDialogueMigrationKeepsLegacyAvatar(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(database) })
	testutil.MustExec(t, database, `CREATE TABLE course_pixel_avatar (
		id integer PRIMARY KEY, course_id integer NOT NULL, role_type text NOT NULL,
		body_type text NOT NULL, hair_color text NOT NULL, hair_style text NOT NULL,
		facial_feature text NOT NULL, accessory text NOT NULL, silhouette_key text NOT NULL,
		personality text NOT NULL, pixel_json text NOT NULL, pixel_json_25 text NOT NULL,
		pixel_json_50 text NOT NULL, pixel_json_75 text NOT NULL, pixel_json_100 text NOT NULL, created_at datetime
	)`)
	testutil.MustExec(t, database, `INSERT INTO course_pixel_avatar VALUES (1, 42, 'scholar', 'slim', 'black', 'short', 'smile', 'none', 'coat', 'calm', 'base', '25', '50', '75', '100', NULL)`)
	if err := db.Migrate(database, &CoursePixelAvatar{}); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, `SELECT COUNT(*) FROM course_pixel_avatar WHERE course_id = 42 AND pixel_json = 'base' AND dialogue_json IS NULL AND dialogue_retry_at = 0 AND dialogue_attempts = 0`, 1)
}
