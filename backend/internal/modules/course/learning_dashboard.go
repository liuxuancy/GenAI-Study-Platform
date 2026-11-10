package course

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

// LearningDashboardTask 今日学习计划条目。
type LearningDashboardTask struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Course    string `json:"course"`
	Type      string `json:"type"`
	Completed bool   `json:"completed"`
}

// LearningRecommendedAction 推荐行动。
type LearningRecommendedAction struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// LearningDashboard 学习中心汇总。
type LearningDashboard struct {
	Tasks             []LearningDashboardTask   `json:"tasks"`
	WeeklyMinutes     []int                     `json:"weeklyMinutes"`
	ActiveDays        []int                     `json:"activeDays"`
	Year              int                       `json:"year"`
	Month             int                       `json:"month"`
	RuleSuggestion    string                    `json:"ruleSuggestion"`
	RecommendedAction LearningRecommendedAction `json:"recommendedAction"`
}

// LearningInsightInput 生成 AI 建议。
type LearningInsightInput struct {
	Provider string `json:"provider"`
	Language string `json:"language"`
}

// LearningInsightResult AI 学习建议。
type LearningInsightResult struct {
	Suggestion        string                    `json:"suggestion"`
	RecommendedAction LearningRecommendedAction `json:"recommendedAction"`
	Meta              ai.Meta                   `json:"meta"`
}

type learningEventRow struct {
	EventType       string
	DurationSeconds int
	OccurredAt      time.Time
}

// GetLearningDashboard GET /api/student/learning/dashboard
func (service *Service) GetLearningDashboard(ctx context.Context, principal *access.Principal, year, month int) (*LearningDashboard, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can view learning dashboard")
	}
	now := time.Now().UTC()
	if year <= 0 {
		year = now.Year()
	}
	if month < 1 || month > 12 {
		month = int(now.Month())
	}

	courses, err := service.GetCourseList(ctx, principal)
	if err != nil {
		return nil, err
	}
	questions, err := service.GetQuestionList(ctx, principal)
	if err != nil {
		return nil, err
	}

	tasks := buildLearningTasks(courses, questions)
	weeklyMinutes, err := service.aggregateWeeklyMinutes(ctx, principal.UserID, now)
	if err != nil {
		return nil, err
	}
	activeDays, err := service.aggregateActiveDays(ctx, principal.UserID, year, month)
	if err != nil {
		return nil, err
	}

	continueCourse := pickContinueCourseListItem(courses)
	pendingQuiz := countPendingQuizzes(questions)
	ruleSuggestion := buildRuleSuggestion(courses, continueCourse, pendingQuiz)
	recommended := buildRecommendedAction(continueCourse, pendingQuiz)

	return &LearningDashboard{
		Tasks:             tasks,
		WeeklyMinutes:     weeklyMinutes,
		ActiveDays:        activeDays,
		Year:              year,
		Month:             month,
		RuleSuggestion:    ruleSuggestion,
		RecommendedAction: recommended,
	}, nil
}

func pickContinueCourseListItem(courses []CourseListItem) *CourseListItem {
	if len(courses) == 0 {
		return nil
	}
	for index := range courses {
		course := courses[index]
		if course.Percent > 0 && course.Percent < 100 {
			return &course
		}
	}
	for index := range courses {
		course := courses[index]
		if course.Percent < 100 {
			return &course
		}
	}
	return &courses[0]
}

func countPendingQuizzes(questions []QuestionListItem) int {
	count := 0
	for _, item := range questions {
		if item.IsSubmit {
			continue
		}
		if item.Answered == 0 || item.Answered < item.QuestionCount {
			count++
		}
	}
	return count
}

