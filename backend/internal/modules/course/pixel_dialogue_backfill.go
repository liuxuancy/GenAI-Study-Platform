package course

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/liuxuancy/Libs/logs"
	"caveclass/internal/usage"
)

const (
	dialogueLeaseDuration = 3 * time.Minute
	dialogueCallTimeout   = 2 * time.Minute
	dialoguePollInterval  = 10 * time.Second
)

type dialogueJob struct {
	CourseID    int64
	Personality string
	OwnerID     int64
	Attempts    int
}

// RunPixelDialogueBackfill processes missing dialogues without blocking course reads.
func (service *Service) RunPixelDialogueBackfill(ctx context.Context) {
	for ctx.Err() == nil {
		found, err := service.backfillPixelDialogue(ctx)
		if err != nil && ctx.Err() == nil {
			logs.CtxError(ctx, "pixel dialogue backfill: %v", err)
		}
		if found && err == nil {
			continue
		}
		timer := time.NewTimer(dialoguePollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (service *Service) backfillPixelDialogue(ctx context.Context) (bool, error) {
	if service.ai == nil {
		return false, fmt.Errorf("pixel dialogue backfill: AI service unavailable")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return false, err
	}
	token := hex.EncodeToString(random[:])
	now := time.Now().Unix()
	var job dialogueJob
	claim := service.database.WithContext(ctx).Raw(
		`UPDATE course_pixel_avatar SET dialogue_lease = ?, dialogue_retry_at = ?
		 WHERE id = (
		   SELECT a.id FROM course_pixel_avatar a JOIN course c ON c.id = a.course_id
		   WHERE (a.dialogue_json IS NULL OR a.dialogue_json = '') AND a.dialogue_retry_at <= ?
		   ORDER BY a.dialogue_retry_at, a.id LIMIT 1
		 )
		 AND (dialogue_json IS NULL OR dialogue_json = '') AND dialogue_retry_at <= ?
		 RETURNING course_id, personality, dialogue_attempts AS attempts`,
		token, now+int64(dialogueLeaseDuration.Seconds()), now, now,
	).Scan(&job)
	if claim.Error != nil {
		return false, claim.Error
	}
	if claim.RowsAffected == 0 {
		return false, nil
	}
	if err := service.database.WithContext(ctx).Raw(`SELECT owner_id FROM course WHERE id = ?`, job.CourseID).Scan(&job.OwnerID).Error; err != nil {
		return true, err
	}
	callCtx, cancel := context.WithTimeout(ctx, dialogueCallTimeout)
	dialogue, meta, callErr := service.ai.GeneratePixelDialogue(callCtx, job.Personality)
	cancel()
	service.recordUsage(context.WithoutCancel(ctx), job.OwnerID, usage.SceneGeneratePixelDialogue, meta, callErr)
	if callErr != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		delay := time.Minute * time.Duration(1<<min(job.Attempts, 6))
		result := service.database.WithContext(ctx).Exec(
			`UPDATE course_pixel_avatar SET dialogue_lease = '', dialogue_retry_at = ?, dialogue_attempts = dialogue_attempts + 1
			 WHERE course_id = ? AND dialogue_lease = ? AND (dialogue_json IS NULL OR dialogue_json = '')`,
			time.Now().Add(delay).Unix(), job.CourseID, token,
		)
		if result.Error != nil {
			return true, result.Error
		}
		return true, fmt.Errorf("course %d dialogue generation failed: %w", job.CourseID, callErr)
	}
	body, err := json.Marshal(dialogue)
	if err != nil {
		return true, err
	}
	result := service.database.WithContext(ctx).Exec(
		`UPDATE course_pixel_avatar SET dialogue_json = ?, dialogue_lease = '', dialogue_retry_at = 0
		 WHERE course_id = ? AND dialogue_lease = ? AND (dialogue_json IS NULL OR dialogue_json = '')`,
		string(body), job.CourseID, token,
	)
	if result.Error != nil {
		return true, result.Error
	}
	if result.RowsAffected != 1 {
		return true, fmt.Errorf("course %d dialogue lease lost", job.CourseID)
	}
	return true, nil
}
