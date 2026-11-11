package course

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/ai"
	"caveclass/internal/ai/llm"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

func useMockAI(t *testing.T, service *Service) {
	t.Helper()
	t.Setenv("AI_PROVIDER", "mock")
	aiService, err := ai.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	service.UseAI(aiService)
}

func TestSelectPixelTraitsMaximizesNearestAppearanceDistance(t *testing.T) {
	first := PixelTraits{RoleType: "scholar", BodyType: "slim", HairColor: "black", HairStyle: "short", FacialFeature: "smile", Accessory: "hat", SilhouetteKey: "coat", Personality: "calm"}
	second := PixelTraits{RoleType: "artist", BodyType: "round", HairColor: "silver", HairStyle: "curly", FacialFeature: "wink", Accessory: "glasses", SilhouetteKey: "cape", Personality: "bold"}
	personalityOnly := first
	personalityOnly.Personality = "gentle"
	nearSecond := second
	nearSecond.HairStyle = "bob"
	farFromBoth := PixelTraits{RoleType: "explorer", BodyType: "tall", HairColor: "blue", HairStyle: "bun", FacialFeature: "focused", Accessory: "scarf", SilhouetteKey: "hood", Personality: "calm"}
	used := map[PixelTraits]struct{}{first: {}, second: {}}
	selected, err := selectPixelTraits([]PixelTraits{first, personalityOnly, nearSecond, farFromBoth}, used, func(int) int { return 1 })
	if err != nil || selected != farFromBoth {
		t.Fatalf("selected = %+v, error %v", selected, err)
	}
	selected, err = selectPixelTraits([]PixelTraits{personalityOnly, nearSecond}, used, func(int) int { return 1 })
	if err != nil || selected != nearSecond {
		t.Fatalf("personality must not increase appearance distance: selected = %+v, error %v", selected, err)
	}
}

func TestSelectPixelTraitsUsesWeightedDistanceAndFrequency(t *testing.T) {
	first := PixelTraits{RoleType: "scholar", BodyType: "slim", HairColor: "black", HairStyle: "short", FacialFeature: "smile", Accessory: "hat", SilhouetteKey: "coat", Personality: "calm"}
	roleOnly := first
	roleOnly.RoleType = "artist"
	hairOnly := first
	hairOnly.HairStyle = "bob"
	selected, err := selectPixelTraits([]PixelTraits{roleOnly, hairOnly}, map[PixelTraits]struct{}{first: {}}, func(int) int { return 1 })
	if err != nil || selected != hairOnly {
		t.Fatalf("hair should have greater weight than role: %+v, %v", selected, err)
	}
	other := first
	other.Personality = "bold"
	other.RoleType = "artist"
	other.HairStyle = "curly"
	other.HairColor = "silver"
	commonHair := first
	commonHair.HairStyle = "curly"
	rareHair := first
	rareHair.HairStyle = "bun"
	selected, err = selectPixelTraits([]PixelTraits{commonHair, rareHair}, map[PixelTraits]struct{}{first: {}, other: {}}, func(int) int { return 1 })
	if err != nil || selected != rareHair {
		t.Fatalf("equally distant rare attributes should win: %+v, %v", selected, err)
	}
}

func TestSelectPixelTraitsHandlesTiesAndUsedCandidates(t *testing.T) {
	first := PixelTraits{RoleType: "scholar", Personality: "calm"}
	second := PixelTraits{RoleType: "artist", Personality: "bold"}
	selected, err := selectPixelTraits([]PixelTraits{first, second}, nil, func(int) int { return 0 })
	if err != nil || selected != second {
		t.Fatalf("tie selection = %+v, error %v", selected, err)
	}
	_, err = selectPixelTraits([]PixelTraits{first}, map[PixelTraits]struct{}{first: {}}, func(int) int { return 0 })
	if !errors.Is(err, errPixelCandidatesExhausted) {
		t.Fatalf("all used candidate error = %v", err)
	}
	for i := 0; i < 10; i++ {
		selected, err := drawRandomPixelTraits(nil)
		if err != nil || !selected.valid() {
			t.Fatalf("random traits = %+v, error %v", selected, err)
		}
	}
}

