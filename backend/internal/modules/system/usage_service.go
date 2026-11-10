// 管理员查看模型用量。只读 ai_usage 和 sys_user，不 import course。
// token 是模型用量，不是登录凭证。不返回提示词。

package system

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/liuxuancy/Libs/logs"
	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

type UsageService struct {
	database *gorm.DB
}

// NewUsageService 全局只初始化一次：启动时在路由表里创建。
func NewUsageService(database *gorm.DB) *UsageService {
	return &UsageService{database: database}
}

// UsageCall 一次模型调用。师生明细里的一行。
type UsageCall struct {
	ID               int64     `json:"id"`
	Scene            string    `json:"scene"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"promptTokens"`
	CompletionTokens int       `json:"completionTokens"`
	TotalTokens      int       `json:"totalTokens"`
	CallStatus       string    `json:"callStatus"`
	CreatedAt        time.Time `json:"createdAt"`
}

// UsageUser 一位用户在当前筛选范围内的合计，以及调用明细。
type UsageUser struct {
	UserID           int64       `json:"userId"`
	Username         string      `json:"username"`
	Nickname         string      `json:"nickname"`
	Role             string      `json:"role"`
	RoleName         string      `json:"roleName" gorm:"-"`
	CallCount        int         `json:"callCount"`
	PromptTokens     int         `json:"promptTokens"`
	CompletionTokens int         `json:"completionTokens"`
	TotalTokens      int         `json:"totalTokens"`
	Calls            []UsageCall `json:"calls" gorm:"-"`
}

// UsageReport 系统页用的账号数、筛选后的 token 合计和分页用户用量。
type UsageReport struct {
	AccountCount     int64       `json:"accountCount"`
	CallCount        int         `json:"callCount"`
	PromptTokens     int         `json:"promptTokens"`
	CompletionTokens int         `json:"completionTokens"`
	TotalTokens      int         `json:"totalTokens"`
	Users            []UsageUser `json:"users"`
	Total            int64       `json:"total"`
	Page             int         `json:"page"`
	PageSize         int         `json:"pageSize"`
}

type usageCallRow struct {
	UsageCall
	UserID int64
}

