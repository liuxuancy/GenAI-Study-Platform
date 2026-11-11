package course

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"caveclass/internal/ai"

	"gorm.io/gorm"
)

const (
	pixelAvatarAttempts  = 3
	pixelTraitSampleSize = 64
)

var (
	pixelRoles         = []string{"scholar", "explorer", "inventor", "artist", "ranger", "keeper"}
	pixelBodies        = []string{"petite", "slim", "average", "stocky", "tall", "round"}
	pixelHairColors    = []string{"black", "brown", "blonde", "auburn", "blue", "silver"}
	pixelHairStyles    = []string{"short", "bob", "ponytail", "curly", "bun", "spiky"}
	pixelFaces         = []string{"round-eyes", "sharp-eyes", "dot-eyes", "smile", "focused", "wink"}
	pixelAccessories   = []string{"none", "glasses", "hat", "scarf", "bow", "headphones"}
	pixelSilhouettes   = []string{"round-head", "square-jaw", "hood", "cape", "coat", "overalls"}
	pixelPersonalities = []string{"calm", "cheerful", "curious", "bold", "gentle", "serious"}

	errPixelAppearancesExhausted = errors.New("pixel avatar appearances are exhausted")
	errPixelCandidatesExhausted  = errors.New("no unused pixel avatar appearance found in candidate sample")
	drawPixelTraits              = drawRandomPixelTraits
)

// PixelTraits 是一个像素小人的外观身份。组合在库里唯一。
type PixelTraits struct {
	RoleType      string
	BodyType      string
	HairColor     string
	HairStyle     string
	FacialFeature string
	Accessory     string
	SilhouetteKey string
	Personality   string
}

func (traits PixelTraits) key() string {
	return strings.Join([]string{
		traits.RoleType, traits.BodyType, traits.HairColor, traits.HairStyle,
		traits.FacialFeature, traits.Accessory, traits.SilhouetteKey, traits.Personality,
	}, "\x1f")
}

func (traits PixelTraits) valid() bool {
	return containsString(pixelRoles, traits.RoleType) &&
		containsString(pixelBodies, traits.BodyType) &&
		containsString(pixelHairColors, traits.HairColor) &&
		containsString(pixelHairStyles, traits.HairStyle) &&
		containsString(pixelFaces, traits.FacialFeature) &&
		containsString(pixelAccessories, traits.Accessory) &&
		containsString(pixelSilhouettes, traits.SilhouetteKey) &&
		containsString(pixelPersonalities, traits.Personality)
}

// PixelSprite 是一份可绘制的 16×16 网格。
type PixelSprite struct {
	Palette []string `json:"palette"`
	Pixels  []string `json:"pixels"`
}

// PixelSprites 初始图和四档好感度表情动作。
type PixelSprites struct {
	Base  PixelSprite `json:"base"`
	At25  PixelSprite `json:"25"`
	At50  PixelSprite `json:"50"`
	At75  PixelSprite `json:"75"`
	At100 PixelSprite `json:"100"`
}

// CourseAvatar 学生课程接口里的像素小人。
type CourseAvatar struct {
	RoleType      string            `json:"roleType"`
	BodyType      string            `json:"bodyType"`
	HairColor     string            `json:"hairColor"`
	HairStyle     string            `json:"hairStyle"`
	FacialFeature string            `json:"facialFeature"`
	Accessory     string            `json:"accessory"`
	SilhouetteKey string            `json:"silhouetteKey"`
	Personality   string            `json:"personality"`
	Sprites       PixelSprites      `json:"sprites"`
	Dialogue      *ai.PixelDialogue `json:"dialogue"`
}

func drawRandomPixelTraits(used map[PixelTraits]struct{}) (PixelTraits, error) {
	space := len(pixelRoles) * len(pixelBodies) * len(pixelHairColors) * len(pixelHairStyles) *
		len(pixelFaces) * len(pixelAccessories) * len(pixelSilhouettes) * len(pixelPersonalities)
	if len(used) >= space {
		return PixelTraits{}, errPixelAppearancesExhausted
	}
	candidates := make([]PixelTraits, 0, pixelTraitSampleSize)
	for attempt := 0; attempt < pixelTraitSampleSize; attempt++ {
		candidates = append(candidates, PixelTraits{
			RoleType:      pixelRoles[rand.IntN(len(pixelRoles))],
			BodyType:      pixelBodies[rand.IntN(len(pixelBodies))],
			HairColor:     pixelHairColors[rand.IntN(len(pixelHairColors))],
			HairStyle:     pixelHairStyles[rand.IntN(len(pixelHairStyles))],
			FacialFeature: pixelFaces[rand.IntN(len(pixelFaces))],
			Accessory:     pixelAccessories[rand.IntN(len(pixelAccessories))],
			SilhouetteKey: pixelSilhouettes[rand.IntN(len(pixelSilhouettes))],
			Personality:   pixelPersonalities[rand.IntN(len(pixelPersonalities))],
		})
	}
	return selectPixelTraits(candidates, used, rand.IntN)
}

