"""우땅이 표정 머리 만들기

원본 머리(assets/head_full.png)에서 검은 눈·입을 inpaint 로 지우고
새 표정을 그려 assets/ 에 저장합니다.
  head_eat0 / head_eat1 (먹기), head_shy (수줍음), head_sad (벌받기),
  head_blink (깜빡), head_sleep (잠)

    python tools/heads.py
"""
import numpy as np, math
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter
A = Path(__file__).resolve().parent.parent / "assets"
h=np.asarray(Image.open(A/'head_full.png')).copy();rgb=h[...,:3].astype(int);al=h[...,3]
K=(10,10,10,255)
EYES=[(415,596,286,468),(429,587,694,850)]
MOUTH=(708,896,366,792)
def cover(im,box,grow=16,blur=20):
    """검은 눈/입만 지우고 주변 흰 번짐으로 자연스럽게 메우기 (inpaint)"""
    import cv2
    y0,y1,x0,x1=box
    arr=np.asarray(im).copy()
    rgb_=cv2.cvtColor(arr[...,:3],cv2.COLOR_RGB2BGR)
    P=70
    sub=rgb_[max(y0-P,0):y1+P,max(x0-P,0):x1+P]
    g=cv2.cvtColor(sub,cv2.COLOR_BGR2GRAY)
    m=(g<200).astype(np.uint8)*255           # 검은 선 + 반사광 주변
    yy,xx=np.mgrid[0:sub.shape[0],0:sub.shape[1]]
    cy,cx=(y1-y0)/2+(y0-max(y0-P,0)),(x1-x0)/2+(x0-max(x0-P,0))
    inside=((xx-cx)/((x1-x0)/2+grow))**2+((yy-cy)/((y1-y0)/2+grow*0.6))**2<=1
    m=np.where(inside,m,0).astype(np.uint8)
    m=cv2.dilate(m,np.ones((9,9),np.uint8))
    # 반사광(흰 점)도 눈 안쪽이면 같이 지움
    m=np.where(inside&(cv2.dilate(m,np.ones((15,15),np.uint8))>0)&((((xx-cx)/((x1-x0)/2+4))**2+((yy-cy)/((y1-y0)/2+4))**2)<=1),255,m).astype(np.uint8)
    out=cv2.inpaint(sub,m,25,cv2.INPAINT_TELEA)
    out=cv2.GaussianBlur(out,(0,0),6)*(m[...,None]>0)+out*(m[...,None]==0)
    rgb_[max(y0-P,0):y1+P,max(x0-P,0):x1+P]=out.astype(np.uint8)
    arr[...,:3]=cv2.cvtColor(rgb_,cv2.COLOR_BGR2RGB)
    return Image.fromarray(arr,'RGBA')
def curve(d,pts,w=30):
    d.line(pts,fill=K,width=w,joint='curve')
    for p in (pts[0],pts[-1]): d.ellipse((p[0]-w/2,p[1]-w/2,p[0]+w/2,p[1]+w/2),fill=K)
def arc(cx,cy,L,amp,n=24):  # amp>0: 아래로 볼록, <0: 위로 볼록
    return [(cx-L+2*L*t/n, cy+amp*math.sin(math.pi*t/n)) for t in range(n+1)]
def happy_eyes(im):
    for b in EYES: im=cover(im,b)
    d=ImageDraw.Draw(im)
    for (y0,y1,x0,x1) in EYES:
        cx,cy=(x0+x1)/2,(y0+y1)/2; curve(d,arc(cx,cy+18,62,-46))
    return im
def blush(im,a=150):
    lay=Image.new('RGBA',im.size,(0,0,0,0));d=ImageDraw.Draw(lay)
    for cx in (215,935): d.ellipse((cx-85,640,cx+85,725),fill=(242,135,140,a))
    lay=lay.filter(ImageFilter.GaussianBlur(10));im=im.copy();im.alpha_composite(lay);return im
def finish(im):
    im.putalpha(Image.fromarray(al));return im
base=Image.fromarray(h,'RGBA')
# 먹기: ^^ 눈 + 볼터치 + 오물오물 입(닫힘/벌림)
e=blush(happy_eyes(base))
m0=cover(e,MOUTH,55,22);d=ImageDraw.Draw(m0)
curve(d,arc(579,790,70,38),30)                    # 오물 (닫힌 미소)
finish(m0).save(A/'head_eat0.png')
m1=cover(e,MOUTH,55,22);d=ImageDraw.Draw(m1)
d.ellipse((520,742,640,850),fill=K)                # 냠 (작게 벌린 입)
d.ellipse((555,800,605,836),fill=(220,110,115,255))
finish(m1).save(A/'head_eat1.png')
# 수줍음: ^^ 눈 + 진한 볼터치 + 작은 미소
s=blush(blush(happy_eyes(base),190),120)
s=cover(s,MOUTH,55,22);d=ImageDraw.Draw(s)
curve(d,arc(579,780,48,30),28)
finish(s).save(A/'head_shy.png')
# 벌받기: 원래 눈 + 처진 눈썹 + 삐죽 입 + 눈물
sd=cover(base,MOUTH,55,22);d=ImageDraw.Draw(sd)
pts=[(500+i*8, 812+10*math.sin(i*0.9)) for i in range(20)]
curve(d,pts,26)
curve(d,[(300,395),(420,360)],24); curve(d,[(860,395),(745,360)],24)
lay=Image.new('RGBA',sd.size,(0,0,0,0));dl=ImageDraw.Draw(lay)
for x in (330,820):
    dl.polygon([(x,600),(x-26,670),(x,700),(x+26,670)],fill=(120,200,255,235));dl.ellipse((x-26,645,x+26,702),fill=(120,200,255,235))
sd.alpha_composite(lay)
finish(sd).save(A/'head_sad.png')
# 깜빡/잠든 눈도 같은 방식으로
for kind,amp in (('blink',8),('sleep',34)):
    im=base
    for b in EYES: im=cover(im,b)
    d=ImageDraw.Draw(im)
    for (y0,y1,x0,x1) in EYES:
        cx,cy=(x0+x1)/2,(y0+y1)/2; curve(d,arc(cx,cy+6,58,amp))
    finish(im).save(A/f'head_{kind}.png')
print('heads: done ->', A)
