package com.woottangi.pet;

import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Paint;
import android.graphics.Path;
import android.graphics.RectF;
import android.text.Layout;
import android.text.StaticLayout;
import android.text.TextPaint;
import android.view.View;

/** 말풍선 (아래쪽에 꼬리) */
public class BubbleView extends View {
    private final float dp;
    private final TextPaint tp = new TextPaint(Paint.ANTI_ALIAS_FLAG);
    private final Paint fill = new Paint(Paint.ANTI_ALIAS_FLAG), line = new Paint(Paint.ANTI_ALIAS_FLAG);
    private StaticLayout lay;
    private final RectF r = new RectF();
    private final Path tail = new Path();
    float tailX = -1; // 꼬리 위치 (뷰 안 x), -1 이면 가운데

    public BubbleView(Context c) {
        super(c);
        dp = c.getResources().getDisplayMetrics().density;
        tp.setColor(0xFF4A3520);
        tp.setTextSize(14 * dp);
        tp.setFakeBoldText(true);
        fill.setColor(0xFFFFFDF5);
        line.setColor(0xFFE0B860);
        line.setStyle(Paint.Style.STROKE);
        line.setStrokeWidth(2 * dp);
    }

    @SuppressWarnings("deprecation")
    public void setText(String t) {
        int maxW = (int) (230 * dp);
        int want = 0;
        for (String l : t.split("\n")) want = Math.max(want, (int) Math.ceil(tp.measureText(l)));
        int w = Math.max((int) (40 * dp), Math.min(maxW, want));
        lay = new StaticLayout(t, tp, w, Layout.Alignment.ALIGN_CENTER, 1.1f, 0, false);
        requestLayout();
        invalidate();
    }

    float pad() { return 12 * dp; }

    float tailH() { return 9 * dp; }

    @Override protected void onMeasure(int ws, int hs) {
        if (lay == null) { setMeasuredDimension(1, 1); return; }
        setMeasuredDimension((int) (lay.getWidth() + pad() * 2 + 4 * dp), (int) (lay.getHeight() + pad() * 1.4f + tailH() + 4 * dp));
    }

    @Override protected void onDraw(Canvas c) {
        if (lay == null) return;
        float s = 2 * dp, w = getWidth(), h = getHeight() - tailH();
        r.set(s, s, w - s, h - s);
        float tx = tailX < 0 ? w / 2 : Math.max(18 * dp, Math.min(w - 18 * dp, tailX));
        tail.reset();
        tail.moveTo(tx - 8 * dp, h - s - 1);
        tail.lineTo(tx, getHeight() - s);
        tail.lineTo(tx + 8 * dp, h - s - 1);
        c.drawRoundRect(r, 14 * dp, 14 * dp, fill);
        c.drawRoundRect(r, 14 * dp, 14 * dp, line);
        c.drawPath(tail, fill);
        c.drawLine(tx - 8 * dp, h - s - 1, tx, getHeight() - s, line);
        c.drawLine(tx, getHeight() - s, tx + 8 * dp, h - s - 1, line);
        c.save();
        c.translate((w - lay.getWidth()) / 2f, pad() * 0.7f + s);
        lay.draw(c);
        c.restore();
    }
}