func TestCreateStudentCourseStoresUniquePixelAvatar(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")

	first, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Sprite one", Description: "a"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Sprite two"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Avatar == nil || second.Avatar == nil {
		t.Fatal("created course is missing an avatar")
	}
	if first.Avatar.Dialogue == nil || first.Avatar.Dialogue.Zh.Touch == "" || first.Avatar.Dialogue.En.Rest == "" {
		t.Fatal("created course is missing dialogue")
	}
	if first.Avatar.key() == second.Avatar.key() {
		t.Fatalf("avatars share an appearance: %+v", first.Avatar)
	}
	if first.Affection != 0 || second.Affection != 0 {
		t.Fatalf("affection = %d and %d", first.Affection, second.Affection)
	}
	assertDistinctSprites(t, first.Avatar.Sprites)
	assertCount(t, database, "SELECT COUNT(*) FROM course_pixel_avatar WHERE course_id = ?", 1, first.ID)
	assertCount(t, database, "SELECT COUNT(*) FROM student_course_progress WHERE user_id = ? AND course_id = ? AND percent = 0 AND affection = 0", 1, student.UserID, first.ID)
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ? AND call_status = ?", 2, student.UserID, usage.SceneGeneratePixelSprite, usage.StatusSuccess)

	detail, err := service.GetCourse(context.Background(), student, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Avatar == nil || detail.Avatar.RoleType != first.Avatar.RoleType || detail.Affection != 0 || detail.Avatar.Dialogue == nil || *detail.Avatar.Dialogue != *first.Avatar.Dialogue {
		t.Fatalf("detail avatar = %+v affection %d", detail.Avatar, detail.Affection)
	}
}

func TestCreateStudentCourseUsesSelectedProvider(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	sprite, _, err := service.ai.GeneratePixelSprite(context.Background(), ai.PixelSpriteInput{Title: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"palette": sprite.Palette, "base": sprite.Base, "25": sprite.At25,
		"50": sprite.At50, "75": sprite.At75, "100": sprite.At100, "dialogue": sprite.Dialogue,
	})
	if err != nil {
		t.Fatal(err)
	}
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/deepseek" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"message": map[string]string{"content": string(payload)}}},
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "completed", "output": map[string]string{"text": string(payload)},
			})
		}
	}))
	defer server.Close()
	t.Setenv("AI_DEEPSEEK_API_KEY", "deepseek-test-key")
	t.Setenv("AI_TEAMILY_API_KEY", "teamily-test-key")
	t.Setenv("AI_DEEPSEEK_BASE_URL", server.URL+"/deepseek")
	t.Setenv("AI_TEAMILY_BASE_URL", server.URL+"/teamily")

	for _, provider := range []string{"deepseek", "teamily"} {
		created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: provider, Provider: provider})
		if err != nil || created.Avatar == nil {
			t.Fatalf("create using %s: avatar=%v error=%v", provider, created, err)
		}
		if path := <-requests; path != "/"+provider {
			t.Fatalf("provider %s sent request to %s", provider, path)
		}
	}
}

func TestCreateStudentCourseRejectsInvalidOrUnconfiguredProviderBeforeInsert(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	t.Setenv("AI_DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TEAMILY_API_KEY", "")
	t.Setenv("AI_API_KEY", "")
	before := countRows(t, database, "SELECT COUNT(*) FROM course")
	for _, provider := range []string{"invalid", "deepseek", "teamily"} {
		if _, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: provider, Provider: provider}); err == nil {
			t.Fatalf("expected error for %s", provider)
		}
		assertCount(t, database, "SELECT COUNT(*) FROM course", before)
	}
	if _, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "mock", Provider: "mock"}); err != nil {
		t.Fatalf("explicit mock: %v", err)
	}
}

type failedDialogueClient struct{}

func (failedDialogueClient) Complete(context.Context, llm.Request) (llm.Response, error) {
	return llm.Response{}, errors.New("test model failure")
}

type spriteRetryClient struct {
	calls int
	valid string
}

func (client *spriteRetryClient) Complete(_ context.Context, request llm.Request) (llm.Response, error) {
	client.calls++
	text := `{"palette":[]}`
	if client.calls > 1 {
		text = client.valid
	}
	return llm.Response{Text: text, Model: request.Model, Usage: llm.Usage{TotalTokens: 10}}, nil
}

