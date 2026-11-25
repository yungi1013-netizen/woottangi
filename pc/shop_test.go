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

// 새 꾸미기 · 코디 저장 · 오늘의 코디
func TestOutfits(t *testing.T) {
	loadSeq("idle")
	linkAnchors()
	statePathOverride = t.TempDir()
	S = petState{Bananas: 500, Full: 70, Mood: 50}
	loadAccessories()
	for _, it := range items {
		if accSprites[it.id] == nil {
			t.Fatalf("%s 그림 없음", it.id)
		}
	}
	buy := func(id string) {
		for i, it := range items {
			if it.id == id {
				doItem(i)
			}
		}
		if !owns(id) {
			t.Fatalf("%s 못 삼", id)
		}
	}
	saveOutfit(0) // 아무것도 안 썼으면 저장 안 함
	if outfitLabel(outfitAt(0)) != "" {
		t.Fatal("빈 모습이 저장됨")
	}
	buy("catears")
	buy("blush")
	buy("pearl")
	if S.Wear[slotHead] != "catears" || S.Wear[slotFace] != "blush" || S.Wear[slotNeck] != "pearl" {
		t.Fatalf("산 아이템을 안 씀: %v", S.Wear)
	}
	saveOutfit(1)
	if got := outfitLabel(outfitAt(1)); got != "고양이 귀 · 발그레 볼터치 · 진주 목걸이" {
		t.Fatalf("코디 2: %q", got)
	}
	takeOffAll()
	wearOutfit(1)
	if S.Wear[slotHead] != "catears" || S.Wear[slotNeck] != "pearl" || worn[slotHead] == nil {
		t.Fatalf("코디 입기 실패: %v", S.Wear)
	}
	wearOutfit(2) // 빈 칸: 그대로
	if S.Wear[slotHead] != "catears" {
		t.Fatal("빈 코디가 옷을 벗김")
	}
	for i := 0; i < 30; i++ {
		randomOutfit()
		if outfitLabel(S.Wear) == "" {
			t.Fatal("오늘의 코디가 빈 모습")
		}
		for _, w := range S.Wear {
			if w != "" && !owns(w) {
				t.Fatalf("없는 아이템을 입음: %s", w)
			}
		}
	}
	if S.RandomCnt != 30 {
		t.Fatalf("RandomCnt %d", S.RandomCnt)
	}
	// 저장 후 다시 읽어도 코디가 남아 있음
	saveState()
	S = petState{}
	loadState()
	if outfitLabel(outfitAt(1)) == "" || S.RandomCnt != 30 {
		t.Fatalf("코디가 저장 안 됨: %v", S.Outfits)
	}
}
