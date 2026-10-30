// 管理员查看模型用量。只读 ai_usage 和 sys_user，不 import course。
// token 是模型用量，不是登录凭证。不返回提示词。

package system

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
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

// UsageUser 一位教师或学生的合计，以及他的调用明细。
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

// UsageReport 系统页用的账号数和全站 token，加上每位师生的用量。
// 全站合计包含所有角色的调用。名单只列教师和学生。
type UsageReport struct {
	AccountCount     int64       `json:"accountCount"`
	CallCount        int         `json:"callCount"`
	PromptTokens     int         `json:"promptTokens"`
	CompletionTokens int         `json:"completionTokens"`
	TotalTokens      int         `json:"totalTokens"`
	Users            []UsageUser `json:"users"`
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

func usageTotalsSQL() string {
	return fmt.Sprintf(`
	SELECT COUNT(*) AS call_count,
	       COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
	       COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
	       COALESCE(SUM(total_tokens), 0) AS total_tokens
	FROM %s`, usage.Usage{}.TableName())
}

func usageUsersSQL() string {
	return fmt.Sprintf(`
	SELECT u.id AS user_id, u.username, COALESCE(u.nickname, '') AS nickname, u.role,
	       COUNT(a.id) AS call_count,
	       COALESCE(SUM(a.prompt_tokens), 0) AS prompt_tokens,
	       COALESCE(SUM(a.completion_tokens), 0) AS completion_tokens,
	       COALESCE(SUM(a.total_tokens), 0) AS total_tokens
	FROM sys_user u
	LEFT JOIN %s a ON a.user_id = u.id
	WHERE u.role IN (?, ?)
	GROUP BY u.id
	ORDER BY total_tokens DESC, u.id ASC`, usage.Usage{}.TableName())
}

func usageCallsSQL() string {
	return fmt.Sprintf(`
	SELECT a.id, a.user_id, a.scene, COALESCE(a.model, '') AS model,
	       a.prompt_tokens, a.completion_tokens, a.total_tokens, a.call_status, a.created_at
	FROM %s a
	JOIN sys_user u ON u.id = a.user_id
	WHERE u.role IN (?, ?)
	ORDER BY a.created_at DESC, a.id DESC`, usage.Usage{}.TableName())
}

// GetUsageReport 返回账号数、全站 token 合计，以及每位师生的次数、三类 token、场景和时间。
func (service *UsageService) GetUsageReport(ctx context.Context) (*UsageReport, error) {
	database := service.database.WithContext(ctx)
	report := &UsageReport{Users: []UsageUser{}}

	if err := database.Raw(countAccountsSQL).Scan(&report.AccountCount).Error; err != nil {
		return nil, err
	}
	var totals usageTotals
	if err := database.Raw(usageTotalsSQL()).Scan(&totals).Error; err != nil {
		return nil, err
	}
	report.CallCount = totals.CallCount
	report.PromptTokens = totals.PromptTokens
	report.CompletionTokens = totals.CompletionTokens
	report.TotalTokens = totals.TotalTokens

	var users []UsageUser
	err := database.Raw(usageUsersSQL(), access.RoleTeacher, access.RoleStudent).Scan(&users).Error
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []UsageUser{}
	}
	byUser := make(map[int64]int, len(users))
	for index := range users {
		users[index].RoleName = access.GetRoleName(users[index].Role)
		users[index].Calls = []UsageCall{}
		byUser[users[index].UserID] = index
	}

	var calls []usageCallRow
	err = database.Raw(usageCallsSQL(), access.RoleTeacher, access.RoleStudent).Scan(&calls).Error
	if err != nil {
		return nil, err
	}
	for _, call := range calls {
		index, ok := byUser[call.UserID]
		if !ok {
			continue
		}
		users[index].Calls = append(users[index].Calls, call.UsageCall)
	}
	report.Users = users
	return report, nil
}
