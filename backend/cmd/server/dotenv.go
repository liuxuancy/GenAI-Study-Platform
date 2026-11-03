package main

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv 把本地 .env 填进进程环境。已有环境变量不覆盖，不记录键值。文件不存在则跳过。
func loadDotEnv() error {
	for _, path := range []string{".env", "backend/.env"} {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), `"'`)
			if key == "" || os.Getenv(key) != "" {
				continue
			}
			if err := os.Setenv(key, value); err != nil {
				file.Close()
				return err
			}
		}
		err = scanner.Err()
		file.Close()
		return err
	}
	return nil
}
