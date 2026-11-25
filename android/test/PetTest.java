import com.woottangi.pet.Pet;

import java.util.HashMap;
import java.util.Map;
import java.util.Random;

/** PC 에서 돌리는 우땅이 두뇌 테스트: python tools/build_android.py --test */
public class PetTest {
    static int fails = 0;

    static void check(boolean ok, String what) {
        if (!ok) { fails++; System.out.println("FAIL: " + what); }
    }

    static class MapStore implements Pet.Store {
        final Map<String, Object> m = new HashMap<>();
        public int getInt(String k, int d) { Object v = m.get(k); return v == null ? d : (Integer) v; }
        public float getFloat(String k, float d) { Object v = m.get(k); return v == null ? d : (Float) v; }
        public long getLong(String k, long d) { Object v = m.get(k); return v == null ? d : (Long) v; }
        public String getString(String k, String d) { Object v = m.get(k); return v == null ? d : (String) v; }
        public void put(String k, Object v) { m.put(k, v); }
        public void commit() {}
    }

    static Pet fresh(long seed) {
        Pet p = new Pet(new Random(seed));
        p.load(new MapStore(), System.currentTimeMillis());
        p.scrW = 1080; p.floorY = 2200; p.w = 300; p.h = 375; p.dp = 2.75f;
        p.x = 400; p.y = 0;
        return p;
    }

    static void run(Pet p, float sec, long now) {
        for (float t = 0; t < sec; t += 1 / 60f) p.tick(1 / 60f, now, 14, 30);
    }

