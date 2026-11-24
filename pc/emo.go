package main

// 이모티콘 동작
//   - 새 동작 6가지: 박수 · 화이팅 · 사랑해 · 흥! · 신나 · 갸우뚱 (tools/compose.py pose_emo)
//   - 원래 스티커 동작(놀람 · 뒹굴 · 저요 · 인사)과 함께 우클릭 → 이모티콘 메뉴에서 바로 보여 줌
//   - 평소에도: 쓰다듬으면 가끔 사랑해, 마구 연타하면 흥!, 화이팅·사랑해 대사엔 그 동작, 기분 좋으면 가끔 춤

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed emo/*.gif
var emoGifFS embed.FS

// GIF 파일 이름 (tools/emo_gif.py 와 같음)
var emoGifNames = map[string]string{
	"clap": "박수", "cheer": "화이팅", "heart": "사랑해", "angry": "흥", "dance": "신나는춤", "huh": "갸우뚱",
	"st_surprise": "깜짝", "st_me": "저요", "st_lounge": "뒹굴뒹굴", "st_bow": "꾸벅",
	"laugh": "신나", "shy": "부끄", "wave": "안녕", "sleep": "쿨쿨", "st_cry": "엉엉",
	"v": "브이", "thumb": "엄지척", "sing": "노래", "no": "안돼요", "flower": "꽃선물",
	"yum": "잘먹었어요", "what": "뭐해", "cute": "꽃받침",
}

const idEmoGif = 1449

var emoGifDirOverride = "" // 테스트용

// 이모티콘 GIF 를 사진 폴더에 저장 (저장한 폴더, 개수)
func saveEmoGifs() (string, int, error) {
	dir := emoGifDirOverride
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Pictures", "우땅이 이모티콘")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}
	ents, _ := emoGifFS.ReadDir("emo")
	n := 0
	for _, e := range ents {
		b, err := emoGifFS.ReadFile("emo/" + e.Name())
		if err != nil {
			continue
		}
		seq := strings.TrimSuffix(e.Name(), ".gif")
		name := emoGifNames[seq]
		if name == "" {
			name = seq
		}
		if os.WriteFile(filepath.Join(dir, "우땅이_"+name+".gif"), b, 0o644) == nil {
			n++
		}
	}
	return dir, n, nil
}

func onSaveEmoGifs() {
	dir, n, err := saveEmoGifs()
	if err != nil || n == 0 {
		say("앗, 저장을 못 했어…", 3*time.Second)
		return
	}
	S.GifSaved = true
	saveState()
	playEmo("cheer", false)
	say(fmt.Sprintf("이모티콘 %d개를 저장했어!\n카톡에 보내 봐~", n), 4*time.Second)
	toast("우땅이 이모티콘", fmt.Sprintf("사진 폴더 '우땅이 이모티콘'에 GIF %d개를 저장했어요", n))
	exec.Command("explorer", dir).Start()
}

type emoT struct {
	seq, name string
	fps       float64
	sec       float64
	lines     func(n string) []string
}

var emos = []emoT{
	{"clap", "박수 짝짝", 10, 2.6, func(n string) []string {
		return []string{"짝짝짝! 최고야!", n + " 잘했어! 짝짝~", "박수 짝짝!"}
	}},
	{"cheer", "화이팅!", 9, 2.6, func(n string) []string {
		return []string{n + " 화이팅!", "아자아자 화이팅!", "할 수 있어! 화이팅!"}
	}},
	{"heart", "사랑해", 7, 3.2, func(n string) []string {
		return []string{n + " 사랑해 ♥", "이만큼 좋아해! ♥", "두근두근… ♥"}
	}},
	{"angry", "흥!", 9, 2.6, func(n string) []string { return []string{"흥! 삐졌어!", "뿌우… 나 화났어!", "흥칫뿡!"} }},
	{"dance", "신나는 춤", 9, 3.6, func(n string) []string {
		return []string{"룰루랄라~ 신난다!", "우땅우땅 춤이다~", "둠칫 둠칫!"}
	}},
	{"huh", "갸우뚱", 6, 2.8, func(n string) []string {
		return []string{"음…? 그게 뭐야?", "갸우뚱…?", "잘 모르겠어~ 헤헤"}
	}},
	{"v", "브이~", 8, 2.6, func(n string) []string {
		return []string{"브이~! 찰칵!", n + " 나 귀엽지? 브이!", "브이브이~"}
	}},
	{"thumb", "엄지척", 8, 2.6, func(n string) []string { return []string{"최고야! 엄지척!", n + " 최고!", "완전 짱이야!"} }},
	{"sing", "노래하기", 8, 3.6, func(n string) []string {
		return []string{"랄랄라~ ♪", "우땅이 노래 들어볼래? ♬", "바나나 바나나~ ♪"}
	}},
	{"no", "안 돼요!", 10, 2.4, func(n string) []string { return []string{"안 돼요!", "그건 안 돼~!", "절대 안 돼요!"} }},
	{"flower", "꽃 선물", 7, 3.2, func(n string) []string {
		return []string{n + " 이거 선물이야!", "꽃 받아줘~", "예쁜 꽃 줄게!"}
	}},
	{"yum", "잘 먹었어요", 8, 3, func(n string) []string {
		return []string{"잘 먹었어요! 배 통통~", "바나나 최고! 잘 먹었다~"}
	}},
	{"what", "뭐해?", 8, 2.8, func(n string) []string { return []string{n + " 뭐해?", "뭐해~? 나랑 놀자!", "빼꼼! 뭐해?"} }},
	{"cute", "꽃받침", 7, 3, func(n string) []string {
		return []string{"나 귀엽지? 헤헤", "꽃받침~ 뿅!", "귀여움 발사!"}
	}},
	{"st_surprise", "깜짝 놀라기", 9, 1.8, func(n string) []string { return []string{"깜짝이야!", "헉!"} }},
	{"st_me", "저요 저요", 9, 2.2, func(n string) []string { return []string{"저요! 저요!", "나 여기 있어!"} }},
	{"st_lounge", "뒹굴뒹굴", 4, 5, func(n string) []string { return []string{"뒹굴뒹굴~ 편하다", "아무것도 하기 싫다~"} }},
	{"st_bow", "꾸벅 인사", 5, 2.2, func(n string) []string { return []string{"안녕하세요! 꾸벅", n + " 반가워! 꾸벅"} }},
}

const idEmo = 1450 // ~ 1450+len(emos)

func emoOf(seq string) *emoT {
	for i := range emos {
		if emos[i].seq == seq {
			return &emos[i]
		}
	}
	return nil
}

// 새로 만든 이모티콘 동작인지 (그림 · 다음 동작 고르기에서 씀)
func isNewEmo(m string) bool {
	switch m {
	case "clap", "cheer", "heart", "angry", "dance", "huh", "v", "thumb", "sing", "no", "flower", "yum", "what", "cute":
		return true
	}
	return false
}

// 이모티콘 동작 보여 주기 (talk = 대사도 같이)
func playEmo(seq string, talk bool) bool {
	e := emoOf(seq)
	if e == nil || mode == "drag" || mode == "fall" || q != 0 || busy() || hiddenFS {
		return false
	}
	setMode(seq, time.Duration(e.sec*float64(time.Second)))
	if talk {
		say(pick(e.lines(voc())), 2500*time.Millisecond)
	}
	return true
}

// 우클릭 메뉴 → 이모티콘
func appendEmoMenu(m uintptr) {
	em, _, _ := pCreatePopupMenu.Call()
	for i, e := range emos {
		if i == 14 {
			appendMenu(em, MF_SEPARATOR, 0, "")
		}
		appendMenu(em, MF_STRING, uintptr(idEmo+i), e.name)
	}
	appendMenu(em, MF_SEPARATOR, 0, "")
	appendMenu(em, MF_STRING, idEmoGif, "이모티콘 GIF로 저장하기…")
	appendMenu(m, MF_POPUP, em, "이모티콘")
}

func onEmoCmd(cmd uintptr) bool {
	if cmd == idEmoGif {
		onSaveEmoGifs()
		return true
	}
	i := int(cmd) - idEmo
	if i < 0 || i >= len(emos) {
		return false
	}
	if q != 0 {
		comeDown()
		say("내려가서 보여 줄게~ 다시 골라 줘!", 2500*time.Millisecond)
		return true
	}
	if !playEmo(emos[i].seq, true) {
		say("지금은 못 해~ 조금 이따!", 2500*time.Millisecond)
	}
	return true
}

// ── 평소 반응 ──

var clickTimes []time.Time

// 클릭 기록: 3초 안에 5번 넘게 누르면 true (흥!)
func clickSpam(now time.Time) bool {
	keep := clickTimes[:0]
	for _, t := range clickTimes {
		if now.Sub(t) < 3*time.Second {
			keep = append(keep, t)
		}
	}
	clickTimes = append(keep, now)
	if len(clickTimes) >= 5 {
		clickTimes = clickTimes[:0]
		return true
	}
	return false
}

// 혼잣말에 맞는 이모티콘 (없으면 "")
func emoForLine(l string) string {
	switch {
	case strings.Contains(l, "화이팅"):
		return "cheer"
	case strings.Contains(l, "사랑해"):
		return "heart"
	case strings.Contains(l, "최고"):
		return "clap"
	}
	return ""
}
