// 演示数据：团队共享的展示数据。框架部分来自 demo/*.json，各模块的部分由 DemoProvider 写入。

package seed

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"caveclass/internal/framework/module"
)

// 编译时嵌入程序，不管从哪个目录启动都能读到
//
//go:embed demo/users.json
var demoUsersJSON []byte

// LoadDemo 写入演示数据，整个过程在一个事务里。已存在的数据保持原样。
func LoadDemo(ctx context.Context, database *gorm.DB, modules []module.Module) error {
	var accounts []Account
	if err := json.Unmarshal(demoUsersJSON, &accounts); err != nil {
		return fmt.Errorf("parse demo/users.json: %w", err)
	}
	loader := &demoLoader{ctx: ctx, accounts: accounts, modules: modules}
	return database.WithContext(ctx).Transaction(loader.run)
}

type demoLoader struct {
	ctx      context.Context
	accounts []Account
	modules  []module.Module
}

func (loader *demoLoader) run(tx *gorm.DB) error {
	if err := ensureAccounts(loader.ctx, tx, "demo accounts", loader.accounts); err != nil {
		return fmt.Errorf("demo users: %w", err)
	}
	for _, current := range loader.modules {
		provider, ok := current.(module.DemoProvider)
		if !ok {
			continue
		}
		if err := provider.LoadDemo(loader.ctx, tx); err != nil {
			return fmt.Errorf("demo data of module %s: %w", current.GetName(), err)
		}
	}
	return nil
}