func buildLearningTasks(courses []CourseListItem, questions []QuestionListItem) []LearningDashboardTask {
	tasks := make([]LearningDashboardTask, 0, 8)

	learningAdded := 0
	for _, course := range courses {
		if course.Percent > 0 && course.Percent < 100 {
			tasks = append(tasks, LearningDashboardTask{
				ID:        fmt.Sprintf("course-%d", course.ID),
				Title:     fmt.Sprintf("继续 %s", course.Title),
				Course:    course.Title,
				Type:      "学习",
				Completed: false,
			})
			learningAdded++
			if learningAdded >= 3 {
				break
			}
		}
	}
	if learningAdded < 3 {
		continueCourse := pickContinueCourseListItem(courses)
		if continueCourse != nil && continueCourse.Percent == 0 {
			tasks = append(tasks, LearningDashboardTask{
				ID:        fmt.Sprintf("course-%d", continueCourse.ID),
				Title:     fmt.Sprintf("开始 %s", continueCourse.Title),
				Course:    continueCourse.Title,
				Type:      "学习",
				Completed: false,
			})
		}
	}

	quizCount := 0
	for _, item := range questions {
		if item.IsSubmit {
			continue
		}
		title := fmt.Sprintf("完成 %s", displayQuizTitle(item))
		if item.Answered > 0 {
			title = fmt.Sprintf("继续 %s", displayQuizTitle(item))
		}
		tasks = append(tasks, LearningDashboardTask{
			ID:        fmt.Sprintf("quiz-%d", item.ID),
			Title:     title,
			Course:    item.CourseTitle,
			Type:      "测验",
			Completed: false,
		})
		quizCount++
		if quizCount >= 5 {
			break
		}
	}

	reviewCount := 0
	for _, item := range questions {
		if !item.IsSubmit || item.Score == nil {
			continue
		}
		if *item.Score >= 70 {
			continue
		}
		tasks = append(tasks, LearningDashboardTask{
			ID:        fmt.Sprintf("review-%d", item.ID),
			Title:     fmt.Sprintf("复习 %s", displayQuizTitle(item)),
			Course:    item.CourseTitle,
			Type:      "复习",
			Completed: *item.Score >= 70,
		})
		reviewCount++
		if reviewCount >= 3 {
			break
		}
	}

	return tasks
}

func displayQuizTitle(item QuestionListItem) string {
	title := strings.TrimSpace(item.Title)
	if item.ChapterName != "" {
		return item.ChapterName + "测验"
	}
	return title
}

func buildRuleSuggestion(courses []CourseListItem, continueCourse *CourseListItem, pendingQuiz int) string {
	if len(courses) == 0 {
		return "你还没有可学习的课程。前往「我的课程」创建一门课，或等待教师发布到班级。"
	}
	parts := make([]string, 0, 2)
	if continueCourse != nil && continueCourse.Percent < 100 {
		parts = append(parts, fmt.Sprintf("建议优先继续「%s」（当前进度 %d%%）", continueCourse.Title, continueCourse.Percent))
	} else if continueCourse != nil && continueCourse.Percent == 100 {
		parts = append(parts, fmt.Sprintf("「%s」已完成，可以在测验中心巩固或探索其他课程", continueCourse.Title))
	}
	if pendingQuiz > 0 {
		parts = append(parts, fmt.Sprintf("测验中心有 %d 项待完成，可与当前章节配合练习", pendingQuiz))
	}
	if len(parts) == 0 {
		return "从「我的课程」选择一门课开始学习，进度与测验将在此汇总。"
	}
	return strings.Join(parts, "；") + "。"
}

func buildRecommendedAction(continueCourse *CourseListItem, pendingQuiz int) LearningRecommendedAction {
	if continueCourse != nil && continueCourse.Percent < 100 {
		return LearningRecommendedAction{
			Title:  "继续当前课程",
			Detail: fmt.Sprintf("进入「%s」，完成下一章节内容。", continueCourse.Title),
		}
	}
	if pendingQuiz > 0 {
		return LearningRecommendedAction{
			Title:  "完成待办测验",
			Detail: "前往测验中心，完成尚未提交的练习。",
		}
	}
	return LearningRecommendedAction{
		Title:  "探索课程库",
		Detail: "在「我的课程」中挑选一门课并开始学习。",
	}
}

