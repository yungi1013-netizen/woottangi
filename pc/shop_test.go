package main

// go test -run TestWornRender   →  pc/test_worn.png (실제 그리기 코드로 아이템을 씌운 프레임 모음)

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestWornRender(t *testing.T) {
	names := []string{"idle", "walk", "sit", "sleep", "eat", "laugh", "punish", "st_cry", "st_bow", "st_me", "st_lounge", "st_surprise"}
	for _, n := range names {
		loadSeq(n)
	}
	linkAnchors()
	S = petState{Owned: []string{"crown", "glasses", "bowtie"}, Wear: []string{"crown", "glasses", "bowtie"}}
	loadAccessories()
	for _, n := range names {
		for k, f := range seqs[n] {
			if f.Anc == nil {
				t.Fatalf("%s_%d: 머리 위치 없음", n, k)
			}
		}
	}
	const tw, th = 240, 300
	out := image.NewNRGBA(image.Rect(0, 0, tw*len(names), th))
	for i, n := range names {
		f := seqs[n][0]
		for y := 0; y < th; y++ {
			for x := 0; x < tw; x++ {
				su, sv := (float64(x)+0.5)*2-0.5, (float64(y)+0.5)*2-0.5
				c := sample(f, su, sv)
				drawWorn(f, su+0.5, sv+0.5, &c)
				a := c[3]
				bg := float32(230)
				px := func(v float32) uint8 { return uint8(v + bg*(1-a/255) + 0.5) }
				out.SetNRGBA(i*tw+x, y, color.NRGBA{px(c[2]), px(c[1]), px(c[0]), 255})
			}
		}
	}
	fp, err := os.Create("test_worn.png")
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()
	png.Encode(fp, out)
}

func TestShopFlow(t *testing.T) {
	loadSeq("idle")
	linkAnchors()
	statePathOverride = t.TempDir()
	S = petState{Bananas: 30, Full: 70, Mood: 50}
	loadAccessories()
	doItem(5) // 왕관 40개: 못 삼
	if owns("crown") || S.Bananas != 30 {
		t.Fatal("바나나가 모자란데 샀음")
	}
	doItem(0) // 리본 8개
	if !owns("ribbon") || S.Bananas != 22 || S.Wear[slotHead] != "ribbon" || worn[slotHead] == nil {
		t.Fatalf("리본 구매/착용 실패: %+v", S)
	}
	doItem(1) // 꽃 핀 10개 → 같은 머리 자리라 리본 대신 씀
	if S.Wear[slotHead] != "flower" || S.Bananas != 12 {
		t.Fatalf("자리 교체 실패: %+v", S)
	}
	doItem(1) // 다시 누르면 벗기
	if S.Wear[slotHead] != "" || worn[slotHead] != nil {
		t.Fatal("벗기 실패")
	}
	doItem(0) // 가진 리본은 공짜로 다시 쓰기
	if S.Wear[slotHead] != "ribbon" || S.Bananas != 12 {
		t.Fatal("다시 쓰기 실패")
	}
	takeOffAll()
	if worn[slotHead] != nil || len(S.Owned) != 2 {
		t.Fatal("모두 벗기 실패")
	}
}
