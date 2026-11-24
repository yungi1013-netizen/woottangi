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
		sdc, _, _ := pGetDC.Call(0)
		before, _, _ := pGetPixelT.Call(sdc, 400, 400)
		th.draw()
		time.Sleep(300 * time.Millisecond)
		after, _, _ := pGetPixelT.Call(sdc, 400, 400)
		pReleaseDC.Call(0, sdc)
		th.close()
		if after == before {
			t.Fatalf("%s 가 화면에 안 보임 (픽셀 %06x)", k, after)
		}
		t.Logf("%s: %06x → %06x", k, before, after)
	}
}
