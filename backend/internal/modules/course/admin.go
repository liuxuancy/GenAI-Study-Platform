// 管理员看全部课程，并可下架或恢复。
// 下架只改 is_publish。course_class、正文和进度都留着，创建者自己仍能看见。
// 恢复时把 is_publish 设回 1，原来发到哪些班仍然有效。

package course

import (
	"context"
	"time"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

// AdminCourseItem 一门课在管理员列表里的样子。
type AdminCourseItem struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Published     bool      `json:"published" gorm:"-"`
	IsPublish     int       `json:"-"`
	OwnerID       int64     `json:"ownerId"`
	OwnerUsername string    `json:"ownerUsername"`
	OwnerNickname string    `json:"ownerNickname"`
	OwnerRole     string    `json:"ownerRole"`
	OwnerRoleName string    `json:"ownerRoleName" gorm:"-"`
	ClassCount    int       `json:"classCount"`
	CreatedAt     time.Time `json:"createdAt"`
}

// AdminCourseList 全部课程，以及已放出的门数。已放出指 is_publish = 1。
type AdminCourseList struct {
	Items          []AdminCourseItem `json:"items"`
	Total          int               `json:"total"`
	PublishedCount int               `json:"publishedCount"`
}

// SetPublishInput 下架或恢复。false 对学生隐藏，true 恢复。
type SetPublishInput struct {
	Published *bool `json:"published" binding:"required"`
}

const adminCourseSelectSQL = `
	SELECT c.id, c.title, COALESCE(c.description, '') AS description, c.is_publish,
	       c.owner_id, u.username AS owner_username, COALESCE(u.nickname, '') AS owner_nickname,
	       u.role AS owner_role, c.created_at,
	       (SELECT COUNT(*) FROM course_class cc WHERE cc.course_id = c.id) AS class_count
	FROM course c
	JOIN sys_user u ON u.id = c.owner_id`

// GetAdminCourseList 管理员不走学生可见性，直接列出全部课程。新的在前。
func (service *Service) GetAdminCourseList(ctx context.Context) (*AdminCourseList, error) {
	var rows []AdminCourseItem
	err := service.database.WithContext(ctx).Raw(adminCourseSelectSQL + ` ORDER BY c.created_at DESC, c.id DESC`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return adminCourseList(rows), nil
}

// SetAdminCoursePublished 下架或恢复一门课。没有这门课返回 404。
// 不删除 course_class，这样恢复后仍发到原来的班。
func (service *Service) SetAdminCoursePublished(ctx context.Context, courseID int64, published bool) (*AdminCourseItem, error) {
	database := service.database.WithContext(ctx)
	var count int64
	if err := database.Raw("SELECT COUNT(*) FROM course WHERE id = ?", courseID).Scan(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errs.NotFound("course not found")
	}
	publish := PublishOff
	if published {
		publish = PublishOn
	}
	err := database.Exec(
		"UPDATE course SET is_publish = ?, updated_at = ? WHERE id = ?",
		publish, time.Now().UTC(), courseID,
	).Error
	if err != nil {
		return nil, err
	}
	var rows []AdminCourseItem
	err = database.Raw(adminCourseSelectSQL+` WHERE c.id = ?`, courseID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errs.NotFound("course not found")
	}
	item := fillAdminCourse(rows[0])
	return &item, nil
}

func adminCourseList(rows []AdminCourseItem) *AdminCourseList {
	if rows == nil {
		rows = []AdminCourseItem{}
	}
	published := 0
	for index := range rows {
		rows[index] = fillAdminCourse(rows[index])
		if rows[index].Published {
			published++
		}
	}
	return &AdminCourseList{Items: rows, Total: len(rows), PublishedCount: published}
}

func fillAdminCourse(item AdminCourseItem) AdminCourseItem {
	item.Published = item.IsPublish == PublishOn
	item.OwnerRoleName = access.GetRoleName(item.OwnerRole)
	return item
}
