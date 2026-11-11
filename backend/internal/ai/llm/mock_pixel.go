package llm

import (
	"encoding/json"
	"hash/fnv"
	"strings"
)

var (
	mockHairColors  = []string{"#1b1b1f", "#6b3e26", "#e6c36a", "#8d3b2a", "#3d6bff", "#c5c9d1"}
	mockClothColors = []string{"#355c7d", "#3e7a4c", "#b85c38", "#6b4c9a", "#2f6f94", "#8a3d4b"}
)

func mockPixelSpriteText(prompt string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(prompt))
	hash := int(hasher.Sum32())
	payload := map[string]any{
		"palette": []string{
			mockHairColors[hash%len(mockHairColors)],
			"#f3c7a5",
			mockClothColors[(hash/7)%len(mockClothColors)],
			"#f7f1e8",
			"#ef5678",
		},
		"base":     mockPixelFrame(hash, poseStanding),
		"25":       mockPixelFrame(hash, poseWave),
		"50":       mockPixelFrame(hash, poseBook),
		"75":       mockPixelFrame(hash, poseCheer),
		"100":      mockPixelFrame(hash, poseJump),
		"dialogue": mockPixelDialogue(prompt),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return `{"palette":["#111111","#f3c7a5"],"base":[],"25":[],"50":[],"75":[],"100":[]}`
	}
	return string(encoded)
}

func mockPixelDialogue(prompt string) map[string]any {
	touchZh, restZh := "请别再碰我啦。", "走累了，休息一下。"
	touchEn, restEn := "Please stop touching me.", "I'm tired; I'll rest a little."
	switch {
	case strings.Contains(prompt, "cheerful"):
		touchZh, restZh = "嘿，别再碰我啦！", "走累啦，歇一歇再出发！"
		touchEn, restEn = "Hey, no more poking!", "I'm tired; a little break, then off we go!"
	case strings.Contains(prompt, "curious"):
		touchZh, restZh = "咦，别再碰我啦。", "走累了，歇会儿再探索。"
		touchEn, restEn = "Oh, please stop poking me.", "I'm tired; I'll rest before exploring."
	case strings.Contains(prompt, "bold"):
		touchZh, restZh = "别再碰我！", "走累了，休息一下再出发！"
		touchEn, restEn = "No more poking!", "I'm tired; a short rest, then onward!"
	case strings.Contains(prompt, "gentle"):
		touchZh, restZh = "请不要再碰我，好吗？", "有点累了，让我休息一下。"
		touchEn, restEn = "Please don't touch me again, okay?", "I'm a little tired; let me rest."
	case strings.Contains(prompt, "serious"):
		touchZh, restZh = "请停止触碰。", "走累了，需要休息一下。"
		touchEn, restEn = "Please stop touching me.", "I'm tired and need a short rest."
	}
	return map[string]any{
		"zh": map[string]string{"touch": touchZh, "rest": restZh},
		"en": map[string]string{"touch": touchEn, "rest": restEn},
	}
}

func mockPixelDialogueText(prompt string) string {
	body, err := json.Marshal(mockPixelDialogue(prompt))
	if err != nil {
		return "{}"
	}
	return string(body)
}

const (
	poseStanding = iota
	poseWave
	poseBook
	poseCheer
	poseJump
)

func mockPixelFrame(hash int, pose int) []string {
	var grid [16][16]byte
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			grid[y][x] = '.'
		}
	}
	paintPixelBody(&grid, hash)
	switch pose {
	case poseWave:
		grid[8][5] = '.'
		for y := 3; y <= 7; y++ {
			for x := 3; x <= 5; x++ {
				grid[y][x] = '1'
			}
		}
	case poseBook:
		for y := 8; y <= 11; y++ {
			for x := 10; x <= 13; x++ {
				grid[y][x] = '3'
			}
		}
	case poseCheer:
		grid[8][5], grid[8][10] = '.', '.'
		for y := 3; y <= 7; y++ {
			for _, x := range []int{4, 5, 10, 11} {
				grid[y][x] = '1'
			}
		}
	case poseJump:
		for y := 1; y < 15; y++ {
			grid[y] = grid[y+1]
		}
		for x := range grid[15] {
			grid[15][x] = '.'
		}
		grid[11][5], grid[11][10] = '2', '2'
		grid[12][5], grid[12][10] = '2', '2'
		for _, x := range []int{0, 13} {
			grid[3][x], grid[3][x+2] = '4', '4'
			grid[4][x], grid[4][x+1], grid[4][x+2] = '4', '4', '4'
			grid[5][x+1] = '4'
		}
	}
	rows := make([]string, 16)
	for y := 0; y < 16; y++ {
		rows[y] = string(grid[y][:])
	}
	return rows
}

func paintPixelBody(grid *[16][16]byte, hash int) {
	for y := 2; y <= 4; y++ {
		for x := 6; x <= 9; x++ {
			grid[y][x] = '0'
		}
	}
	grid[3][6+(hash%3)] = '3'
	for y := 5; y <= 7; y++ {
		for x := 6; x <= 9; x++ {
			grid[y][x] = '1'
		}
	}
	grid[6][7] = '0'
	grid[6][8] = '0'
	for y := 8; y <= 11; y++ {
		for x := 6; x <= 9; x++ {
			grid[y][x] = '2'
		}
	}
	if hash%2 == 0 {
		grid[9][7] = '3'
	}
	grid[12][6] = '2'
	grid[12][7] = '2'
	grid[12][8] = '2'
	grid[12][9] = '2'
	grid[13][6] = '2'
	grid[13][9] = '2'
	grid[8][5] = '1'
	grid[8][10] = '1'
}