type usageTotals struct {
	CallCount        int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

const countAccountsSQL = `SELECT COUNT(*) FROM sys_user`

type UsageListQuery struct {
	From       string `form:"from" binding:"omitempty,max=10"`
	To         string `form:"to" binding:"omitempty,max=10"`
	User       string `form:"user" binding:"omitempty,max=64"`
	Role       string `form:"role"`
	Scene      string `form:"scene" binding:"omitempty,max=32"`
	Model      string `form:"model" binding:"omitempty,max=64"`
	CallStatus string `form:"call_status" binding:"omitempty,oneof=success failed"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

type usageFilters struct {
	from       *time.Time
	to         *time.Time
	user       string
	userID     int64
	role       string
	scene      string
	model      string
	callStatus string
}

func parseUsageFilters(query UsageListQuery) (usageFilters, error) {
	result := usageFilters{
		user: strings.TrimSpace(query.User), role: strings.TrimSpace(query.Role),
		scene: strings.TrimSpace(query.Scene), model: strings.TrimSpace(query.Model),
		callStatus: strings.TrimSpace(query.CallStatus),
	}
	result.userID = -1
	if parsedID, err := strconv.ParseInt(result.user, 10, 64); err == nil {
		result.userID = parsedID
	}
	if result.role != "" && !access.IsValidRole(result.role) {
		return result, errs.BadRequest("unknown role")
	}
	parseDate := func(value, name string) (*time.Time, error) {
		if value == "" {
			return nil, nil
		}
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return nil, errs.BadRequest(name + " must use YYYY-MM-DD")
		}
		return &parsed, nil
	}
	var err error
	if result.from, err = parseDate(query.From, "from"); err != nil {
		return result, err
	}
	if result.to, err = parseDate(query.To, "to"); err != nil {
		return result, err
	}
	if result.from != nil && result.to != nil && result.from.After(*result.to) {
		return result, errs.BadRequest("from must not be after to")
	}
	return result, nil
}

func applyUsageFilters(database *gorm.DB, filters usageFilters) *gorm.DB {
	if filters.from != nil {
		database = database.Where("a.created_at >= ?", *filters.from)
	}
	if filters.to != nil {
		database = database.Where("a.created_at < ?", filters.to.AddDate(0, 0, 1))
	}
	if filters.user != "" {
		pattern := "%" + escapeLikePattern(filters.user) + "%"
		database = database.Where(
			"(u.id = ? OR u.username LIKE ? ESCAPE '\\' OR u.nickname LIKE ? ESCAPE '\\')",
			filters.userID, pattern, pattern,
		)
	}
	if filters.role != "" {
		database = database.Where("u.role = ?", filters.role)
	}
	if filters.scene != "" {
		database = database.Where("a.scene = ?", filters.scene)
	}
	if filters.model != "" {
		database = database.Where("a.model = ?", filters.model)
	}
	if filters.callStatus != "" {
		database = database.Where("a.call_status = ?", filters.callStatus)
	}
	return database
}

func usageBase(database *gorm.DB, filters usageFilters) *gorm.DB {
	return applyUsageFilters(
		database.Table(usage.Usage{}.TableName()+" AS a").Joins("JOIN sys_user u ON u.id = a.user_id"),
		filters,
	)
}

// GetUsageReport 返回同一组筛选条件下的全站合计，以及分页后的用户聚合和调用明细。
func (service *UsageService) GetUsageReport(ctx context.Context, query UsageListQuery) (*UsageReport, error) {
	database := service.database.WithContext(ctx)
	filters, err := parseUsageFilters(query)
	if err != nil {
		logs.CtxError(ctx, "[GetUsageReport] failed, err: %v", err)
		return nil, err
	}
	page, pageSize := normalizePage(query.Page, query.PageSize)
	report := &UsageReport{Users: []UsageUser{}, Page: page, PageSize: pageSize}

	if err := database.Raw(countAccountsSQL).Scan(&report.AccountCount).Error; err != nil {
		logs.CtxError(ctx, "[GetUsageReport] failed, err: %v", err)
		return nil, err
	}
	var totals usageTotals
	if err := usageBase(database, filters).Select(
		"COUNT(*) AS call_count, COALESCE(SUM(a.prompt_tokens), 0) AS prompt_tokens, " +
			"COALESCE(SUM(a.completion_tokens), 0) AS completion_tokens, COALESCE(SUM(a.total_tokens), 0) AS total_tokens",
	).Scan(&totals).Error; err != nil {
		logs.CtxError(ctx, "[GetUsageReport] failed, err: %v", err)
		return nil, err
	}
	report.CallCount = totals.CallCount
	report.PromptTokens = totals.PromptTokens
	report.CompletionTokens = totals.CompletionTokens
	report.TotalTokens = totals.TotalTokens

	if err := usageBase(database, filters).Distinct("u.id").Count(&report.Total).Error; err != nil {
		logs.CtxError(ctx, "[GetUsageReport] failed, err: %v", err)
		return nil, err
	}
	var users []UsageUser
	err = usageBase(database, filters).
		Select(`u.id AS user_id, u.username, COALESCE(u.nickname, '') AS nickname, u.role,
			COUNT(a.id) AS call_count, COALESCE(SUM(a.prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(a.completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(a.total_tokens), 0) AS total_tokens`).
		Group("u.id, u.username, u.nickname, u.role").
		Order("total_tokens DESC, u.id ASC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&users).Error
	if err != nil {
		logs.CtxError(ctx, "[GetUsageReport] failed, err: %v", err)
		return nil, err
	}
	byUser := make(map[int64]int, len(users))
	userIDs := make([]int64, 0, len(users))
	for index := range users {
		users[index].RoleName = access.GetRoleName(users[index].Role)
		users[index].Calls = []UsageCall{}
		byUser[users[index].UserID] = index
		userIDs = append(userIDs, users[index].UserID)
	}

	var calls []usageCallRow
	if len(userIDs) > 0 {
		err = usageBase(database, filters).
			Select(`a.id, a.user_id, a.scene, COALESCE(a.model, '') AS model,
				a.prompt_tokens, a.completion_tokens, a.total_tokens, a.call_status, a.created_at`).
			Where("a.user_id IN ?", userIDs).
			Order("a.created_at DESC, a.id DESC").
			Scan(&calls).Error
		if err != nil {
			logs.CtxError(ctx, "[GetUsageReport] failed, err: %v", err)
			return nil, err
		}
	}
	for _, call := range calls {
		index, ok := byUser[call.UserID]
		if !ok {
			continue
		}
		users[index].Calls = append(users[index].Calls, call.UsageCall)
	}
	report.Users = users
	// 明细可能很多，只记汇总数字和条数
	logs.CtxInfo(ctx, "[GetUsageReport] done: req=%+v accountCount=%d callCount=%d totalTokens=%d total=%d len(users)=%d len(calls)=%d page=%d pageSize=%d",
		query, report.AccountCount, report.CallCount, report.TotalTokens, report.Total, len(users), len(calls), page, pageSize)
	return report, nil
}