func startOfWeekUTC(reference time.Time) time.Time {
	weekday := int(reference.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := reference.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

func (service *Service) aggregateWeeklyMinutes(ctx context.Context, userID int64, reference time.Time) ([]int, error) {
	weekStart := startOfWeekUTC(reference)
	weekEnd := weekStart.AddDate(0, 0, 7)
	var rows []learningEventRow
	err := service.database.WithContext(ctx).Raw(
		`SELECT event_type, duration_seconds, occurred_at
		 FROM student_learning_event
		 WHERE user_id = ? AND occurred_at >= ? AND occurred_at < ?`,
		userID, weekStart, weekEnd,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	minutes := make([]int, 7)
	for _, row := range rows {
		dayIndex := int(row.OccurredAt.Sub(weekStart).Hours()) / 24
		if dayIndex < 0 || dayIndex > 6 {
			continue
		}
		minutes[dayIndex] += effectiveEventMinutes(row.EventType, row.DurationSeconds)
	}
	return minutes, nil
}

func (service *Service) aggregateActiveDays(ctx context.Context, userID int64, year, month int) ([]int, error) {
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)
	var rows []struct {
		Day int
	}
	err := service.database.WithContext(ctx).Raw(
		`SELECT CAST(strftime('%d', occurred_at) AS INTEGER) AS day
		 FROM student_learning_event
		 WHERE user_id = ? AND occurred_at >= ? AND occurred_at < ?
		 GROUP BY day`,
		userID, monthStart, monthEnd,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	days := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.Day > 0 {
			days = append(days, row.Day)
		}
	}
	sort.Ints(days)
	return days, nil
}

type learningInsightPayload struct {
	StudentName      string   `json:"studentName"`
	ContinueCourse   string   `json:"continueCourse,omitempty"`
	ContinuePercent  int      `json:"continuePercent,omitempty"`
	PendingQuizCount int      `json:"pendingQuizCount"`
	CourseSummaries  []string `json:"courseSummaries"`
}

// GenerateLearningInsight POST /api/student/learning/insights
func (service *Service) GenerateLearningInsight(ctx context.Context, principal *access.Principal, input LearningInsightInput) (*LearningInsightResult, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can generate learning insights")
	}
	courses, err := service.GetCourseList(ctx, principal)
	if err != nil {
		return nil, err
	}
	questions, err := service.GetQuestionList(ctx, principal)
	if err != nil {
		return nil, err
	}

	continueCourse := pickContinueCourseListItem(courses)
	payload := learningInsightPayload{
		PendingQuizCount: countPendingQuizzes(questions),
	}
	if continueCourse != nil {
		payload.ContinueCourse = continueCourse.Title
		payload.ContinuePercent = continueCourse.Percent
	}
	summaries := make([]string, 0, len(courses))
	for _, course := range courses {
		if len(summaries) >= 8 {
			break
		}
		summaries = append(summaries, fmt.Sprintf("%s (%d%%)", course.Title, course.Percent))
	}
	payload.CourseSummaries = summaries

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	aiService, err := service.aiForGenerate(input.Provider)
	if err != nil {
		return nil, err
	}
	response, meta, callErr := aiService.GenerateLearningInsight(ctx, string(encoded), input.Language)
	service.recordUsage(ctx, principal.UserID, usage.SceneLearningInsight, meta, callErr)
	if callErr != nil {
		ai.LogError(ctx, callErr, meta, principal.UserID)
		return nil, businessAIError(callErr)
	}

	var parsed struct {
		Suggestion        string                    `json:"suggestion"`
		RecommendedAction LearningRecommendedAction `json:"recommendedAction"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil || strings.TrimSpace(parsed.Suggestion) == "" {
		return &LearningInsightResult{
			Suggestion:        strings.TrimSpace(response),
			RecommendedAction: buildRecommendedAction(continueCourse, payload.PendingQuizCount),
			Meta:              meta,
		}, nil
	}
	if strings.TrimSpace(parsed.RecommendedAction.Title) == "" {
		parsed.RecommendedAction = buildRecommendedAction(continueCourse, payload.PendingQuizCount)
	}
	return &LearningInsightResult{
		Suggestion:        parsed.Suggestion,
		RecommendedAction: parsed.RecommendedAction,
		Meta:              meta,
	}, nil
}

func parseDashboardMonth(queryYear, queryMonth string) (int, int) {
	year, _ := strconv.Atoi(strings.TrimSpace(queryYear))
	month, _ := strconv.Atoi(strings.TrimSpace(queryMonth))
	return year, month
}
