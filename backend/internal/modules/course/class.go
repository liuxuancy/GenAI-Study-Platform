package course

import (
	"context"
	"encoding/json"
	"time"

	"github.com/liuxuancy/Libs/logs"
	"gorm.io/gorm"

	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

// ClassMember 班里的一个人。角色仍看 sys_user.role。
type ClassMember struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// ClassItem 一个班和它的成员。
type ClassItem struct {
	ID      int64         `json:"id"`
	Name    string        `json:"name"`
	Members []ClassMember `json:"members"`
}

// ClassMembersInput 用这份名单替换班里的全部成员。空数组表示清班。
type ClassMembersInput struct {
	UserIDs *[]int64 `json:"userIds"`
}

// RenameClassInput 只改班名。
type RenameClassInput struct {
	Name string `json:"name"`
}

// TeacherClassItem 教师能发布到的班。
type TeacherClassItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// TeacherCourseDetail 教师看到的一门课，含发布到哪些班和气泡树。
type TeacherCourseDetail struct {
	ID           int64                    `json:"id"`
	Title        string                   `json:"title"`
	Description  string                   `json:"description"`
	CourseType   string                   `json:"courseType"`
	Audience     string                   `json:"audience"`
	Language     string                   `json:"language"`
	Published    bool                     `json:"published"`
	ClassIDs     []int64                  `json:"classIds"`
	ChapterCount int                      `json:"chapterCount"`
	HasContent   bool                     `json:"hasContent"`
	Tree         *BubbleNode              `json:"tree"`
	Content      *aicourse.StandardCourse `json:"content"`
}

// PublishClassesInput 发布到这些班。空数组表示撤回：is_publish 改为 0，并删掉该课全部 course_class。
type PublishClassesInput struct {
	ClassIDs *[]int64 `json:"classIds"`
}

// ClassStudentProgress 发布范围内一名学生的课内进度。没有进度行时为 0。
type ClassStudentProgress struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Percent  int    `json:"percent"`
}

// CourseClassProgress 这门课当前发布到的班里的学生。
type CourseClassProgress struct {
	Students []ClassStudentProgress `json:"students"`
}

// GetClassList 全部班级，按 id 排序。
func (service *Service) GetClassList(ctx context.Context) ([]ClassItem, error) {
	var rows []struct {
		ID   int64
		Name string
	}
	err := service.database.WithContext(ctx).Raw(
		"SELECT id, name FROM study_class ORDER BY id",
	).Scan(&rows).Error
	if err != nil {
		logs.CtxError(ctx, "[GetClassList] failed, err: %v", err)
		return nil, err
	}
	classes := make([]ClassItem, 0, len(rows))
	for _, row := range rows {
		classes = append(classes, ClassItem{ID: row.ID, Name: row.Name, Members: []ClassMember{}})
	}
	var members []struct {
		ClassID  int64
		UserID   int64
		Username string
		Nickname string
		Role     string
	}
	err = service.database.WithContext(ctx).Raw(
		`SELECT m.class_id, u.id AS user_id, u.username, u.nickname, u.role
		 FROM study_class_member m
		 JOIN sys_user u ON u.id = m.user_id
		 ORDER BY m.class_id, u.id`,
	).Scan(&members).Error
	if err != nil {
		logs.CtxError(ctx, "[GetClassList] failed, err: %v", err)
		return nil, err
	}
	grouped := map[int64][]ClassMember{}
	for _, member := range members {
		grouped[member.ClassID] = append(grouped[member.ClassID], ClassMember{
			UserID:   member.UserID,
			Username: member.Username,
			Nickname: member.Nickname,
			Role:     member.Role,
		})
	}
	for index := range classes {
		classMembers := grouped[classes[index].ID]
		if classMembers == nil {
			classMembers = []ClassMember{}
		}
		classes[index].Members = classMembers
	}
	logs.CtxInfo(ctx, "[GetClassList] done: len(classes)=%d", len(classes))
	return classes, nil
}

