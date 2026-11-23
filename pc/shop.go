package main

// 꾸미기 상점: 바나나로 모자·안경·나비넥타이를 사서 씌워 주기
//
// 아이템 그림(acc/*.png)과 프레임별 머리 위치(anchors.go)는 tools/accessories.py 가 만듭니다.
// 그림은 "머리 좌표계"(기준 머리의 아래 가운데가 원점)로 그려져 있고,
// render() 가 프레임마다 headAnchors 로 옮겨 그 위에 덧그립니다.

import (
	"embed"
	"fmt"
	"math"
	"time"
)

//go:embed acc/*.png
var accFS embed.FS

type anchor struct{ X, Y, S, Ang float64 }

const (
	slotHead = iota
	slotFace
	slotNeck
	slotCount
)

var slotNames = [slotCount]string{"머리", "얼굴", "목"}

type item struct {
	id, name string
	price    int
	slot     int
	line     string // 처음 썼을 때 하는 말
}

var items = []item{
	{"ribbon", "빨간 리본", 8, slotHead, "리본 예뻐? 히히"},
	{"flower", "꽃 핀", 10, slotHead, "꽃 냄새 난다~ 향기롭지?"},
	{"sprout", "새싹", 12, slotHead, "머리에서 새싹이 났어! 쑥쑥 클게!"},
	{"beanie", "털모자", 25, slotHead, "따뜻하다~ 겨울 준비 끝!"},
	{"santa", "산타 모자", 25, slotHead, "호호호! 선물 줄게~"},
	{"crown", "왕관", 40, slotHead, "나 이제 우땅이 왕이야!"},
	{"glasses", "동그란 안경", 15, slotFace, "똑똑해 보여? 에헴!"},
	{"sunglasses", "선글라스", 20, slotFace, "멋있지? 우땅 스웩~"},
	{"bowtie", "나비넥타이", 15, slotNeck, "오늘은 신사 우땅이야!"},
	// 가격 0 = 팔지 않는 선물 (ach.go 의 출석 업적으로 받음)
	{"rainbow", "무지개 핀", 0, slotHead, "무지개처럼 반짝반짝!"},
	{"bananaband", "바나나 머리띠", 0, slotHead, "100일 동안 고마워! 바나나 왕이다!"},
	// 날짜 이벤트 선물 (season.go)
	{"rudolph", "루돌프 뿔", 0, slotHead, "루돌프 사슴 우땅이~ 메리 크리스마스!"},
	{"witch", "마녀 모자", 0, slotHead, "수리수리 바나나~ 얍!"},
	{"bokjumeoni", "복주머니", 0, slotNeck, "복 많이 받아! 복주머니 짤랑짤랑~"},
}

// 선물 아이템을 받는 방법 (메뉴·수첩 표시용)
var giftFrom = map[string]string{
	"rainbow":    "누적 출석 30일 선물",
	"bananaband": "누적 출석 100일 선물",
	"rudolph":    "크리스마스 선물",
	"witch":      "할로윈 선물",
	"bokjumeoni": "설날·추석 선물",
}

// 아이템을 그냥 손에 넣음 (선물). 이미 있으면 false
func giveItem(id string) bool {
	if owns(id) {
		return false
	}
	it, ok := findItem(id)
	if !ok {
		return false
	}
	ensureWearSlots()
	S.Owned = append(S.Owned, id)
	S.Wear[it.slot] = id
	applyWear()
	return true
}

// 착용 중인 아이템의 그림 (자리별, 없으면 nil)
var (
	accSprites = map[string]*Sprite{}
	worn       [slotCount]*Sprite
	wornFace   bool // 얼굴 자리 아이템이 있는지 (noFaceSeq 처리용)
)

const accOX, accOY = 200.0, 330.0 // 아이템 그림 안의 머리 원점

func findItem(id string) (item, bool) {
	for _, it := range items {
		if it.id == id {
			return it, true
		}
	}
	return item{}, false
}

func owns(id string) bool {
	for _, o := range S.Owned {
		if o == id {
			return true
		}
	}
	return false
}

func wearing(id string) bool {
	for _, w := range S.Wear {
		if w == id {
			return true
		}
	}
	return false
}

func loadAccessories() {
	for _, it := range items {
		b, err := accFS.ReadFile("acc/" + it.id + ".png")
		if err != nil {
			continue
		}
		accSprites[it.id] = decodeSprite(b)
	}
	applyWear()
}

// 저장된 착용 상태 → 그릴 그림
func applyWear() {
	for i := range worn {
		worn[i] = nil
		if i < len(S.Wear) && S.Wear[i] != "" && owns(S.Wear[i]) {
			worn[i] = accSprites[S.Wear[i]]
		}
	}
	wornFace = worn[slotFace] != nil
}

func ensureWearSlots() {
	for len(S.Wear) < slotCount {
		S.Wear = append(S.Wear, "")
	}
}