func (traits PixelTraits) appearanceValues() [7]string {
	return [7]string{traits.RoleType, traits.BodyType, traits.HairColor, traits.HairStyle,
		traits.FacialFeature, traits.Accessory, traits.SilhouetteKey}
}

func selectPixelTraits(candidates []PixelTraits, used map[PixelTraits]struct{}, tieIndex func(int) int) (PixelTraits, error) {
	weights := [7]int{1, 2, 2, 2, 1, 2, 2}
	maxDistance := 0
	for _, weight := range weights {
		maxDistance += weight
	}
	var frequency [7]map[string]int
	for i := range frequency {
		frequency[i] = make(map[string]int)
	}
	for traits := range used {
		for i, value := range traits.appearanceValues() {
			frequency[i][value]++
		}
	}
	bestDistance, bestFrequency, ties := -1, 0, 0
	var best PixelTraits
	for _, candidate := range candidates {
		if _, taken := used[candidate]; taken {
			continue
		}
		values := candidate.appearanceValues()
		minimum := maxDistance
		for existing := range used {
			distance := 0
			for i, value := range existing.appearanceValues() {
				if value != values[i] {
					distance += weights[i]
				}
			}
			if distance < minimum {
				minimum = distance
			}
		}
		popularity := 0
		for i, value := range values {
			popularity += frequency[i][value] * weights[i]
		}
		if minimum > bestDistance || (minimum == bestDistance && popularity < bestFrequency) {
			best, bestDistance, bestFrequency, ties = candidate, minimum, popularity, 1
		} else if minimum == bestDistance && popularity == bestFrequency {
			ties++
			if tieIndex(ties) == 0 {
				best = candidate
			}
		}
	}
	if ties == 0 {
		return PixelTraits{}, errPixelCandidatesExhausted
	}
	return best, nil
}