// CreateClass 新建一个还没有成员的班。重名返回 409。
func (service *Service) CreateClass(ctx context.Context, name string) (*ClassItem, error) {
	normalized, err := requireText(name, maxClassNameRunes, "class name is required", "class name is too long")
	if err != nil {
		logs.CtxError(ctx, "[CreateClass] failed, err: %v", err)
		return nil, err
	}
	now := time.Now().UTC()
	result := service.database.WithContext(ctx).Exec(
		"INSERT INTO study_class (name, created_at, updated_at) VALUES (?, ?, ?)",
		normalized, now, now,
	)
	if duplicateKey(result.Error) {
		return nil, errs.Conflict("class name already exists")
	}
	if result.Error != nil {
		logs.CtxError(ctx, "[CreateClass] failed, err: %v", result.Error)
		return nil, result.Error
	}
	output, err := service.classByName(ctx, normalized)
	if err != nil {
		logs.CtxError(ctx, "[CreateClass] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[CreateClass] done: resp=%+v", output)
	return output, nil
}

// RenameClass 只改名称，不动成员。
func (service *Service) RenameClass(ctx context.Context, classID int64, name string) (*ClassItem, error) {
	normalized, err := requireText(name, maxClassNameRunes, "class name is required", "class name is too long")
	if err != nil {
		logs.CtxError(ctx, "[RenameClass] failed, err: %v", err)
		return nil, err
	}
	result := service.database.WithContext(ctx).Exec(
		"UPDATE study_class SET name = ?, updated_at = ? WHERE id = ?",
		normalized, time.Now().UTC(), classID,
	)
	if duplicateKey(result.Error) {
		return nil, errs.Conflict("class name already exists")
	}
	if result.Error != nil {
		logs.CtxError(ctx, "[RenameClass] failed, err: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("class not found")
	}
	output, err := service.classByID(ctx, classID)
	if err != nil {
		logs.CtxError(ctx, "[RenameClass] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[RenameClass] done: class_id=%d resp=%+v", classID, output)
	return output, nil
}

// SetClassMembers 用这份用户名单替换成员。只能放教师和学生。
func (service *Service) SetClassMembers(ctx context.Context, classID int64, userIDs []int64) (*ClassItem, error) {
	ids, err := uniqueIDs(userIDs)
	if err != nil {
		logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", err)
		return nil, err
	}
	database := service.database.WithContext(ctx)
	if _, err := service.classByID(ctx, classID); err != nil {
		logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", err)
		return nil, err
	}
	tx := database.Begin()
	if tx.Error != nil {
		logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", tx.Error)
		return nil, tx.Error
	}
	for _, userID := range ids {
		if err := requireClassUser(tx, userID); err != nil {
			tx.Rollback()
			logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", err)
			return nil, err
		}
	}
	if err := tx.Exec("DELETE FROM study_class_member WHERE class_id = ?", classID).Error; err != nil {
		tx.Rollback()
		logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", err)
		return nil, err
	}
	for _, userID := range ids {
		err := tx.Exec(
			"INSERT INTO study_class_member (class_id, user_id) VALUES (?, ?)",
			classID, userID,
		).Error
		if err != nil {
			tx.Rollback()
			logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", err)
			return nil, err
		}
	}
	if err := tx.Exec(
		"UPDATE study_class SET updated_at = ? WHERE id = ?",
		time.Now().UTC(), classID,
	).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", err)
		return nil, err
	}
	output, err := service.classByID(ctx, classID)
	if err != nil {
		logs.CtxError(ctx, "[SetClassMembers] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[SetClassMembers] done: class_id=%d user_ids=%v len(members)=%d", classID, userIDs, len(output.Members))
	return output, nil
}

func requireClassUser(tx *gorm.DB, userID int64) error {
	var role string
	result := tx.Raw("SELECT role FROM sys_user WHERE id = ?", userID).Scan(&role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.NotFound("user not found")
	}
	if role != access.RoleTeacher && role != access.RoleStudent {
		return errs.BadRequest("only teachers and students can join a class")
	}
	return nil
}

func (service *Service) classByName(ctx context.Context, name string) (*ClassItem, error) {
	var classID int64
	result := service.database.WithContext(ctx).Raw("SELECT id FROM study_class WHERE name = ?", name).Scan(&classID)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("class not found")
	}
	return service.classByID(ctx, classID)
}

func (service *Service) classByID(ctx context.Context, classID int64) (*ClassItem, error) {
	classes, err := service.GetClassList(ctx)
	if err != nil {
		return nil, err
	}
	for index := range classes {
		if classes[index].ID == classID {
			return &classes[index], nil
		}
	}
	return nil, errs.NotFound("class not found")
}

// GetTeacherClasses 当前教师所在的班，发布时只能勾这些。
func (service *Service) GetTeacherClasses(ctx context.Context, principal *access.Principal) ([]TeacherClassItem, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can view classes")
	}
	var items []TeacherClassItem
	err := service.database.WithContext(ctx).Raw(
		`SELECT c.id, c.name
		 FROM study_class c
		 JOIN study_class_member m ON m.class_id = c.id AND m.user_id = ?
		 ORDER BY c.id`,
		principal.UserID,
	).Scan(&items).Error
	if err != nil {
		logs.CtxError(ctx, "[GetTeacherClasses] failed, err: %v", err)
		return nil, err
	}
	if items == nil {
		items = []TeacherClassItem{}
	}
	logs.CtxInfo(ctx, "[GetTeacherClasses] done: len(classes)=%d resp=%+v", len(items), items)
	return items, nil
}

// GetTeacherCourseList 教师只看自己的课。
func (service *Service) GetTeacherCourseList(ctx context.Context, principal *access.Principal) ([]TeacherCourseDetail, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can view courses")
	}
	var rows []storedCourse
	err := service.database.WithContext(ctx).Raw(
		`SELECT id, title, COALESCE(description, '') AS description, COALESCE(course_type, '') AS course_type,
		        COALESCE(audience, '') AS audience, COALESCE(language, '') AS language,
		        COALESCE(content_json, '') AS content_json, is_publish
		 FROM course
		 WHERE owner_id = ?
		 ORDER BY id`,
		principal.UserID,
	).Scan(&rows).Error
	if err != nil {
		logs.CtxError(ctx, "[GetTeacherCourseList] failed, err: %v", err)
		return nil, err
	}
	classIDs, err := service.classIDsForOwner(ctx, principal.UserID)
	if err != nil {
		logs.CtxError(ctx, "[GetTeacherCourseList] failed, err: %v", err)
		return nil, err
	}
	items := make([]TeacherCourseDetail, 0, len(rows))
	for _, row := range rows {
		ids := classIDs[row.ID]
		if ids == nil {
			ids = []int64{}
		}
		items = append(items, TeacherCourseDetail{
			ID:           row.ID,
			Title:        row.Title,
			Description:  row.Description,
			CourseType:   row.CourseType,
			Audience:     row.Audience,
			Language:     shownLanguage(row.Language),
			Published:    row.IsPublish == PublishOn,
			ClassIDs:     ids,
			ChapterCount: courseChapterCount(row.ContentJSON),
			HasContent:   row.ContentJSON != "",
		})
	}
	logs.CtxInfo(ctx, "[GetTeacherCourseList] done: len(courses)=%d", len(items))
	return items, nil
}

