package course

import (
	"bytes"
	"mime"
	"path"
	"strings"

	"caveclass/internal/platform/cos"
	"caveclass/internal/platform/errs"
)

var archiveExtensions = map[string]struct{}{
	"zip": {}, "rar": {}, "7z": {},
}

func maxBytesForExtension(ext string, limits cos.Limits) int64 {
	if _, ok := archiveExtensions[ext]; ok {
		return limits.ArchiveMaxBytes
	}
	return limits.DocMaxBytes
}

func normalizeResourceScope(scopeType, chapterKey, sectionKey string) (string, string, string, error) {
	scopeType = strings.ToLower(strings.TrimSpace(scopeType))
	chapterKey = strings.TrimSpace(chapterKey)
	sectionKey = strings.TrimSpace(sectionKey)
	switch scopeType {
	case "", ResourceScopeCourse:
		return ResourceScopeCourse, "", "", nil
	case ResourceScopeChapter:
		if chapterKey == "" {
			return "", "", "", errs.BadRequest("chapterKey is required for chapter scope")
		}
		return ResourceScopeChapter, chapterKey, "", nil
	case ResourceScopeSection:
		if chapterKey == "" || sectionKey == "" {
			return "", "", "", errs.BadRequest("chapterKey and sectionKey are required for section scope")
		}
		return ResourceScopeSection, chapterKey, sectionKey, nil
	default:
		return "", "", "", errs.BadRequest("scopeType must be course, chapter, or section")
	}
}

func sniffContentType(filename string, head []byte) string {
	if len(head) > 512 {
		head = head[:512]
	}
	detected := mime.TypeByExtension(path.Ext(filename))
	if detected == "" {
		detected = httpDetectContentType(head)
	}
	return detected
}

func httpDetectContentType(head []byte) string {
	if len(head) == 0 {
		return "application/octet-stream"
	}
	return strings.Split(httpDetect(head), ";")[0]
}

func httpDetect(b []byte) string {
	// avoid importing net/http in hot path duplication — use bytes prefix checks
	if len(b) >= 4 && bytes.Equal(b[:4], []byte{0x25, 0x50, 0x44, 0x46}) {
		return "application/pdf"
	}
	if len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png"
	}
	if len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF {
		return "image/jpeg"
	}
	if len(b) >= 2 && b[0] == 0x50 && b[1] == 0x4B {
		return "application/zip"
	}
	return "application/octet-stream"
}

func validateResourceFile(filename string, size int64, head []byte, limits cos.Limits) (ext, mimeType string, err error) {
	ext = strings.ToLower(strings.TrimPrefix(path.Ext(strings.TrimSpace(filename)), "."))
	if ext == "" {
		return "", "", errs.BadRequest("filename must include an allowed extension")
	}
	objectKey, keyErr := cos.OriginalObjectKey(1, "x."+ext)
	if keyErr != nil {
		return "", "", errs.BadRequest("file type is not allowed")
	}
	_ = objectKey
	if size <= 0 {
		return "", "", errs.BadRequest("file is empty")
	}
	max := maxBytesForExtension(ext, limits)
	if size > max {
		return "", "", errs.BadRequest("file exceeds size limit")
	}
	mimeType = sniffContentType(filename, head)
	if !mimeMatchesExtension(ext, mimeType) {
		return "", "", errs.BadRequest("file content does not match extension")
	}
	return ext, mimeType, nil
}

func mimeMatchesExtension(ext, mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch ext {
	case "pdf":
		return strings.Contains(mimeType, "pdf") || mimeType == "application/octet-stream"
	case "png", "jpg", "jpeg", "gif", "webp":
		return strings.HasPrefix(mimeType, "image/") || mimeType == "application/octet-stream"
	case "zip", "rar", "7z":
		return strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "octet-stream") ||
			strings.Contains(mimeType, "x-rar") || strings.Contains(mimeType, "7z")
	case "ppt", "pptx", "doc", "docx", "xls", "xlsx":
		return strings.Contains(mimeType, "officedocument") || strings.Contains(mimeType, "msword") ||
			strings.Contains(mimeType, "ms-excel") || strings.Contains(mimeType, "ms-powerpoint") ||
			strings.Contains(mimeType, "octet-stream") || strings.Contains(mimeType, "zip")
	case "txt", "md":
		return strings.HasPrefix(mimeType, "text/") || mimeType == "application/octet-stream"
	default:
		return false
	}
}

func previewKindForResource(row *CourseResource) string {
	ext := strings.ToLower(row.FileExtension)
	switch ext {
	case "pdf":
		return "pdf"
	case "png", "jpg", "jpeg", "gif", "webp":
		return "image"
	case "ppt", "pptx", "doc", "docx", "xls", "xlsx":
		return "office"
	default:
		return "unsupported"
	}
}

func initialPreviewStatus(ext string) string {
	switch previewKindForResource(&CourseResource{FileExtension: ext}) {
	case "pdf", "image":
		return ResourcePreviewReady
	default:
		return ResourcePreviewUnsupported
	}
}