// 메뉴에서 아이템을 골랐을 때: 없으면 사서 쓰고, 있으면 쓰기/벗기
func doItem(i int) {
	it := items[i]
	ensureWearSlots()
	if !owns(it.id) {
		if it.price == 0 { // 선물 아이템은 살 수 없음
			say(josa(it.name, "은", "는")+" "+giftFrom[it.id]+"이야!", 3*time.Second)
			return
		}
		if S.Bananas < it.price {
			setMode("st_cry", 2500*time.Millisecond)
			say(fmt.Sprintf("%s 갖고 싶은데…\n바나나가 %d개 모자라…", it.name, it.price-S.Bananas), 3500*time.Millisecond)
			return
		}
		S.Bananas -= it.price
		S.Owned = append(S.Owned, it.id)
		S.Wear[it.slot] = it.id
		S.Mood = clamp(S.Mood + 15)
		addExp(expBuy) // 저장 포함
		applyWear()
		setMode("laugh", rd(2.5, 3.2))
		say(fmt.Sprintf("우와! %s 샀다!\n%s", it.name, it.line), 4500*time.Millisecond)
		nextTalk = time.Now().Add(rd(60, 120))
		return
	}
	if S.Wear[it.slot] == it.id {
		S.Wear[it.slot] = ""
		saveState()
		applyWear()
		setMode("idle", rd(1.5, 2.5))
		say(it.name+" 벗었어!", 2500*time.Millisecond)
		return
	}
	S.Wear[it.slot] = it.id
	saveState()
	applyWear()
	setMode("st_me", 2200*time.Millisecond)
	say(pick([]string{"어때? 어울려?", it.name + " 썼어! 히히", it.line}), 3*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}

func takeOffAll() {
	S.Wear = make([]string, slotCount)
	saveState()
	applyWear()
	setMode("idle", rd(1.5, 2.5))
	say("다 벗었어! 가뿐하다~", 2500*time.Millisecond)
}

// 꾸미고 있을 때 가끔 하는 자랑 (없으면 "")
func wearLine() string {
	var on []string
	for _, w := range S.Wear {
		if it, ok := findItem(w); ok && owns(w) {
			on = append(on, it.name)
		}
	}
	if len(on) == 0 {
		return ""
	}
	n := pick(on)
	return pick([]string{"내 " + n + " 예쁘지?", n + " 사줘서 고마워 " + voc() + "!", n + " 쓰니까 기분 좋다~"})
}

const idShopBase = 1100 // + items 번호
const idTakeOff = 1199

func appendShopMenu(m uintptr) {
	sh, _, _ := pCreatePopupMenu.Call()
	for s := 0; s < slotCount; s++ {
		if s > 0 {
			appendMenu(sh, MF_SEPARATOR, 0, "")
		}
		appendMenu(sh, MF_GRAYED, 0, "─ "+slotNames[s]+" ─")
		for i, it := range items {
			if it.slot != s {
				continue
			}
			fl := uintptr(MF_STRING)
			var label string
			switch {
			case wearing(it.id) && owns(it.id):
				fl |= MF_CHECKED
				label = it.name + "  (쓰는 중 · 누르면 벗기)"
			case owns(it.id):
				label = it.name + "  (가지고 있음)"
			case it.price == 0:
				fl |= MF_GRAYED
				label = it.name + "  (" + giftFrom[it.id] + ")"
			default:
				label = fmt.Sprintf("%s  (바나나 %d개)", it.name, it.price)
				if S.Bananas < it.price {
					fl |= MF_GRAYED
				}
			}
			appendMenu(sh, fl, uintptr(idShopBase+i), label)
		}
	}
	appendMenu(sh, MF_SEPARATOR, 0, "")
	appendMenu(sh, MF_STRING, idTakeOff, "모두 벗기")
	appendMenu(m, MF_POPUP, sh, "꾸미기")
}

// ── 그리기 ──

// 프레임 좌표(fx, fy: 연속 좌표)에서 착용 아이템 색을 dst 위에 덧칠 (premultiplied)
func drawWorn(fr *Sprite, fx, fy float64, dst *[4]float32) {
	if dirtSprite != nil && dirtyShown() && !noFaceSeq[fr.Seq] { // 얼룩은 아이템 아래
		drawLayer(fr, dirtSprite, fx, fy, dst)
	}
	for s := 0; s < slotCount; s++ {
		if worn[s] != nil && !(s == slotFace && noFaceSeq[fr.Seq]) {
			drawLayer(fr, worn[s], fx, fy, dst)
		}
	}
}

// 아이템 그림 하나를 프레임 fr 의 머리 위치에 맞춰 덧칠
func drawLayer(fr *Sprite, sp *Sprite, fx, fy float64, dst *[4]float32) {
	a := fr.Anc
	if a == nil {
		return
	}
	dx, dy := fx-a.X, fy-a.Y
	lx := (dx*a.cos - dy*a.sin) / a.S
	ly := (dx*a.sin + dy*a.cos) / a.S
	ax, ay := lx+accOX-0.5, ly+accOY-0.5
	if ax < -1 || ay < -1 || ax > 401 || ay > 401 {
		return
	}
	c := sample(sp, ax, ay)
	if c[3] < 1 {
		return
	}
	k := 1 - c[3]/255
	dst[0] = c[0] + dst[0]*k
	dst[1] = c[1] + dst[1]*k
	dst[2] = c[2] + dst[2]*k
	dst[3] = c[3] + dst[3]*k
}

// 각 프레임에 머리 위치 연결 (loadSeq 뒤에 호출)
type anchorRT struct {
	anchor
	cos, sin float64
}

func linkAnchors() {
	for name, frs := range seqs {
		an := headAnchors[name]
		for k, f := range frs {
			f.Seq = name
			if k < len(an) {
				a := an[k]
				r := a.Ang * math.Pi / 180
				f.Anc = &anchorRT{a, math.Cos(r), math.Sin(r)}
			}
		}
	}
}