// GetTeacherCourse 教师打开自己的课。看不见和没有这门课都是 404。
func (service *Service) GetTeacherCourse(ctx context.Context, principal *access.Principal, courseID int64) (*TeacherCourseDetail, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can view courses")
	}
	var row storedCourse
	result := service.database.WithContext(ctx).Raw(
		`SELECT id, title, COALESCE(description, '') AS description, COALESCE(course_type, '') AS course_type,
		        COALESCE(audience, '') AS audience, COALESCE(language, '') AS language,
		        COALESCE(content_json, '') AS content_json, is_publish
		 FROM course
		 WHERE id = ? AND owner_id = ?`,
		courseID, principal.UserID,
	).Scan(&row)
	if result.Error != nil {
		logs.CtxError(ctx, "[GetTeacherCourse] failed, err: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}
	output, err := service.teacherDetail(ctx, row)
	if err != nil {
		logs.CtxError(ctx, "[GetTeacherCourse] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[GetTeacherCourse] done: id=%d title=%q published=%t class_ids=%v chapter_count=%d has_content=%t", output.ID, output.Title, output.Published, output.ClassIDs, output.ChapterCount, output.HasContent)
	return output, nil
}

// PublishCourseToClasses 把课发到指定班，或在名单为空时整门撤回。
func (service *Service) PublishCourseToClasses(ctx context.Context, principal *access.Principal, courseID int64, classIDs []int64) (*TeacherCourseDetail, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can publish courses")
	}
	ids, err := uniqueIDs(classIDs)
	if err != nil {
		logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", err)
		return nil, err
	}
	database := service.database.WithContext(ctx)
	var ownerID int64
	found := database.Raw(
		"SELECT owner_id FROM course WHERE id = ? AND owner_id = ?",
		courseID, principal.UserID,
	).Scan(&ownerID)
	if found.Error != nil {
		logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", found.Error)
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}

	tx := database.Begin()
	if tx.Error != nil {
		logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", tx.Error)
		return nil, tx.Error
	}
	now := time.Now().UTC()
	publish := PublishOff
	if len(ids) > 0 {
		publish = PublishOn
		for _, classID := range ids {
			if err := teacherOwnsClass(tx, classID, principal.UserID); err != nil {
				tx.Rollback()
				logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", err)
				return nil, err
			}
		}
	}
	if err := tx.Exec("DELETE FROM course_class WHERE course_id = ?", courseID).Error; err != nil {
		tx.Rollback()
		logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", err)
		return nil, err
	}
	for _, classID := range ids {
		err := tx.Exec(
			"INSERT INTO course_class (course_id, class_id) VALUES (?, ?)",
			courseID, classID,
		).Error
		if err != nil {
			tx.Rollback()
			logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", err)
			return nil, err
		}
	}
	if err := tx.Exec(
		"UPDATE course SET is_publish = ?, updated_at = ? WHERE id = ?",
		publish, now, courseID,
	).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", err)
		return nil, err
	}
	output, err := service.GetTeacherCourse(ctx, principal, courseID)
	if err != nil {
		logs.CtxError(ctx, "[PublishCourseToClasses] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[PublishCourseToClasses] done: class_ids=%v id=%d title=%q published=%t class_ids=%v chapter_count=%d has_content=%t", classIDs, output.ID, output.Title, output.Published, output.ClassIDs, output.ChapterCount, output.HasContent)
	return output, nil
}

func teacherOwnsClass(tx *gorm.DB, classID int64, userID int64) error {
	var classCount int64
	if err := tx.Raw("SELECT COUNT(*) FROM study_class WHERE id = ?", classID).Scan(&classCount).Error; err != nil {
		return err
	}
	if classCount == 0 {
		return errs.NotFound("class not found")
	}
	var memberCount int64
	err := tx.Raw(
		"SELECT COUNT(*) FROM study_class_member WHERE class_id = ? AND user_id = ?",
		classID, userID,
	).Scan(&memberCount).Error
	if err != nil {
		return err
	}
	if memberCount == 0 {
		return errs.BadRequest("you can only publish to a class you belong to")
	}
	return nil
}

// GetTeacherCourseProgress 只列出当前发布到的班里的学生，以及他们的进度。
func (service *Service) GetTeacherCourseProgress(ctx context.Context, principal *access.Principal, courseID int64) (*CourseClassProgress, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can view course progress")
	}
	var ownerID int64
	found := service.database.WithContext(ctx).Raw(
		"SELECT owner_id FROM course WHERE id = ? AND owner_id = ?",
		courseID, principal.UserID,
	).Scan(&ownerID)
	if found.Error != nil {
		logs.CtxError(ctx, "[GetTeacherCourseProgress] failed, err: %v", found.Error)
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}
	var students []ClassStudentProgress
	err := service.database.WithContext(ctx).Raw(
		`SELECT DISTINCT u.id AS user_id, u.username, u.nickname, COALESCE(p.percent, 0) AS percent
		 FROM course_class cc
		 JOIN study_class_member m ON m.class_id = cc.class_id
		 JOIN sys_user u ON u.id = m.user_id AND u.role = ?
		 LEFT JOIN student_course_progress p ON p.user_id = u.id AND p.course_id = cc.course_id
		 WHERE cc.course_id = ?
		 ORDER BY u.id`,
		access.RoleStudent, courseID,
	).Scan(&students).Error
	if err != nil {
		logs.CtxError(ctx, "[GetTeacherCourseProgress] failed, err: %v", err)
		return nil, err
	}
	if students == nil {
		students = []ClassStudentProgress{}
	}
	output := &CourseClassProgress{Students: students}
	logs.CtxInfo(ctx, "[GetTeacherCourseProgress] done: course_id=%d len(students)=%d", courseID, len(output.Students))
	return output, nil
}

