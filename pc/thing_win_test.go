package main

import (
	"testing"
	"time"
)

var pGetPixelT = gdi32.NewProc("GetPixel")

// 실제 창으로 공·풍선을 띄워 화면에 그림이 보이는지 확인 (가운데 픽셀이 배경과 달라야 함)
func TestThingVisibleOnScreen(t *testing.T) {
	loadFX()
	old := thingWindows
	thingWindows = true
	defer func() { thingWindows = old }()
	for _, k := range []string{"ball", "balloon"} {
		th := newThing(k, 120)
		if th.hwnd == 0 {
			t.Skip("창을 만들 수 없는 환경")
		}
		th.x, th.y = 400, 400
		// 그림 가운데가 흰색일 수도 있어서 여러 점을 봄 (하나라도 바뀌면 보이는 것)
		var pts [][2]uintptr
		for dy := -30; dy <= 30; dy += 15 {
			for dx := -30; dx <= 30; dx += 15 {
				pts = append(pts, [2]uintptr{uintptr(400 + dx), uintptr(400 + dy)})
			}
		}
		sdc, _, _ := pGetDC.Call(0)
		before := make([]uintptr, len(pts))
		for i, p := range pts {
			before[i], _, _ = pGetPixelT.Call(sdc, p[0], p[1])
		}
		th.draw()
		time.Sleep(300 * time.Millisecond)
		changed := 0
		for i, p := range pts {
			if c, _, _ := pGetPixelT.Call(sdc, p[0], p[1]); c != before[i] {
				changed++
			}
		}
		pReleaseDC.Call(0, sdc)
		th.close()
		if changed == 0 {
			t.Fatalf("%s 가 화면에 안 보임 (%d점 모두 그대로)", k, len(pts))
		}
		t.Logf("%s: %d/%d점 바뀜", k, changed, len(pts))
	}
}
