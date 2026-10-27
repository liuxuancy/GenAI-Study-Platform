package course

import (
	"fmt"
	"sort"
)

const (
	// DropReasonIllegalActivityType 表示 activity type 不在四种合法枚举中，已被丢弃。
	DropReasonIllegalActivityType = "illegal_activity_type"
	// DropReasonEmptyChapter 表示章节没有 activity，或 activity 全部被丢弃后章节已空。
	DropReasonEmptyChapter = "empty_chapter"
)

// DroppedItem 记录 Normalize 丢掉的章节或 activity。Path 是整理前的 JSON 路径。
type DroppedItem struct {
	Path   string
	Key    string
	Reason string
}

// NormalizeReport 是 Normalize 的丢弃清单。Normalize 本身不返回 error。
type NormalizeReport struct {
	DroppedActivities []DroppedItem
	DroppedChapters   []DroppedItem
}

// Normalize 就地整理课程，调用方可以再跑 Validate：
//  1. 丢弃非法 type 的 activity
//  2. 丢弃因此没有任何 activity 的章节
//  3. 按原 order 稳定排序，并把章节、activity 的 order 重写成从 1 开始的连续序号
//
// 不补 schemaVersion、estimatedMinutes 等默认值，也不检查 payload。
func (course *StandardCourse) Normalize() NormalizeReport {
	if course == nil {
		return NormalizeReport{}
	}
	report := NormalizeReport{}
	keptChapters := make([]Chapter, 0, len(course.Chapters))
	for chapterIndex, chapter := range course.Chapters {
		keptActivities, droppedActivities := normalizeActivities(chapter.Activities, fmt.Sprintf("chapters[%d]", chapterIndex))
		report.DroppedActivities = append(report.DroppedActivities, droppedActivities...)
		if len(keptActivities) == 0 {
			report.DroppedChapters = append(report.DroppedChapters, DroppedItem{
				Path:   fmt.Sprintf("chapters[%d]", chapterIndex),
				Key:    chapter.Key,
				Reason: DropReasonEmptyChapter,
			})
			continue
		}
		chapter.Activities = keptActivities
		keptChapters = append(keptChapters, chapter)
	}

	sort.SliceStable(keptChapters, func(i, j int) bool {
		return keptChapters[i].Order < keptChapters[j].Order
	})
	for chapterIndex := range keptChapters {
		keptChapters[chapterIndex].Order = chapterIndex + 1
	}
	course.Chapters = keptChapters
	return report
}

// Normalize 整理单个章节片段：丢掉非法 activity，并把本章和 activity 的 order 重写成从 1 开始。
// 章节本身不会因为变空而被删掉，留给 Validate 拒绝。
func (chapter *Chapter) Normalize() NormalizeReport {
	if chapter == nil {
		return NormalizeReport{}
	}
	keptActivities, droppedActivities := normalizeActivities(chapter.Activities, "")
	chapter.Activities = keptActivities
	chapter.Order = 1
	return NormalizeReport{DroppedActivities: droppedActivities}
}

func normalizeActivities(activities []Activity, pathPrefix string) ([]Activity, []DroppedItem) {
	kept := make([]Activity, 0, len(activities))
	var dropped []DroppedItem
	for activityIndex, activity := range activities {
		if validActivityType(activity.Type) {
			kept = append(kept, activity)
			continue
		}
		path := fmt.Sprintf("activities[%d]", activityIndex)
		if pathPrefix != "" {
			path = pathPrefix + "." + path
		}
		dropped = append(dropped, DroppedItem{
			Path:   path,
			Key:    activity.Key,
			Reason: DropReasonIllegalActivityType,
		})
	}
	sort.SliceStable(kept, func(i, j int) bool {
		return kept[i].Order < kept[j].Order
	})
	for activityIndex := range kept {
		kept[activityIndex].Order = activityIndex + 1
	}
	return kept, dropped
}