func TestCreateStudentCourseRetriesInvalidSpriteAndRecordsUsage(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	sprite, _, err := service.ai.GeneratePixelSprite(context.Background(), ai.PixelSpriteInput{Title: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"palette": sprite.Palette, "base": sprite.Base, "25": sprite.At25, "50": sprite.At50,
		"75": sprite.At75, "100": sprite.At100, "dialogue": sprite.Dialogue,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &spriteRetryClient{valid: string(payload)}
	aiService, err := ai.New(llm.Config{MaxRetries: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	service.UseAI(aiService)
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Retried sprite"})
	if err != nil || created == nil || created.Avatar == nil || client.calls != 2 {
		t.Fatalf("created=%+v calls=%d err=%v", created, client.calls, err)
	}
	assertCount(t, database, `SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ? AND call_status = ? AND total_tokens = 20`,
		1, student.UserID, usage.SceneGeneratePixelSprite, usage.StatusSuccess)
}

func TestCreateStudentCourseAcceptsIdenticalSpriteFrames(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	sprite, _, err := service.ai.GeneratePixelSprite(context.Background(), ai.PixelSpriteInput{Title: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"palette": sprite.Palette, "base": sprite.Base, "25": sprite.Base, "50": sprite.Base,
		"75": sprite.Base, "100": sprite.Base, "dialogue": sprite.Dialogue,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &spriteRetryClient{calls: 1, valid: string(payload)}
	aiService, err := ai.New(llm.Config{MaxRetries: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	service.UseAI(aiService)
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Identical frames"})
	if err != nil || created == nil || created.Avatar == nil || client.calls != 2 {
		t.Fatalf("created=%+v calls=%d err=%v", created, client.calls, err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course_pixel_avatar WHERE course_id = ?", 1, created.ID)
	assertCount(t, database, `SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ? AND call_status = ? AND total_tokens = 10`,
		1, student.UserID, usage.SceneGeneratePixelSprite, usage.StatusSuccess)
}

type cancellingSpriteClient struct {
	cancel context.CancelFunc
}

func (client cancellingSpriteClient) Complete(ctx context.Context, _ llm.Request) (llm.Response, error) {
	client.cancel()
	return llm.Response{}, ctx.Err()
}

func TestCreateStudentCourseCleansUpAfterCancellation(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	aiService, err := ai.New(llm.Config{MaxRetries: 0}, cancellingSpriteClient{cancel: cancel})
	if err != nil {
		t.Fatal(err)
	}
	service.UseAI(aiService)
	student := studentPrincipal(t, database, "student1")
	before := countRows(t, database, "SELECT COUNT(*) FROM course")
	_, err = service.CreateStudentCourse(ctx, student, CreateCourseInput{Title: "Cancelled course"})
	if err == nil || ctx.Err() == nil {
		t.Fatalf("cancelled creation returned %v, context %v", err, ctx.Err())
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course", before)
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE title = ?", 0, "Cancelled course")
	assertCount(t, database, "SELECT COUNT(*) FROM course_pixel_avatar", 0)
	assertCount(t, database, "SELECT COUNT(*) FROM student_course_progress", 0)
}

func TestCreateStudentCourseCleansUpPartialWrites(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	existing, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Existing course"})
	if err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, database, `CREATE TRIGGER fail_owner_progress BEFORE INSERT ON student_course_progress
		BEGIN SELECT RAISE(FAIL, 'progress failure'); END`)
	_, err = service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Partial course"})
	if err == nil {
		t.Fatal("expected progress insertion failure")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE title = ?", 0, "Partial course")
	assertCount(t, database, "SELECT COUNT(*) FROM course_pixel_avatar", 1)
	assertCount(t, database, "SELECT COUNT(*) FROM student_course_progress", 1)
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE id = ?", 1, existing.ID)
}

func TestCreateStudentCourseReportsCleanupFailure(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	testutil.MustExec(t, database, `CREATE TRIGGER fail_owner_progress BEFORE INSERT ON student_course_progress
		BEGIN SELECT RAISE(FAIL, 'progress failure'); END`)
	testutil.MustExec(t, database, `CREATE TRIGGER fail_course_cleanup BEFORE DELETE ON course
		BEGIN SELECT RAISE(FAIL, 'cleanup failure'); END`)
	_, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Cleanup failure"})
	if !errors.Is(err, errs.ErrInternal) || !strings.Contains(err.Error(), "cleanup failure") || !strings.Contains(err.Error(), "progress failure") {
		t.Fatalf("cleanup error = %v", err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE title = ?", 1, "Cleanup failure")
	assertCount(t, database, "SELECT COUNT(*) FROM course_pixel_avatar", 1)
}

func TestPixelDialogueBackfillRetryAndPersistence(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Legacy avatar"})
	if err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, database, `UPDATE course_pixel_avatar SET dialogue_json = NULL WHERE course_id = ?`, created.ID)
	before, err := service.loadAvatarView(context.Background(), created.ID)
	if err != nil || before.Dialogue != nil {
		t.Fatalf("legacy avatar: %+v %v", before, err)
	}
	failed, err := ai.New(llm.Config{MaxRetries: 0}, failedDialogueClient{})
	if err != nil {
		t.Fatal(err)
	}
	service.UseAI(failed)
	found, err := service.backfillPixelDialogue(context.Background())
	if !found || err == nil {
		t.Fatalf("failed backfill: found %v error %v", found, err)
	}
	assertCount(t, database, `SELECT COUNT(*) FROM course_pixel_avatar WHERE course_id = ? AND dialogue_json IS NULL AND dialogue_attempts = 1 AND dialogue_retry_at > 0`, 1, created.ID)
	assertCount(t, database, `SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ? AND call_status = ?`, 1, student.UserID, usage.SceneGeneratePixelDialogue, usage.StatusFailed)
	useMockAI(t, service)
	found, err = service.backfillPixelDialogue(context.Background())
	if found || err != nil {
		t.Fatalf("retried before deadline: found %v error %v", found, err)
	}
	testutil.MustExec(t, database, `UPDATE course_pixel_avatar SET dialogue_retry_at = 0 WHERE course_id = ?`, created.ID)
	found, err = service.backfillPixelDialogue(context.Background())
	if !found || err != nil {
		t.Fatalf("successful backfill: found %v error %v", found, err)
	}
	after, err := service.loadAvatarView(context.Background(), created.ID)
	if err != nil || after.Dialogue == nil || after.Dialogue.Zh.Rest == "" || after.Dialogue.En.Touch == "" {
		t.Fatalf("backfilled avatar: %+v %v", after, err)
	}
	if before.Personality != after.Personality || before.Sprites.Base.Pixels[0] != after.Sprites.Base.Pixels[0] {
		t.Fatal("backfill changed the character")
	}
	assertCount(t, database, `SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ? AND call_status = ?`, 1, student.UserID, usage.SceneGeneratePixelDialogue, usage.StatusSuccess)
	found, err = service.backfillPixelDialogue(context.Background())
	if found || err != nil {
		t.Fatalf("completed avatar claimed again: found %v error %v", found, err)
	}
}
func TestCreateStudentCourseRollsBackWithoutAvatar(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	before := countRows(t, database, "SELECT COUNT(*) FROM course")
	_, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "No model"})
	if !errors.Is(err, errs.ErrInternal) {
		t.Fatalf("create without AI: %v", err)
	}
	if got := countRows(t, database, "SELECT COUNT(*) FROM course"); got != before {
		t.Fatalf("course count = %d, want %d", got, before)
	}

	useMockAI(t, service)
	previous := drawPixelTraits
	drawPixelTraits = func(map[PixelTraits]struct{}) (PixelTraits, error) {
		return PixelTraits{}, errPixelAppearancesExhausted
	}
	t.Cleanup(func() { drawPixelTraits = previous })
	_, err = service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Exhausted"})
	if !errors.Is(err, errs.ErrConflict) {
		t.Fatalf("exhausted appearances: %v", err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE title = ?", 0, "Exhausted")
}

func TestCreateStudentCourseRetriesDuplicateAppearance(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	useMockAI(t, service)
	teacher := teacherPrincipal(t, database, "teacher1")
	holder, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Appearance holder"})
	if err != nil {
		t.Fatal(err)
	}
	first := PixelTraits{
		RoleType: pixelRoles[0], BodyType: pixelBodies[0], HairColor: pixelHairColors[0], HairStyle: pixelHairStyles[0],
		FacialFeature: pixelFaces[0], Accessory: pixelAccessories[0], SilhouetteKey: pixelSilhouettes[0], Personality: pixelPersonalities[0],
	}
	second := first
	second.RoleType = pixelRoles[1]
	if err := service.insertPixelAvatar(context.Background(), holder.ID, first, samplePixelSet()); err != nil {
		t.Fatal(err)
	}

	previous := drawPixelTraits
	draws := 0
	drawPixelTraits = func(map[PixelTraits]struct{}) (PixelTraits, error) {
		draws++
		if draws == 1 {
			return first, nil
		}
		return second, nil
	}
	t.Cleanup(func() { drawPixelTraits = previous })

	student := studentPrincipal(t, database, "student1")
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Retried sprite"})
	if err != nil {
		t.Fatal(err)
	}
	if draws != 2 || created.Avatar == nil || created.Avatar.RoleType != pixelRoles[1] {
		t.Fatalf("draws = %d avatar = %+v", draws, created.Avatar)
	}
}

func TestAffectionFollowsProgressAndQuizAndAdjustment(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	useMockAI(t, service)
	student := studentPrincipal(t, database, "student1")
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Affection course"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.AdjustAffection(context.Background(), student.UserID, created.ID, 10); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, database, student.UserID, created.ID, 10)

	if _, err := service.UpdateCourseProgress(context.Background(), student, created.ID, 80); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, database, student.UserID, created.ID, 74)

	now := time.Now().UTC()
	testutil.MustExec(t, database,
		`INSERT INTO course_question (title, course_id, creator_id, question_count, is_publish, created_at, updated_at)
		 VALUES (?, ?, ?, 2, 0, ?, ?)`,
		"Affection quiz", created.ID, student.UserID, now, now,
	)
	var questionID int64
	if err := database.Raw(`SELECT id FROM course_question WHERE title = ?`, "Affection quiz").Scan(&questionID).Error; err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"one", "two"} {
		testutil.MustExec(t, database,
			`INSERT INTO course_question_content
			 (question_id, content, options, correct_answer, explanation, question_type, sort_order, score, created_at, updated_at)
			 VALUES (?, ?, 'A|B', 'A', 'because', 'single_choice', 1, 1, ?, ?)`,
			questionID, content, now, now,
		)
	}
	var contentID int64
	if err := database.Raw(`SELECT id FROM course_question_content WHERE question_id = ? ORDER BY id LIMIT 1`, questionID).Scan(&contentID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.AnswerQuestion(context.Background(), student, questionID, contentID, "A", false); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, database, student.UserID, created.ID, 84)

	if err := service.AdjustAffection(context.Background(), student.UserID, created.ID, 1000); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, database, student.UserID, created.ID, 100)
	if err := service.AdjustAffection(context.Background(), student.UserID, created.ID, -500); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, database, student.UserID, created.ID, 0)

	listed, err := service.GetCourseList(context.Background(), student)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range listed {
		if item.ID == created.ID {
			found = item.Affection == 0 && item.Avatar != nil && item.Avatar.Sprites.Base.Pixels != nil
		}
	}
	if !found {
		t.Fatalf("list missing affection course: %+v", listed)
	}
}

func TestBackfillAffectionOnce(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	studentID := testutil.GetUserID(t, database, "student1")
	courseID := courseIDByTitle(t, database, "Asking on a line")
	testutil.MustExec(t, database,
		`INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, 80, 0)`,
		studentID, courseID,
	)
	if err := backfillAffection(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, database, studentID, courseID, 64)
	testutil.MustExec(t, database,
		`UPDATE student_course_progress SET affection = 55 WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	)
	if err := backfillAffection(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, database, studentID, courseID, 55)
}

func (avatar *CourseAvatar) key() string {
	return PixelTraits{
		RoleType: avatar.RoleType, BodyType: avatar.BodyType, HairColor: avatar.HairColor, HairStyle: avatar.HairStyle,
		FacialFeature: avatar.FacialFeature, Accessory: avatar.Accessory, SilhouetteKey: avatar.SilhouetteKey, Personality: avatar.Personality,
	}.key()
}

func assertDistinctSprites(t *testing.T, sprites PixelSprites) {
	t.Helper()
	frames := [][]string{sprites.Base.Pixels, sprites.At25.Pixels, sprites.At50.Pixels, sprites.At75.Pixels, sprites.At100.Pixels}
	for left := 0; left < len(frames); left++ {
		if len(frames[left]) != 16 {
			t.Fatalf("frame %d rows = %d", left, len(frames[left]))
		}
		for right := left + 1; right < len(frames); right++ {
			same := true
			for index := range frames[left] {
				if frames[left][index] != frames[right][index] {
					same = false
					break
				}
			}
			if same {
				t.Fatalf("frames %d and %d are identical", left, right)
			}
		}
	}
}

func samplePixelSet() ai.PixelSpriteSet {
	row := "0000000000000000"
	grid := make([]string, 16)
	for index := range grid {
		grid[index] = row
	}
	other := append([]string(nil), grid...)
	other[0] = "1111111111111111"
	third := append([]string(nil), grid...)
	third[1] = "1111111111111111"
	fourth := append([]string(nil), grid...)
	fourth[2] = "1111111111111111"
	fifth := append([]string(nil), grid...)
	fifth[3] = "1111111111111111"
	return ai.PixelSpriteSet{
		Palette: []string{"#111111", "#eeeeee"},
		Base:    grid, At25: other, At50: third, At75: fourth, At100: fifth,
	}
}

func countRows(t *testing.T, database *gorm.DB, query string) int64 {
	t.Helper()
	var got int64
	if err := database.Raw(query).Scan(&got).Error; err != nil {
		t.Fatal(err)
	}
	return got
}

func assertAffection(t *testing.T, database *gorm.DB, userID, courseID int64, want int) {
	t.Helper()
	var got int
	err := database.Raw(
		`SELECT affection FROM student_course_progress WHERE user_id = ? AND course_id = ?`,
		userID, courseID,
	).Scan(&got).Error
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("affection = %d, want %d", got, want)
	}
}
