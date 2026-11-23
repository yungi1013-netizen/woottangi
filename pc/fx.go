package main

// 효과 조각 그리기
//   - 우땅이 주변(우땅이 창 안): 목욕 거품 · 반짝이 · 하트
//   - 화면 전체(효과 창): 계절 날씨 쇼 — 눈 · 벚꽃잎 · 낙엽 · 비눗방울 · 꽃가루(색종이)
//     효과 창은 마우스가 그냥 지나가는(클릭이 아래 창으로 가는) 투명 창이고, 쇼가 있을 때만 만들어서 씀
//
// 조각 그림(fx/*.png)은 tools/effects.py 가 만듭니다.

import (
	"embed"
	"io/fs"
	"math"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed fx/*.png
var fxFS embed.FS

// 조각 그림 (작게 그릴 때 깨지지 않도록 절반 크기도 같이)
type fxSprite struct{ full, half *Sprite }

var fxSprites = map[string]*fxSprite{}

func loadFX() {
	ents, _ := fs.ReadDir(fxFS, "fx")
	for _, e := range ents {
		b, err := fxFS.ReadFile("fx/" + e.Name())
		if err != nil {
			continue
		}
		sp := decodeSprite(b)
		fxSprites[strings.TrimSuffix(e.Name(), ".png")] = &fxSprite{sp, halve(sp)}
	}
}

// 2x2 평균으로 절반 크기
func halve(s *Sprite) *Sprite {
	w, h := s.W/2, s.H/2
	o := &Sprite{W: w, H: h, Pix: make([]uint8, w*h*4)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for c := 0; c < 4; c++ {
				a := int(s.Pix[((2*y)*s.W+2*x)*4+c]) + int(s.Pix[((2*y)*s.W+2*x+1)*4+c]) +
					int(s.Pix[((2*y+1)*s.W+2*x)*4+c]) + int(s.Pix[((2*y+1)*s.W+2*x+1)*4+c])
				o.Pix[(y*w+x)*4+c] = uint8((a + 2) / 4)
			}
		}
	}
	return o
}

// 버퍼(premultiplied BGRA, w×h)에 조각 하나: 중심 (x, y), 폭 size px, 회전 rot(라디안), 투명도 alpha
func blit(buf []byte, w, h int, fs *fxSprite, x, y, size, rot, alpha float64) {
	if fs == nil || alpha <= 0.01 || size < 1 {
		return
	}
	sp := fs.full
	if size < float64(sp.W)*0.6 {
		sp = fs.half
	}
	k := size / float64(sp.W)
	r := size * 0.72
	x0, x1 := max(0, int(x-r)), min(w-1, int(x+r)+1)
	y0, y1 := max(0, int(y-r)), min(h-1, int(y+r)+1)
	cs, sn := math.Cos(rot), math.Sin(rot)
	hw, hh := float64(sp.W)/2-0.5, float64(sp.H)/2-0.5
	al := float32(alpha)
	for py := y0; py <= y1; py++ {
		dy := float64(py) + 0.5 - y
		for px := x0; px <= x1; px++ {
			dx := float64(px) + 0.5 - x
			u := (dx*cs+dy*sn)/k + hw
			v := (-dx*sn+dy*cs)/k + hh
			c := sample(sp, u, v)
			if c[3] < 1 {
				continue
			}
			o := (py*w + px) * 4
			a := c[3] * al / 255
			for i := 0; i < 4; i++ {
				buf[o+i] = clampB(c[i]*al + float32(buf[o+i])*(1-a))
			}
		}
	}
}

// ── 조각 하나 ──

type particle struct {
	sp                 *fxSprite
	x, y, vx, vy, grav float64
	rot, vr, size      float64
	sway, phase        float64 // 좌우 흔들림 (px/s)
	age, life          float64 // 초
	fadeIn             bool
}

func (p *particle) step(dt float64) {
	p.age += dt
	p.vy += p.grav * dt
	p.x += (p.vx + p.sway*math.Sin(p.age*1.7+p.phase)) * dt
	p.y += p.vy * dt
	p.rot += p.vr * dt
}

func (p *particle) alpha() float64 {
	a := 1.0
	if p.fadeIn && p.age < 0.6 {
		a = p.age / 0.6
	}
	if left := p.life - p.age; left < 0.8 {
		a = math.Min(a, left/0.8)
	}
	return math.Max(0, a)
}

func stepAll(ps []particle, dt float64, alive func(*particle) bool) []particle {
	out := ps[:0]
	for i := range ps {
		p := ps[i]
		p.step(dt)
		if p.age < p.life && alive(&p) {
			out = append(out, p)
		}
	}
	return out
}

func rf(a, b float64) float64 { return a + rng.Float64()*(b-a) }

// ── 우땅이 주변 조각 (우땅이 창 좌표) ──

var petFx []particle

// 거품 하나 (목욕)
func addBubbleAt(x, y float64) {
	petFx = append(petFx, particle{sp: fxSprites["bubble"], x: x, y: y, vx: rf(-12, 12) * dpi, vy: rf(-38, -18) * dpi,
		size: rf(9, 17) * eff() / 0.75, sway: rf(4, 10) * dpi, phase: rf(0, 6), life: rf(1.2, 2.2), fadeIn: true})
}

// 반짝이 · 하트 퐁퐁 (머리 위쪽에서 사방으로)
func burst(kind string, n int) {
	for i := 0; i < n; i++ {
		a := rf(-math.Pi, 0)
		sp := rf(40, 90) * dpi
		petFx = append(petFx, particle{sp: fxSprites[kind], x: float64(canvasW) * rf(0.3, 0.7), y: float64(canvasH) * rf(0.25, 0.5),
			vx: math.Cos(a) * sp, vy: math.Sin(a) * sp, grav: 60 * dpi, rot: rf(-0.4, 0.4), vr: rf(-2, 2),
			size: rf(11, 18) * eff() / 0.75, life: rf(0.9, 1.5)})
	}
}

func petFxTick(dt float64) {
	if len(petFx) == 0 {
		return
	}
	petFx = stepAll(petFx, dt, func(p *particle) bool { return p.y > -20 })
}

func drawPetFx(buf []byte) {
	for i := range petFx {
		p := &petFx[i]
		blit(buf, canvasW, canvasH, p.sp, p.x, p.y, p.size, p.rot, p.alpha())
	}
}

// ── 화면 전체 효과 창 ──

var (
	fxHwnd, fxDC, fxDib uintptr
	fxBits              unsafe.Pointer
	fxW, fxH            int
	fxX, fxY            int
	screenFx            []particle
	fxClassDone         bool
)

// 날씨 쇼 종류
type weather struct {
	sprites  []string
	rise     bool       // 아래에서 위로 (비눗방울)
	speed    [2]float64 // px/s
	size     [2]float64 // px (dpi 1 기준)
	sway     float64
	spin     float64
	perSec   float64
	fadeIn   bool
	duration time.Duration
}

var weathers = map[string]weather{
	"snow":     {[]string{"snow"}, false, [2]float64{45, 95}, [2]float64{14, 30}, 30, 0.8, 7, true, 40 * time.Second},
	"petal":    {[]string{"petal"}, false, [2]float64{50, 90}, [2]float64{14, 24}, 45, 2.2, 6, true, 35 * time.Second},
	"leaf":     {[]string{"maple", "ginkgo"}, false, [2]float64{60, 110}, [2]float64{18, 30}, 55, 2.6, 4.5, true, 35 * time.Second},
	"bubble":   {[]string{"bubble"}, true, [2]float64{40, 80}, [2]float64{16, 36}, 25, 0, 4, true, 30 * time.Second},
	"confetti": {[]string{"conf0", "conf1", "conf2", "conf3", "star"}, false, [2]float64{90, 160}, [2]float64{10, 18}, 40, 5, 14, false, 25 * time.Second},
	"xmas":     {[]string{"snow", "snow", "snow", "star"}, false, [2]float64{45, 95}, [2]float64{14, 30}, 30, 1, 8, true, 45 * time.Second},
	"heart":    {[]string{"heart", "heart", "star"}, true, [2]float64{50, 90}, [2]float64{16, 28}, 30, 0.6, 4, true, 30 * time.Second},
	"spooky":   {[]string{"candy", "maple", "star"}, false, [2]float64{70, 120}, [2]float64{16, 28}, 45, 2.6, 4.5, true, 30 * time.Second},
}

var (
	showKind  string
	showUntil time.Time
	showAcc   float64 // 다음 조각까지 쌓인 값
)

func fxWndProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}

// 효과 창 열기 (작업 영역 크기)
func openFxWindow() bool {
	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("WoottangiFx")
	if !fxClassDone {
		wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(fxWndProc), Instance: inst, ClassName: cls}
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		fxClassDone = true
	}
	fxX, fxY = int(workL), int(workT)
	fxW, fxH = int(workR-workL), int(ground+float64(canvasH)-2-workT)
	if fxW < 100 || fxH < 100 {
		return false
	}
	if fxHwnd == 0 {
		const exTransparent, exNoActivate = 0x20, 0x08000000
		fxHwnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW|exTransparent|exNoActivate,
			uintptr(unsafe.Pointer(cls)), 0, WS_POPUP, uintptr(int32(fxX)), uintptr(int32(fxY)), uintptr(fxW), uintptr(fxH), 0, 0, inst, 0)
		if fxHwnd == 0 {
			return false
		}
	}
	if fxDC == 0 {
		sdc, _, _ := pGetDC.Call(0)
		fxDC, _, _ = pCreateCompatibleDC.Call(sdc)
		pReleaseDC.Call(0, sdc)
	}
	bi := BITMAPINFOHEADER{Size: 40, Width: int32(fxW), Height: -int32(fxH), Planes: 1, BitCount: 32}
	var p unsafe.Pointer
	fxDib, _, _ = pCreateDIBSection.Call(fxDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&p)), 0, 0)
	if fxDib == 0 || p == nil {
		return false
	}
	fxBits = p
	pSelectObject.Call(fxDC, fxDib)
	return true
}

// 쇼가 끝나면 그림 버퍼를 놓아 줌 (메모리 아끼기)
func closeFxWindow() {
	if fxHwnd != 0 {
		pShowWindow.Call(fxHwnd, SW_HIDE)
	}
	if fxDib != 0 {
		pDeleteObject.Call(fxDib)
		fxDib, fxBits = 0, nil
	}
	screenFx = screenFx[:0]
}

// 날씨 쇼 시작 (이미 하고 있으면 종류만 바꿈)
func startShow(kind string) bool {
	w, ok := weathers[kind]
	if !ok || hiddenFS || catching {
		return false
	}
	if fxBits == nil && !openFxWindow() {
		return false
	}
	showKind, showUntil, showAcc = kind, time.Now().Add(w.duration), 0
	return true
}

func showing() bool { return fxBits != nil }

func spawnWeather(w weather) {
	sp := fxSprites[pick(w.sprites)]
	size := rf(w.size[0], w.size[1]) * dpi
	speed := rf(w.speed[0], w.speed[1]) * dpi * (0.7 + 0.3*size/(w.size[1]*dpi)) // 큰 조각이 조금 빨리
	p := particle{sp: sp, x: rf(0, float64(fxW)), size: size, sway: rf(0.4, 1) * w.sway * dpi, phase: rf(0, 6),
		rot: rf(-math.Pi, math.Pi), vr: rf(-w.spin, w.spin), fadeIn: w.fadeIn, vx: rf(-10, 10) * dpi}
	if w.rise {
		p.y, p.vy = float64(fxH)+size, -speed
		p.life = float64(fxH)*0.85/speed + 1
		p.rot = rf(-0.3, 0.3)
	} else {
		p.y, p.vy = -size, speed
		p.life = (float64(fxH)+size*2)/speed + 1
	}
	screenFx = append(screenFx, p)
}

func screenFxTick(now time.Time, dt float64) {
	if fxBits == nil || catching { // 바탕화면 바나나 받기가 창을 쓰는 중
		return
	}
	if hiddenFS { // 전체화면이 되면 바로 그만
		closeFxWindow()
		showKind = ""
		return
	}
	if showKind != "" {
		w := weathers[showKind]
		if now.After(showUntil) {
			showKind = ""
		} else {
			showAcc += w.perSec * dt * float64(fxW) / (1600 * dpi) // 화면이 넓으면 조금 더
			for showAcc >= 1 {
				showAcc--
				spawnWeather(w)
			}
		}
	}
	h := float64(fxH)
	screenFx = stepAll(screenFx, dt, func(p *particle) bool { return p.y < h+p.size && p.y > -p.size*2 })
	if showKind == "" && len(screenFx) == 0 {
		closeFxWindow()
		return
	}
	buf := unsafe.Slice((*byte)(fxBits), fxW*fxH*4)
	clear(buf)
	for i := range screenFx {
		p := &screenFx[i]
		blit(buf, fxW, fxH, p.sp, p.x, p.y, p.size, p.rot, p.alpha())
	}
	sdc, _, _ := pGetDC.Call(0)
	pt := POINT{int32(fxX), int32(fxY)}
	sz := SIZE{int32(fxW), int32(fxH)}
	src := POINT{0, 0}
	bl := BLENDFUNCTION{AC_SRC_OVER, 0, 255, AC_SRC_ALPHA}
	pUpdateLayeredWindow.Call(fxHwnd, sdc, uintptr(unsafe.Pointer(&pt)), uintptr(unsafe.Pointer(&sz)),
		fxDC, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&bl)), ULW_ALPHA)
	pReleaseDC.Call(0, sdc)
	pShowWindow.Call(fxHwnd, 4) // SW_SHOWNOACTIVATE (이미 보이면 그대로)
}
