package course

import (
	"encoding/json"
	"sort"
	"strings"

	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/platform/errs"
)

// BubbleNode 是课程详情里的气泡。没有 content_json 时整棵树为 null，不再用示例树。
type BubbleNode struct {
	Key          string       `json:"key,omitempty"`
	Label        string       `json:"label"`
	Note         string       `json:"note"`
	Kind         string       `json:"kind,omitempty"`
	ActivityType string       `json:"activityType,omitempty"`
	QuestionID   int64        `json:"questionId,omitempty"`
	Children     []BubbleNode `json:"children,omitempty"`
}

func buildCourseTree(title string, description string, contentJSON string, questionIDs map[string]int64) (*BubbleNode, error) {
	if strings.TrimSpace(contentJSON) == "" {
		return nil, nil
	}
	var parsed aicourse.StandardCourse
	if err := json.Unmarshal([]byte(contentJSON), &parsed); err != nil {
		return nil, errs.Internal(err)
	}
	note := description
	if note == "" {
		note = parsed.Description
	}
	label := title
	if label == "" {
		label = parsed.Title
	}

	chapters := append([]aicourse.Chapter(nil), parsed.Chapters...)
	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Order < chapters[j].Order
	})
	children := make([]BubbleNode, 0, len(chapters))
	for _, chapter := range chapters {
		activities := append([]aicourse.Activity(nil), chapter.Activities...)
		sort.SliceStable(activities, func(i, j int) bool {
			return activities[i].Order < activities[j].Order
		})
		activityNodes := make([]BubbleNode, 0, len(activities))
		for _, activity := range activities {
			activityNodes = append(activityNodes, BubbleNode{
				Key:          activity.Key,
				Label:        activity.Title,
				Note:         activityNote(activity),
				Kind:         "activity",
				ActivityType: activity.Type,
			})
		}
		children = append(children, BubbleNode{
			Key:        chapter.Key,
			Label:      chapter.Title,
			Note:       chapter.Summary,
			Kind:       "chapter",
			QuestionID: questionIDs[chapter.Key],
			Children:   activityNodes,
		})
	}
	return &BubbleNode{
		Label:    label,
		Note:     note,
		Kind:     "course",
		Children: children,
	}, nil
}

func activityNote(activity aicourse.Activity) string {
	switch activity.Type {
	case aicourse.ActivityTypeRichText:
		var payload aicourse.RichTextPayload
		if json.Unmarshal(activity.Payload, &payload) != nil {
			return ""
		}
		return payload.Markdown
	case aicourse.ActivityTypeCoding:
		var payload aicourse.CodingPayload
		if json.Unmarshal(activity.Payload, &payload) != nil {
			return ""
		}
		return payload.PromptMarkdown
	case aicourse.ActivityTypeMockInterview:
		var payload aicourse.MockInterviewPayload
		if json.Unmarshal(activity.Payload, &payload) != nil {
			return ""
		}
		return payload.ScenarioMarkdown
	case aicourse.ActivityTypeQuizGame:
		var payload aicourse.QuizGamePayload
		if json.Unmarshal(activity.Payload, &payload) != nil {
			return ""
		}
		return payload.InstructionsMarkdown
	default:
		return ""
	}
}
