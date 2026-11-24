package com.woottangi.pet;

import android.content.Context;
import android.graphics.Bitmap;
import android.graphics.Canvas;
import android.graphics.Paint;
import android.graphics.RectF;
import android.view.View;

/** 화면 위 작은 물건 하나 (장난감 공·풍선, 선물): 그림 한 장을 돌려서 그림 */
public class ThingView extends View {
    Bitmap bmp;
    float rot, alpha = 1, scale = 0.9f;
    private final Paint paint = new Paint(Paint.FILTER_BITMAP_FLAG | Paint.ANTI_ALIAS_FLAG);
    private final RectF dst = new RectF();

    public ThingView(Context c) { super(c); }

    public void set(Bitmap b, float r, float a) {
        if (b == bmp && Math.abs(r - rot) < 0.004f && Math.abs(a - alpha) < 0.01f) return;
        bmp = b; rot = r; alpha = a;
        invalidate();
    }

    @Override protected void onDraw(Canvas c) {
        if (bmp == null) return;
        float w = getWidth(), h = getHeight(), s = Math.min(w, h) * scale;
        c.save();
        c.rotate((float) Math.toDegrees(rot), w / 2, h / 2);
        dst.set((w - s) / 2, (h - s) / 2, (w + s) / 2, (h + s) / 2);
        paint.setAlpha(Math.round(255 * Math.max(0, Math.min(1, alpha))));
        c.drawBitmap(bmp, null, dst, paint);
        c.restore();
    }
}