func (service *Service) loadUsedAppearances(ctx context.Context) (map[PixelTraits]struct{}, error) {
	var rows []PixelTraits
	err := service.database.WithContext(ctx).Raw(
		`SELECT role_type, body_type, hair_color, hair_style, facial_feature, accessory, silhouette_key, personality
		 FROM course_pixel_avatar`,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	used := make(map[PixelTraits]struct{}, len(rows))
	for _, row := range rows {
		used[row] = struct{}{}
	}
	return used, nil
}

func (service *Service) insertPixelAvatar(ctx context.Context, courseID int64, traits PixelTraits, generated ai.PixelSpriteSet) error {
	if !traits.valid() {
		return errors.New("pixel traits are outside the vocabulary")
	}
	base, at25, at50, at75, at100, err := encodePixelSet(generated)
	if err != nil {
		return err
	}
	var dialogue *string
	if generated.Dialogue.Zh.Touch != "" {
		body, err := json.Marshal(generated.Dialogue)
		if err != nil {
			return err
		}
		encoded := string(body)
		dialogue = &encoded
	}
	return service.database.WithContext(ctx).Exec(
		`INSERT INTO course_pixel_avatar (
		   course_id, role_type, body_type, hair_color, hair_style, facial_feature, accessory, silhouette_key, personality,
		   pixel_json, pixel_json_25, pixel_json_50, pixel_json_75, pixel_json_100, dialogue_json, created_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		courseID, traits.RoleType, traits.BodyType, traits.HairColor, traits.HairStyle, traits.FacialFeature,
		traits.Accessory, traits.SilhouetteKey, traits.Personality,
		base, at25, at50, at75, at100, dialogue, time.Now().UTC(),
	).Error
}

func encodePixelSet(generated ai.PixelSpriteSet) (string, string, string, string, string, error) {
	frames := [][]string{generated.Base, generated.At25, generated.At50, generated.At75, generated.At100}
	encoded := make([]string, len(frames))
	for index, pixels := range frames {
		body, err := json.Marshal(PixelSprite{Palette: generated.Palette, Pixels: pixels})
		if err != nil {
			return "", "", "", "", "", err
		}
		encoded[index] = string(body)
	}
	return encoded[0], encoded[1], encoded[2], encoded[3], encoded[4], nil
}

func avatarFromGenerated(traits PixelTraits, generated ai.PixelSpriteSet) *CourseAvatar {
	sprite := func(pixels []string) PixelSprite {
		return PixelSprite{Palette: generated.Palette, Pixels: pixels}
	}
	return &CourseAvatar{
		RoleType:      traits.RoleType,
		BodyType:      traits.BodyType,
		HairColor:     traits.HairColor,
		HairStyle:     traits.HairStyle,
		FacialFeature: traits.FacialFeature,
		Accessory:     traits.Accessory,
		SilhouetteKey: traits.SilhouetteKey,
		Personality:   traits.Personality,
		Dialogue:      &generated.Dialogue,
		Sprites: PixelSprites{
			Base:  sprite(generated.Base),
			At25:  sprite(generated.At25),
			At50:  sprite(generated.At50),
			At75:  sprite(generated.At75),
			At100: sprite(generated.At100),
		},
	}
}

type storedPixelAvatar struct {
	CourseID      int64
	RoleType      string
	BodyType      string
	HairColor     string
	HairStyle     string
	FacialFeature string
	Accessory     string
	SilhouetteKey string
	Personality   string
	PixelJSON     string
	PixelJSON25   string `gorm:"column:pixel_json_25"`
	PixelJSON50   string `gorm:"column:pixel_json_50"`
	PixelJSON75   string `gorm:"column:pixel_json_75"`
	PixelJSON100  string `gorm:"column:pixel_json_100"`
	DialogueJSON  *string
}

func (service *Service) loadAvatarViews(ctx context.Context, courseIDs []int64) (map[int64]*CourseAvatar, error) {
	views := map[int64]*CourseAvatar{}
	if len(courseIDs) == 0 {
		return views, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(courseIDs)), ",")
	args := make([]any, len(courseIDs))
	for index, id := range courseIDs {
		args[index] = id
	}
	var rows []storedPixelAvatar
	err := service.database.WithContext(ctx).Raw(
		`SELECT course_id, role_type, body_type, hair_color, hair_style, facial_feature, accessory, silhouette_key, personality,
		        pixel_json, pixel_json_25, pixel_json_50, pixel_json_75, pixel_json_100, dialogue_json
		 FROM course_pixel_avatar WHERE course_id IN (`+placeholders+`)`,
		args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		view, err := avatarFromStored(row)
		if err != nil {
			return nil, err
		}
		views[row.CourseID] = view
	}
	return views, nil
}

func (service *Service) loadAvatarView(ctx context.Context, courseID int64) (*CourseAvatar, error) {
	views, err := service.loadAvatarViews(ctx, []int64{courseID})
	if err != nil {
		return nil, err
	}
	return views[courseID], nil
}

func avatarFromStored(row storedPixelAvatar) (*CourseAvatar, error) {
	var dialogue *ai.PixelDialogue
	if row.DialogueJSON != nil && *row.DialogueJSON != "" {
		dialogue = &ai.PixelDialogue{}
		if err := json.Unmarshal([]byte(*row.DialogueJSON), dialogue); err != nil {
			return nil, err
		}
	}
	base, err := decodePixelSprite(row.PixelJSON)
	if err != nil {
		return nil, err
	}
	at25, err := decodePixelSprite(row.PixelJSON25)
	if err != nil {
		return nil, err
	}
	at50, err := decodePixelSprite(row.PixelJSON50)
	if err != nil {
		return nil, err
	}
	at75, err := decodePixelSprite(row.PixelJSON75)
	if err != nil {
		return nil, err
	}
	at100, err := decodePixelSprite(row.PixelJSON100)
	if err != nil {
		return nil, err
	}
	return &CourseAvatar{
		RoleType:      row.RoleType,
		BodyType:      row.BodyType,
		HairColor:     row.HairColor,
		HairStyle:     row.HairStyle,
		FacialFeature: row.FacialFeature,
		Accessory:     row.Accessory,
		SilhouetteKey: row.SilhouetteKey,
		Personality:   row.Personality,
		Dialogue:      dialogue,
		Sprites:       PixelSprites{Base: base, At25: at25, At50: at50, At75: at75, At100: at100},
	}, nil
}

func decodePixelSprite(raw string) (PixelSprite, error) {
	var sprite PixelSprite
	if err := json.Unmarshal([]byte(raw), &sprite); err != nil {
		return PixelSprite{}, err
	}
	return sprite, nil
}

func (service *Service) deleteCreatedCourse(ctx context.Context, courseID int64) error {
	return service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM course_pixel_avatar WHERE course_id = ?`, courseID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM student_course_progress WHERE course_id = ?`, courseID).Error; err != nil {
			return err
		}
		return tx.Exec(`DELETE FROM course WHERE id = ?`, courseID).Error
	})
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