func (service *Service) teacherDetail(ctx context.Context, row storedCourse) (*TeacherCourseDetail, error) {
	questions, err := service.chapterQuestions(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	tree, err := buildCourseTree(row.Title, row.Description, row.ContentJSON, questions)
	if err != nil {
		return nil, err
	}
	classIDs, err := courseClassIDs(service.database.WithContext(ctx), row.ID)
	if err != nil {
		return nil, err
	}
	var content *aicourse.StandardCourse
	if row.ContentJSON != "" {
		content = &aicourse.StandardCourse{}
		if err := json.Unmarshal([]byte(row.ContentJSON), content); err != nil {
			return nil, errs.Internal(err)
		}
	}
	return &TeacherCourseDetail{
		ID:           row.ID,
		Title:        row.Title,
		Description:  row.Description,
		CourseType:   row.CourseType,
		Audience:     row.Audience,
		Language:     shownLanguage(row.Language),
		Published:    row.IsPublish == PublishOn,
		ClassIDs:     classIDs,
		ChapterCount: courseChapterCount(row.ContentJSON),
		HasContent:   row.ContentJSON != "",
		Tree:         tree,
		Content:      content,
	}, nil
}

func (service *Service) classIDsForOwner(ctx context.Context, ownerID int64) (map[int64][]int64, error) {
	var rows []struct {
		CourseID int64
		ClassID  int64
	}
	err := service.database.WithContext(ctx).Raw(
		`SELECT cc.course_id, cc.class_id
		 FROM course_class cc
		 JOIN course c ON c.id = cc.course_id
		 WHERE c.owner_id = ?
		 ORDER BY cc.course_id, cc.class_id`,
		ownerID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	grouped := map[int64][]int64{}
	for _, row := range rows {
		grouped[row.CourseID] = append(grouped[row.CourseID], row.ClassID)
	}
	return grouped, nil
}

func courseClassIDs(database *gorm.DB, courseID int64) ([]int64, error) {
	var ids []int64
	err := database.Raw(
		"SELECT class_id FROM course_class WHERE course_id = ? ORDER BY class_id",
		courseID,
	).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []int64{}
	}
	return ids, nil
}
