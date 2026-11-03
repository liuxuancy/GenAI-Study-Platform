package cos

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
)

// OriginalObjectKey 生成课程原始文件对象 Key：courses/{courseId}/originals/{uuid}.{ext}
func OriginalObjectKey(courseID int64, originalFilename string) (string, error) {
	if courseID <= 0 {
		return "", fmt.Errorf("courseID must be positive")
	}
	ext := extensionFromFilename(originalFilename)
	if ext == "" {
		return "", fmt.Errorf("filename must include a safe extension")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("courses/%d/originals/%s.%s", courseID, id.String(), ext), nil
}

func extensionFromFilename(name string) string {
	base := path.Base(strings.TrimSpace(name))
	if base == "" || base == "." {
		return ""
	}
	dot := strings.LastIndex(base, ".")
	if dot <= 0 || dot == len(base)-1 {
		return ""
	}
	ext := strings.ToLower(base[dot+1:])
	if !allowedExtension(ext) {
		return ""
	}
	return ext
}

func allowedExtension(ext string) bool {
	switch ext {
	case "pdf", "ppt", "pptx", "doc", "docx", "xls", "xlsx", "zip", "rar", "7z", "png", "jpg", "jpeg", "gif", "webp", "txt", "md":
		return true
	default:
		return false
	}
}
