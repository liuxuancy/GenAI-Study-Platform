package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func loadLocalEnv() error {
	for _, path := range []string{".env", "backend/.env"} {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		defer file.Close()
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
				return err
			}
		}
		return scanner.Err()
	}
	return fmt.Errorf("没有找到 .env。请在 backend/.env 写入密钥，且不要提交这个文件")
}