    public static void main(String[] a) {
        long now = System.currentTimeMillis();

        // 레벨 · 칭호
        check(Pet.levelOf(0)[0] == 1 && Pet.levelOf(40)[0] == 2 && Pet.levelOf(99)[0] == 2 && Pet.levelOf(100)[0] == 3, "levelOf");
        check(Pet.levelOf(999999)[0] == 50, "max level");
        check(Pet.titleOf(1).equals("아기 우땅이") && Pet.titleOf(7).equals("장난꾸러기 우땅이") && Pet.titleOf(50).equals("영원한 가족 우땅이"), "titles");

        // 호칭
        Pet p = fresh(1);
        p.name = "지호"; check(p.voc().equals("지호야"), "voc 야: " + p.voc());
        p.name = "민준"; check(p.voc().equals("민준아"), "voc 아: " + p.voc());
        p.name = ""; check(p.voc().equals("주인님"), "voc 기본");

        // 떨어져서 착지 + 인사
        p = fresh(2);
        run(p, 3, now);
        check(Math.abs(p.y - (p.floorY - p.h)) < 1, "landed y=" + p.y);
        check(p.greeted, "greeted");
        check(p.sayText != null && p.sayText.contains("안녕"), "greeting said");

        // 오래 돌려도 화면 안에 있고 NaN 없음
        p = fresh(3);
        for (int i = 0; i < 60 * 600; i++) {
            p.tick(1 / 60f, now, 14, 30);
            p.sayText = null;
            if (Float.isNaN(p.x) || p.x < -p.w * 0.2f || p.x > p.scrW - p.w * 0.8f) { check(false, "x out " + p.x); break; }
        }

        // 던지기: 튕기고 결국 바닥
        p = fresh(4);
        run(p, 3, now); p.sayText = null;
        p.startDrag(); p.dragTo(500, 600);
        p.release(4000, -3000);
        check(p.thrown && p.mode.equals("fall"), "thrown");
        run(p, 6, now);
        check(!p.mode.equals("fall") && Math.abs(p.y - (p.floorY - p.h)) < 1, "landed after throw " + p.mode);
        check(p.x >= -p.w * 0.15f - 0.01f && p.x <= p.scrW - p.w * 0.85f + 0.01f, "thrown stays in screen " + p.x);

        // 살짝 놓기는 그냥 떨어짐
        p = fresh(5);
        run(p, 3, now);
        p.startDrag(); p.dragTo(300, 800); p.release(100, 50);
        check(!p.thrown, "gentle release not thrown");

        // 간식
        p = fresh(6); run(p, 3, now);
        p.full = 50; p.bananas = 10; int e0 = p.exp;
        check(p.snack(0) && p.bananas == 9 && p.full == 75 && p.exp == e0 + 3 && p.mode.equals("eat"), "snack banana");
        p.full = 99; check(!p.snack(0), "too full");
        p.full = 10; p.bananas = 2; check(!p.snack(2) && p.bananas == 2, "not enough bananas");

        // 쓰다듬기: 하루 10번까지만 보상
        p = fresh(7); run(p, 3, now);
        int ex = p.exp;
        for (int i = 0; i < 15; i++) { p.setMode("idle", 5); p.petIt(now); }
        check(p.petCnt == 10 && p.petTotal == 15 && p.exp == ex + 20, "pet limit " + p.petCnt + " " + p.exp);

        // 출석: 하루 한 번, 연속
        p = fresh(8);
        long d1 = now - 86400000L;
        int b0 = p.bananas;
        check(p.attend(d1) == 5 && p.streak == 1, "attend day1");
        check(p.attend(d1) == 0, "attend twice");
        check(p.attend(now) == 5 && p.streak == 2 && p.bananas == b0 + 10, "attend streak");

        // 레벨 업 축하
        p = fresh(9); run(p, 3, now); p.sayText = null;
        p.addExp(45);
        run(p, 0.1f, now);
        check(p.lvShown == 2 && p.sayText != null && p.sayText.contains("레벨 업") && p.toastText != null, "level up");

        // 저장 · 불러오기 · 떨어져 있던 시간
        MapStore st = new MapStore();
        p = fresh(10); p.bananas = 33; p.full = 80; p.mood = 90; p.name = "우리";
        p.save(st, now - 2 * 3600000L);
        Pet q = new Pet(new Random(1)); q.load(st, now);
        check(q.bananas == 33 && q.name.equals("우리"), "save/load");
        check(Math.abs(q.full - 68) < 0.5 && Math.abs(q.mood - 82) < 0.5, "away decay " + q.full + " " + q.mood);

        // 운세는 하루 한 번만 보상
        p = fresh(11); int le = p.exp;
        p.luck(now); p.luck(now);
        check(p.exp == le + 3 && p.sayText.contains("운세"), "luck");

        // 그림 고르기
        p = fresh(12); run(p, 3, now);
        p.setMode("walk", 5); p.dir = -1;
        check(p.flip() && p.seqName().equals("walk") && p.frameIndex(8) >= 0 && p.frameIndex(8) < 8, "frames");

        // 게임 점수
        p = fresh(13); run(p, 3, now); p.sayText = null;
        int bb = p.bananas;
        check(p.gameScore(200, 0) == 80 && p.bananas == bb + 80 && p.best == 80 && p.playCnt == 1, "catch score capped");
        check(p.gameScore(5, 3) == 0, "match too few moves rejected");
        check(p.gameScore(99, 12) == 14 && p.matchCnt == 1 && p.bestMatch == 12, "match score from moves");
        p.full = 10; check(!p.canPlay(), "hungry can't play");

        // 숨바꼭질: 숨고 → 안 보이고 → 빼꼼 → 찾으면 바나나
        p = fresh(14); run(p, 3, now); p.sayText = null;
        p.startHide();
        check(p.hiding && p.visible(), "hide start visible");
        run(p, 2.6f, now);
        check(p.hideStage == 1 && !p.visible(), "hide invisible");
        run(p, 1.6f, now);
        check(p.hideStage == 2 && p.mode.equals("hide") && p.visible(), "hiding spot " + p.mode);
        boolean edge = p.x < 0 || p.x > p.scrW - p.w || !Float.isNaN(p.clipY);
        check(edge, "hide spot at edge x=" + p.x + " clip=" + p.clipY);
        bb = p.bananas; int hc = p.hideCnt;
        p.tap();
        check(!p.hiding && p.bananas == bb + 8 && p.hideCnt == hc + 1 && Float.isNaN(p.clipY), "found quickly +8");
        run(p, 4, now);
        check(Math.abs(p.y - (p.floorY - p.h)) < 1 && p.sayText != null && p.sayText.contains("들켰다"), "lands after found");

        // 숨바꼭질 시간 초과
        p = fresh(15); run(p, 3, now);
        p.startHide();
        for (int i = 0; i < 60 * 70; i++) { p.tick(1 / 60f, now, 14, 30); }
        check(!p.hiding && p.hideCnt == 0, "hide timeout");
        // 여러 번 숨어서 세 군데 다 나오는지
        boolean L = false, R = false, B = false;
        for (int k = 0; k < 40; k++) {
            p = fresh(100 + k); run(p, 3, now); p.startHide(); run(p, 4.2f, now);
            if (!Float.isNaN(p.clipY)) B = true; else if (p.x < 0) L = true; else R = true;
        }
        check(L && R && B, "hide spots L/R/bottom " + L + R + B);

        // 뽑기: 바나나 10개, 결과 나오고, 아이템은 머리/얼굴/목에
        p = fresh(16); run(p, 3, now);
        p.bananas = 5; p.startGacha(); check(p.gachaCnt == 0, "gacha needs 10");
        int got = 0;
        for (int k = 0; k < 300; k++) {
            p.bananas = 100; p.setMode("idle", 5); p.sayText = null;
            p.startGacha();
            run(p, 2, now);
            check(p.sayText != null || p.toastText != null, "gacha result said");
            p.toastText = null;
        }
        got = p.ownedCount();
        check(p.gachaCnt == 300 && got == 4, "gacha all items " + got);
        check(p.wearing("idle").size() == 3 && p.wearing("st_me").size() == 2, "wear slots / no face");
        int idx = Pet.itemIndex(p.wear[2]);
        p.toggleWear(idx); check(p.wear[2].isEmpty(), "take off");
        // 저장
        st = new MapStore(); p.save(st, now); q = new Pet(new Random(1)); q.load(st, now);
        check(q.ownedCount() == 4 && q.wear[0].equals(p.wear[0]) && q.wear[2].isEmpty(), "items saved");

        // 땅콩이: Lv.5 전에는 안 옴
        p = fresh(17); run(p, 3, now); p.sayText = null;
        check(!p.startFriend(true, now) && p.friend == null, "friend needs lv5");
        p.exp = 400; p.lvShown = p.level();
        check(p.startFriend(true, now) && p.friend != null, "friend comes");
        int fc0 = p.friendCnt; bb = p.bananas;
        boolean greeted = false, shared = false; float maxT = 0;
        for (int i = 0; i < 60 * 200 && p.friend != null; i++) {
            p.tick(1 / 60f, now, 14, 30);
            if (p.friend != null && p.friend.stage.equals("play")) greeted = true;
            if (p.sayText != null && p.sayText.contains("나눠")) shared = true;
            p.sayText = null;
            maxT = i / 60f;
        }
        check(greeted && p.friendCnt == fc0 + 1, "friend greeted");
        check(p.friend == null, "friend left after " + maxT + "s");
        check(maxT > 50 && maxT < 120, "visit length " + maxT);
        // 자동 방문: 시간이 되면 옴
        p = fresh(18); run(p, 3, now); p.exp = 400; p.lvShown = p.level(); p.sayText = null;
        p.nextFriendAt = now - 1; p.setMode("idle", 5);
        p.tick(1 / 60f, now, 14, 30);
        check(p.friend != null && p.nextFriendAt > now + 59 * 60000L, "auto friend visit");

        // ── 벽 타기: 메뉴로 부르면 가까운 벽으로 가서 올라감
        p = fresh(30); run(p, 3, now); p.sayText = null;
        p.ceilY = 80;
        p.x = 200; p.goClimb(false);
        float t = 0;
        while (p.q == 0 && t < 20) { p.tick(1 / 60f, now, 14, 30); t += 1 / 60f; }
        check(p.q == 1, "climbs left wall q=" + p.q + " after " + t);
        check(p.winW() == p.h && p.winH() == p.w && p.x < 0, "wall window rotated x=" + p.x);
        check(p.mode.equals("climb") && p.flip() && p.seqName().equals("walk"), "climb anim up-facing");
        float y0 = p.y; run(p, 1, now);
        check(p.y < y0 || !p.mode.equals("climb"), "moves up the wall");

        // 천장까지
        p = fresh(31); run(p, 3, now); p.ceilY = 80;
        p.x = 700; p.goClimb(true);
        boolean sawRight = false, sawCeil = false;
        for (int i = 0; i < 60 * 60 && !sawCeil; i++) {
            p.tick(1 / 60f, now, 14, 30); p.sayText = null;
            if (p.q == 3) sawRight = true;
            if (p.q == 2) sawCeil = true;
            if (p.q == 0 && p.mode.equals("fall") && !sawCeil) { p.goClimb(true); } // 손 놓았으면 다시
        }
        check(sawRight && sawCeil, "right wall then ceiling " + sawRight + sawCeil);
        if (p.q == 2) {
            check(Math.abs(p.y - (p.ceilY - p.h * 0.015f)) < 1 && p.winW() == p.w, "on ceiling y=" + p.y);
            float x0 = p.x; p.setMode("climb", 5); run(p, 0.5f, now);
            check(p.x != x0 || p.q != 2, "walks along ceiling");
        }

        // 오래 돌려도 화면 안 · 결국 바닥으로 돌아옴
        p = fresh(32); run(p, 3, now); p.ceilY = 80;
        int[] seen = new int[4];
        for (int i = 0; i < 60 * 900; i++) {
            if (i % (60 * 40) == 0 && p.q == 0 && !p.busy()) p.goClimb(i % 2 == 0);
            p.tick(1 / 60f, now, 14, 30); p.sayText = null;
            seen[p.q]++;
            boolean bad = Float.isNaN(p.x) || Float.isNaN(p.y) || p.x < -p.h * 0.3f || p.x > p.scrW || p.y > p.floorY || p.y < p.ceilY - p.h * 0.3f;
            if (bad) { check(false, "out of screen q=" + p.q + " x=" + p.x + " y=" + p.y + " " + p.mode); break; }
            if (p.q == 0 && !p.mode.equals("fall") && !p.mode.equals("drag") && !p.perched && Math.abs(p.y - (p.floorY - p.h)) > 1) {
                check(false, "floating " + p.mode + " y=" + p.y); break;
            }
        }
        check(seen[1] > 0 && seen[2] > 0 && seen[3] > 0 && seen[0] > 0, "visited all surfaces " + java.util.Arrays.toString(seen));

        // 벽에서 톡 → 떨어져서 착지
        p = fresh(33); run(p, 3, now); p.ceilY = 80; p.x = 100; p.goClimb(false);
        for (int i = 0; i < 60 * 20 && p.q == 0; i++) p.tick(1 / 60f, now, 14, 30);
        run(p, 1, now);
        p.tap();
        check(p.q == 0 && p.mode.equals("fall"), "tap on wall falls");
        run(p, 4, now);
        check(Math.abs(p.y - (p.floorY - p.h)) < 1, "landed after wall");

        // 벽에서 간식 주면 내려와서 먹음 (떨어짐 → 착지)
        p = fresh(34); run(p, 3, now); p.ceilY = 80; p.x = 100; p.goClimb(false);
        for (int i = 0; i < 60 * 20 && p.q == 0; i++) p.tick(1 / 60f, now, 14, 30);
        p.full = 50; p.bananas = 5; p.snack(0);
        check(p.q == 0 && p.winW() == p.w, "snack detaches from wall");
        run(p, 4, now);
        check(Math.abs(p.y - (p.floorY - p.h)) < 1, "lands after snack on wall");

        // 살살 놓기: 왼쪽 벽 가까이 → 벽에 붙음 / 위쪽 → 천장 / 공중 → 그 자리에 앉음
        p = fresh(35); run(p, 3, now); p.ceilY = 80;
        p.startDrag(); p.dragTo(-30, 900); p.release(10, 10);
        check(p.q == 1 && p.mode.equals("cling"), "gentle near left wall clings q=" + p.q);
        p.startDrag(); p.dragTo(400, 70); p.release(0, 0);
        check(p.q == 2, "gentle near top hangs q=" + p.q);
        p.startDrag(); p.dragTo(400, 900); p.release(0, 0);
        check(p.perched && p.q == 0, "gentle mid-air perches");
        run(p, 15, now);
        check(p.perched && Math.abs(p.y - 900) < 1 && !p.mode.equals("walk"), "stays on perch " + p.mode + " y=" + p.y);
        run(p, 70, now);
        check(!p.perched && Math.abs(p.y - (p.floorY - p.h)) < 1, "hops down later");
        // 세게 던지면 그냥 날아감
        p.startDrag(); p.dragTo(400, 900); p.release(3000, -2000);
        check(!p.perched && p.mode.equals("fall"), "throw does not perch");

        // 내 자리: 정하고 → 바닥에서 폴짝 점프해서 정확히 그 자리로
        p = fresh(36); run(p, 3, now); p.ceilY = 80;
        p.startDrag(); p.dragTo(650, 700); p.release(0, 0);
        p.setSeat();
        check(p.hasSeat() && p.perched, "seat set");
        p.stay = true; // 내려와서 바로 다시 자리로 점프하지 않게
        p.comeDown(); run(p, 4, now); p.sayText = null;
        check(!p.perched && Math.abs(p.y - (p.floorY - p.h)) < 1, "back on floor");
        p.stay = false; p.x = 100; p.setMode("idle", 5);
        check(p.jumpToSeat(), "jump to seat");
        run(p, 3, now);
        check(p.perched && Math.abs(p.x - 650) < 2 && Math.abs(p.y - 700) < 2, "landed on seat x=" + p.x + " y=" + p.y);
        st = new MapStore(); p.save(st, now); q = new Pet(new Random(1)); q.load(st, now);
        check(q.hasSeat() && Math.abs(q.seatFX * q.scrW - 650) < 1, "seat saved");
        p.clearSeat(); check(!p.hasSeat(), "seat cleared");

        // 숨바꼭질은 벽에서도 시작됨 (먼저 내려옴)
        p = fresh(37); run(p, 3, now); p.ceilY = 80; p.x = 100; p.goClimb(false);
        for (int i = 0; i < 60 * 20 && p.q == 0; i++) p.tick(1 / 60f, now, 14, 30);
        p.startHide(); check(p.q == 0 && p.hiding, "hide from wall");

        // ── 장난감: 공을 쫓아가서 여러 번 차고, 공도 우땅이도 화면 안 ──
        p = fresh(50); run(p, 3, now); p.sayText = null;
        p.startToy("ball");
        check(p.toy != null && p.toyActive() && p.toyCnt == 1, "toy ball out");
        p.toy.x = 1000;
        boolean inside = true;
        for (int i = 0; i < 60 * 40; i++) {
            p.tick(1 / 60f, now, 14, 30); p.sayText = null;
            Pet.Toy ty = p.toy;
            if (ty == null) break;
            float r = ty.size / 2;
            if (ty.x < r * 0.8f - 0.5f || ty.x > p.scrW - r * 0.8f + 0.5f || ty.y > p.floorY || Float.isNaN(ty.x)) inside = false;
            if (p.x < -p.w * 0.15f - 0.5f || p.x > p.scrW - p.w * 0.85f + 0.5f) inside = false;
        }
        check(inside, "toy and pet stay on screen");
        check(p.toyKick >= 4, "ball kicked " + p.toyKick);
        p.pokeToy(); check(p.toy.vy < 0, "poke bounces");
        p.startHide(); check(!p.toyActive(), "hide pauses toy");
        p.endHide(false);
        p.putAwayToy(false); check(p.toy == null, "put away");

        // ── 풍선: 머리·발로 통통 ──
        p = fresh(51); run(p, 3, now); p.sayText = null;
        p.startToy("balloon");
        run(p, 40, now);
        check(p.toyKick >= 3, "balloon bounced " + p.toyKick);

        // ── 선물: 계절, 줍기, 사라짐, 저장 ──
        p = fresh(52); run(p, 3, now); p.sayText = null;
        Pet.Gift g = p.dropGiftForTest(0);
        b0 = p.bananas;
        p.collectGift(g);
        check(p.giftList.isEmpty() && p.giftCount("banana") == 1 && p.bananas == b0 + 2 && p.giftCnt == 1, "gift collect " + p.gifts);
        p.dropGiftForTest(3).age = 41 * 60; run(p, 0.1f, now);
        check(p.giftList.isEmpty(), "old gift gone");
        int autumn = 0;
        for (int i = 0; i < 2000; i++) { int k = p.pickGiftForTest(10); if (k >= 3 && k <= 5) autumn++; }
        check(autumn > 600, "autumn leaves " + autumn);
        p.setMode("walk", 5); p.y = p.floorY - p.h;
        p.nextGiftIn = 0.01f; p.tick(1 / 60f, now, 14, 30);
        check(p.giftList.size() == 1, "gift dropped");

        // ── 바나나 탑 ──
        p = fresh(53); run(p, 3, now);
        b0 = p.bananas;
        check(p.towerScore(12) == 7 && p.towerBest == 12 && p.mode.equals("laugh"), "tower 12");
        check(p.towerScore(40) == 15 && p.towerBest == 40, "tower cap");
        check(p.towerScore(0) == 0 && p.towerBest == 40, "tower 0");
        check(p.bananas == b0 + 22, "tower bananas");

        // ── 씻기기: 문지르면 깨끗해짐 ──
        p = fresh(54); run(p, 3, now); p.sayText = null;
        p.dirt = 80;
        check(p.wearing("idle").contains("dirt"), "dirty overlay");
        p.startBath();
        check(p.bathing && p.mode.equals("bath") && p.busy(), "bath start");
        for (int i = 0; i < 40 && p.bathing; i++) { p.rub(0.5f, 0.5f); run(p, 0.05f, now); }
        check(!p.bathing && p.dirt == 0 && p.bathCnt == 1 && p.mode.equals("laugh"), "bath done dirt=" + p.dirt);
        check(!p.wearing("idle").contains("dirt"), "clean overlay");

        // ── 이불 덮어 재우기 ──
        p = fresh(55); run(p, 3, now); p.sayText = null;
        p.tired = 90;
        p.tuckIn();
        check(p.tucked && p.mode.equals("sleep"), "tucked");
        p.tap(); check(p.tucked, "poke early keeps sleeping");
        run(p, p.modeDur + 1, now);
        check(!p.tucked && p.tired == 0 && p.tuckCnt == 1, "woke from tuck " + p.mode);

        // ── 집중 타이머 ──
        p = fresh(56); run(p, 3, now); p.sayText = null;
        p.startFocus(25, now);
        check(p.focusing() && !p.canPlay(), "focus on");
        b0 = p.bananas;
        run(p, 30, now);
        check(p.mode.equals("sit") || p.mode.equals("idle"), "sits during focus " + p.mode);
        run(p, 0.1f, now + 25 * 60000L + 1);
        check(!p.focusing() && p.bananas >= b0 + 5 && p.focusCnt == 1, "focus done");

        // ── 업적: 달성하면 하나씩 축하 + 저장 ──
        p = fresh(57); run(p, 3, now); p.sayText = null;
        p.snackCnt = 1; b0 = p.bananas;
        run(p, 2.5f, now);
        check(p.hasAch("snack1") && p.bananas == b0 + 3, "ach snack1 " + p.ach);
        st = new MapStore(); p.toyKick = 7; p.gifts = "banana:2,maple:1"; p.save(st, now);
        q = new Pet(new Random(1)); q.load(st, now);
        check(q.hasAch("snack1") && q.toyKick == 7 && q.giftCount("maple") == 1 && q.giftKindsGot() == 2, "fun saved");

        // ── 이모티콘: 메뉴로 보여 주고 끝나면 서 있기, 연타하면 흥! ──
        p = fresh(58); run(p, 3, now); p.sayText = null;
        for (String[] e : Pet.EMOS) {
            p.setMode("idle", 1);
            check(p.playEmo(e[0]) && p.mode.equals(e[0]) && p.seqName().equals(e[0]), "emo " + e[0]);
        }
        p.setMode("dance", 0.1f); run(p, 0.2f, now);
        check(!p.mode.equals("dance"), "dance ends " + p.mode);
        for (int i = 0; i < 5; i++) { p.setMode("idle", 5); p.tap(); }
        check(p.mode.equals("angry"), "tap spam angry " + p.mode);
        p.hiding = true; p.setMode("idle", 5);
        check(!p.playEmo("clap") && !p.mode.equals("clap"), "no emo while hiding");
        p.hiding = false;
        for (String[] e : Pet.EMOS) { p.setMode(e[0], 1); check(p.fps() > 0, "fps " + e[0]); }

        // ── 1.2: 바나나 요리 · 요리 도감 · 요리사 모자 ──
        p = fresh(61); run(p, 3, now); p.sayText = null;
        p.full = 40; p.bananas = 2; p.cook();
        check(p.cookCnt == 0 && p.bananas == 2, "cook needs bananas");
        check(Pet.pickRecipe(0) == 0 && Pet.pickRecipe(0.9999f) == Pet.COOK_IDS.length - 1, "recipe range");
        p.bananas = 100;
        for (int i = 0; i < 10; i++) { p.full = 40; p.setMode("idle", 1); p.cook(); }
        check(p.cookCnt == 10 && p.bananas == 100 - 10 * Pet.COOK_COST && p.cookedKinds() >= 1, "cook 10 " + p.cooked);
        for (int i = 0; i < 8; i++) { p.sayText = null; run(p, 2.5f, now); }
        check(p.hasAch("cook1") && p.hasAch("cook10") && p.owns("chefhat") && p.wear[0].equals("chefhat"), "chef hat " + p.ach + " / " + p.owned);
        st = new MapStore(); p.save(st, now);
        q = new Pet(new Random(1)); q.load(st, now);
        check(q.cooked.equals(p.cooked) && q.cookCnt == 10 && q.owns("chefhat"), "cook saved");

        // ── 1.2: 두더지 잡기 점수 ──
        p = fresh(62); run(p, 3, now);
        b0 = p.bananas;
        int ban = p.whackScore(999);
        check(p.whackBest == Pet.WHACK_MAX && ban == 15 && p.bananas == b0 + 15 && p.mode.equals("clap"), "whack score");
        check(p.whackScore(0) == 0 && p.whackCnt == 2, "whack zero");

        // ── 1.2: 예전에 깬 업적 보상은 업데이트 뒤 자동으로 받음 ──
        st = new MapStore(); p = fresh(63); p.ach = "play30,attend30"; p.owned = "starpin"; p.save(st, now);
        q = new Pet(new Random(1)); q.load(st, now);
        check(q.owns("medal") && q.owns("rainbow") && q.owns("starpin") && !q.owns("headphones") && q.ownedCount() == 3, "reward backfill " + q.owned);
        q.toggleWear(Pet.itemIndex("medal"));
        check(q.wear[2].equals("medal"), "wear medal");
        for (String[] ac : Pet.ACH) if (ac.length > 5) check(Pet.itemIndex(ac[5]) >= 0, "reward item " + ac[5]);

        // ── 1.2: 음악이 5초 넘게 나오면 춤 (한 번 추면 한동안 쉼) ──
        p = fresh(64); run(p, 3, now); p.sayText = null;
        p.setMode("idle", 30); p.musicActive = true;
        run(p, 3, now);
        check(!p.mode.equals("dance"), "short music no dance");
        run(p, 2.5f, now);
        check(p.mode.equals("dance") && p.danceCnt == 1, "music dance " + p.mode);
        p.setMode("idle", 30); run(p, 6, now);
        check(p.danceCnt == 1, "dance cooldown");
        p.musicActive = false;

        // ── 1.2: AI 대화 ──
        String[] se = Pet.splitEmo("안녕! 바나나 먹자~ [동작:v]");
        check(se[0].equals("안녕! 바나나 먹자~") && se[1].equals("v"), "ai split " + se[0] + "/" + se[1]);
        se = Pet.splitEmo("헤헤 [동작:rm_rf]");
        check(se[0].equals("헤헤") && se[1].isEmpty(), "ai unknown emo");
        check(Pet.splitEmo(new String(new char[300]).replace('\0', '가'))[0].length() <= 161, "ai long reply");
        p = fresh(65); p.name = "민지"; run(p, 3, now); p.sayText = null;
        String sys = p.aiSystem(now, 9, 5);
        check(sys.contains("\"민지\"") && sys.contains("[동작:이름]") && sys.contains("09시 05분") && sys.contains("st_bow"), "ai system");
        p.setMode("idle", 30); p.aiReply("나도 좋아!", "heart");
        check(p.mode.equals("heart") && p.chatCnt == 1 && "나도 좋아!".equals(p.sayText), "ai reply " + p.mode);
        st = new MapStore(); p.save(st, now); q = new Pet(new Random(1)); q.load(st, now);
        check(q.chatCnt == 1 && q.achDone("chat1"), "ai chatCnt saved");

        // ── 1.2: 꾸미기 상점 · 코디 ──
        check(Pet.ITEM_IDS.length == Pet.ITEM_NAMES.length && Pet.ITEM_IDS.length == Pet.ITEM_PRICE.length
                && Pet.ITEM_IDS.length == Pet.ITEM_LINE.length, "item arrays same length");
        for (String[] ac : Pet.ACH) if (ac.length > 5) check(Pet.itemIndex(ac[5]) >= 0, "ach item " + ac[5]);
        for (String gp : Pet.GACHA_POOL) check(Pet.itemIndex(gp) >= 0 && Pet.ITEM_PRICE[Pet.itemIndex(gp)] == 0, "gacha item " + gp);
        for (int i = 0; i < Pet.ITEM_IDS.length; i++) if (Pet.ITEM_PRICE[i] == 0) check(!Pet.giftFrom(Pet.ITEM_IDS[i]).isEmpty(), "giftFrom " + Pet.ITEM_IDS[i]);
        p = fresh(66); run(p, 3, now); p.bananas = 30;
        int crown = Pet.itemIndex("crown"), cat = Pet.itemIndex("catears"), blush = Pet.itemIndex("blush"), pearl = Pet.itemIndex("pearl");
        check(!p.buyItem(crown) && !p.owns("crown") && p.bananas == 30, "can't buy crown with 30");
        check(!p.buyItem(Pet.itemIndex("halo")) && !p.owns("halo"), "gift item not sold");
        p.bananas = 200;
        check(p.buyItem(cat) && p.owns("catears") && p.wear[0].equals("catears") && p.bananas == 182, "buy catears " + p.bananas);
        check(p.buyItem(blush) && p.buyItem(pearl) && p.wear[1].equals("blush") && p.wear[2].equals("pearl"), "buy blush pearl");
        p.saveOutfit(1);
        check(Pet.outfitLabel(p.outfit(1)).equals("고양이 귀 · 발그레 볼터치 · 진주 목걸이") && p.hasOutfit(), "outfit saved " + p.outfits);
        check(Pet.outfitLabel(p.outfit(0)).isEmpty() && Pet.outfitLabel(p.outfit(2)).isEmpty(), "other outfits empty");
        p.takeOffAll();
        check(Pet.outfitLabel(p.wear).isEmpty(), "take off all");
        p.wearOutfit(1);
        check(p.wear[0].equals("catears") && p.wear[2].equals("pearl"), "wear outfit");
        p.wearOutfit(2);
        check(p.wear[0].equals("catears"), "empty outfit keeps clothes");
        for (int i = 0; i < 30; i++) {
            p.randomOutfit();
            check(!Pet.outfitLabel(p.wear).isEmpty(), "random not empty");
            for (String w : p.wear) check(w.isEmpty() || p.owns(w), "random owns " + w);
        }
        check(p.randomCnt == 30 && p.achDone("random10") && p.achDone("outfit1") && p.achDone("items3") && !p.achDone("items15"), "deco ach");
        st = new MapStore(); p.save(st, now); q = new Pet(new Random(1)); q.load(st, now);
        check(q.outfits.equals(p.outfits) && q.randomCnt == 30 && q.owns("pearl"), "outfits saved");
        // 예전 기록 (outfits 키 없음) 도 그대로 열림
        q = fresh(67);
        check(q.outfits.isEmpty() && !q.hasOutfit() && Pet.outfitLabel(q.outfit(0)).isEmpty(), "old save ok");
        // 옷장 부자: 15개 모으면 티아라
        p.bananas = 2000;
        for (int i = 0; i < Pet.ITEM_IDS.length; i++) if (Pet.ITEM_PRICE[i] > 0 && !p.owns(Pet.ITEM_IDS[i])) p.buyItem(i);
        check(p.ownsAllShop() && p.achDone("itemsAll") && p.achDone("items15"), "all shop");
        p.sayText = null; p.setMode("idle", 30);
        for (int i = 0; i < 40 && !p.owns("tiara"); i++) { p.sayText = null; run(p, 2.1f, now); }
        check(p.owns("tiara"), "tiara from items15");

        if (fails == 0) System.out.println("ok: all Pet tests passed");
        else { System.out.println(fails + " failed"); System.exit(1); }
    }
}
