package com.woottangi.pet;

import android.content.Context;
import android.graphics.Bitmap;
import android.graphics.Canvas;
import android.graphics.Matrix;
import android.graphics.Paint;
import android.graphics.Rect;
import android.view.View;

import java.util.ArrayList;
import java.util.List;

/** 우땅이(또는 땅콩이) 그림 한 장 + 꾸미기 아이템 그리기 (좌우 뒤집기 · 기울기 · 아래 자르기) */
public class PetView extends View {
    Bitmap bmp;
    boolean flip;
    float rot;
    float clipBottom = Float.NaN; // 뷰 안 y: 이 아래는 안 그림 (숨바꼭질)
    final List<Bitmap> accs = new ArrayList<>();
    float[] anchor; // 프레임(480x600) 좌표의 머리 위치 {x, y, 배율, 각도}
    int q; // 0 그대로 · 1 시계 90°(왼쪽 벽) · 2 180°(천장) · 3 반시계 90°(오른쪽 벽)
    private final Paint paint = new Paint(Paint.FILTER_BITMAP_FLAG | Paint.ANTI_ALIAS_FLAG);
    private final Rect dst = new Rect();
    private final Matrix m = new Matrix();
    // 이불 (덮고 잘 때) · 목욕 거품 {x, y (그림 비율), 크기 (그림 너비 비율), 투명도}
    Bitmap blanket, bubbleBmp;
    final List<float[]> bubbles = new ArrayList<>();
    private final Paint bp = new Paint(Paint.FILTER_BITMAP_FLAG | Paint.ANTI_ALIAS_FLAG);
    private final android.graphics.RectF bdst = new android.graphics.RectF();

    public PetView(Context c) { super(c); }

    /** 바뀐 게 있을 때만 다시 그림 */
    public void set(Bitmap b, boolean f, float r, List<Bitmap> acc, float[] anc, float clip, int quarter) {
        boolean same = quarter == q && b == bmp && f == flip && Math.abs(r - rot) < 0.004f && anc == anchor && acc.equals(accs)
                && (Float.isNaN(clip) ? Float.isNaN(clipBottom) : Math.abs(clip - clipBottom) < 1);
        if (same) return;
        bmp = b; flip = f; rot = r; anchor = anc; clipBottom = clip; q = quarter;
        accs.clear();
        accs.addAll(acc);
        invalidate();
    }

    /** 이불 · 거품 (거품이 있으면 매 프레임 다시 그림) */
    public void setExtras(Bitmap blk, Bitmap bub, List<float[]> bs) {
        if (blk == blanket && bs.isEmpty() && bubbles.isEmpty()) return;
        blanket = blk; bubbleBmp = bub;
        bubbles.clear();
        bubbles.addAll(bs);
        invalidate();
    }

    @Override protected void onDraw(Canvas c) {
        if (bmp == null) return;
        int vw = getWidth(), vh = getHeight();
        boolean side = q == 1 || q == 3;
        int w = side ? vh : vw, h = side ? vw : vh; // 그림 크기 (벽에서는 창의 가로세로가 바뀜)
        c.save();
        if (!Float.isNaN(clipBottom)) c.clipRect(0, 0, vw, clipBottom);
        if (q != 0) {
            c.translate(vw / 2f, vh / 2f);
            c.rotate(q == 1 ? 90 : q == 2 ? 180 : -90);
            c.translate(-w / 2f, -h / 2f);
        }
        c.rotate((float) Math.toDegrees(rot), w / 2f, h * 0.55f);
        if (flip) c.scale(-1, 1, w / 2f, 0);
        dst.set(0, 0, w, h);
        c.drawBitmap(bmp, null, dst, paint);
        if (anchor != null && !accs.isEmpty()) {
            float k = w / 480f;
            for (Bitmap a : accs) { // 웹 버전 placeAcc 와 같은 계산 (아이템 그림은 400x400, 머리 원점 200,330)
                float u = 400f / a.getWidth();
                m.setScale(u, u);
                m.postTranslate(-200, -330);
                m.postScale(anchor[2] * k, anchor[2] * k);
                m.postRotate(-anchor[3]);
                m.postTranslate(anchor[0] * k, anchor[1] * k);
                c.drawBitmap(a, m, paint);
            }
        }
        if (blanket != null) c.drawBitmap(blanket, null, dst, paint);
        if (bubbleBmp != null) {
            for (float[] b : bubbles) {
                float bx = b[0] * w, by = b[1] * h, r = b[2] * w / 2;
                if (flip) bx = w - bx;
                bdst.set(bx - r, by - r, bx + r, by + r);
                bp.setAlpha(Math.round(255 * Math.max(0, Math.min(1, b[3]))));
                c.drawBitmap(bubbleBmp, null, bdst, bp);
            }
        }
        c.restore();
    }
}
