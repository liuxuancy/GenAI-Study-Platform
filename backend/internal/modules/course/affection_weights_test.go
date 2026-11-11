package course

import (
	"context"
	"testing"
	"time"

	"caveclass/internal/framework/testutil"
)

func TestAffectionBaseWeights(t *testing.T) {
	for _, test := range []struct {
		course, quiz, want int
	}{
		{0, 0, 0}, {100, 0, 80}, {0, 100, 20},
		{100, 100, 100}, {50, 50, 50}, {1, 1, 1},
		{1, 0, 0}, {0, 4, 0}, {63, 0, 50},
	} {
		if got := affectionBase(test.course, test.quiz); got != test.want {
			t.Errorf("affectionBase(%d, %d) = %d, want %d", test.course, test.quiz, got, test.want)
		}
	}
}

func TestMigrateAffectionWeights(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t, Module{})
	student := studentPrincipal(t, db, "student1")
	first := Course{Title: "Weight course progress", OwnerID: student.UserID}
	second := Course{Title: "Weight quiz progress", OwnerID: student.UserID}
	for _, c := range []*Course{&first, &second} {
		if err := db.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	question := CourseQuestion{Title: "Weight quiz", CourseID: second.ID, CreatorID: student.UserID, QuestionCount: 1}
	if err := db.Create(&question).Error; err != nil {
		t.Fatal(err)
	}
	content := CourseQuestionContent{QuestionID: question.ID, Content: "Question", Options: "A|B", CorrectAnswer: "A", Explanation: "Answer"}
	if err := db.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&StudentQuestionProgress{UserID: student.UserID, QuestionID: question.ID, ContentID: content.ID, UserOption: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []StudentCourseProgress{
		{UserID: student.UserID, CourseID: first.ID, Percent: 80, Affection: 50},
		{UserID: student.UserID, CourseID: second.ID, Percent: 20, Affection: 70},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	testutil.MustExec(t, db, `INSERT INTO migration_log (kind, ref_key, detail, created_at) VALUES (?, '', '', ?)`, affectionBackfillKind, time.Now().UTC())

	for i := 0; i < 2; i++ {
		if err := migrateAffectionWeights(ctx, db); err != nil {
			t.Fatal(err)
		}
		assertAffection(t, db, student.UserID, first.ID, 74)
		assertAffection(t, db, student.UserID, second.ID, 46)
	}
	assertCount(t, db, `SELECT COUNT(*) FROM migration_log WHERE kind = ?`, 1, affectionWeightsKind)
	result, err := NewService(db).SyncAchievements(ctx, student)
	if err != nil || len(result.Unlocked) != 1 || result.Unlocked[0].Key != achievementAffection50 || len(result.Notify) != 1 {
		t.Fatalf("achievement sync = %+v, error %v", result, err)
	}
}

func TestMigrateAffectionWeightsBeforeFirstBackfill(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t, Module{})
	student := studentPrincipal(t, db, "student1")
	c := Course{Title: "Unmigrated affection", OwnerID: student.UserID}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&StudentCourseProgress{UserID: student.UserID, CourseID: c.ID, Percent: 80, Affection: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateAffectionWeights(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := backfillAffection(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := migrateAffectionWeights(ctx, db); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, db, student.UserID, c.ID, 64)
}

func TestMigrateAffectionWeightsRollsBackOnFailure(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t, Module{})
	student := studentPrincipal(t, db, "student1")
	c := Course{Title: "Migration failure", OwnerID: student.UserID}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&StudentCourseProgress{UserID: student.UserID, CourseID: c.ID, Percent: 80, Affection: 40}).Error; err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, db, `INSERT INTO migration_log (kind, ref_key, detail, created_at) VALUES (?, '', '', ?)`, affectionBackfillKind, time.Now().UTC())
	testutil.MustExec(t, db, `CREATE TRIGGER reject_weight_marker BEFORE INSERT ON migration_log
		WHEN NEW.kind = 'affection_weights_80_20'
		BEGIN SELECT RAISE(ABORT, 'marker failed'); END`)
	if err := migrateAffectionWeights(ctx, db); err == nil {
		t.Fatal("expected migration failure")
	}
	assertAffection(t, db, student.UserID, c.ID, 40)
	assertCount(t, db, `SELECT COUNT(*) FROM migration_log WHERE kind = ?`, 0, affectionWeightsKind)
}

func TestSeedProgressAroundWeightMigration(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t, Module{})
	student := studentPrincipal(t, db, "student1")
	before := Course{Title: "Seed before migration", OwnerID: student.UserID}
	after := Course{Title: "Seed after migration", OwnerID: student.UserID}
	for _, c := range []*Course{&before, &after} {
		if err := db.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	testutil.MustExec(t, db, `INSERT INTO migration_log (kind, ref_key, detail, created_at) VALUES (?, '', '', ?)`, affectionBackfillKind, time.Now().UTC())
	if err := ensureProgress(db, student.UserID, before.ID, 80); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, db, student.UserID, before.ID, 40)
	if err := migrateAffectionWeights(ctx, db); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, db, student.UserID, before.ID, 64)
	if err := ensureProgress(db, student.UserID, after.ID, 80); err != nil {
		t.Fatal(err)
	}
	assertAffection(t, db, student.UserID, after.ID, 64)
}
