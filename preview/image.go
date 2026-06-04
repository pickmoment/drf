package preview

import "path/filepath"

// RenderImage returns a placeholder message for image files.
func RenderImage(path string) []string {
	name := filepath.Base(path)
	return []string{
		"",
		"  🖼  " + name,
		"",
		"  이미지 미리보기는 지원 예정입니다.",
		"",
	}
}
