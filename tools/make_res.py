"""윈도우 리소스(.syso) 만들기: 아이콘 + DPI 매니페스트 + 버전정보(1.0, 제작 K)

    python tools/make_res.py frames/idle_0.png pc/rsrc_windows_amd64.syso [파일설명] [원래파일이름]
Go 빌드할 때 같은 폴더의 .syso 가 자동으로 exe 에 들어갑니다.
"""
import struct,io,sys
from PIL import Image
src=sys.argv[1]
dst=sys.argv[2] if len(sys.argv)>2 else 'rsrc_windows_amd64.syso'
DESC=sys.argv[3] if len(sys.argv)>3 else "우땅이"            # 파일 설명 (설치 프로그램은 "우땅이 설치")
ORIG=sys.argv[4] if len(sys.argv)>4 else "Woottangi.exe"     # 원래 파일 이름
import os
APPVER=os.environ.get("WOO_VERSION","1.1")                 # build.py 의 VERSION
_mj,_mn=(int(x) for x in (APPVER.split(".")+["0"])[:2])
im=Image.open(src).convert('RGBA');w,h=im.size;S=max(w,h)
sq=Image.new('RGBA',(S,S),(0,0,0,0));sq.paste(im,((S-w)//2,(S-h)//2))
ico=sq.resize((256,256),Image.LANCZOS)
b=io.BytesIO();ico.save(b,'ICO',sizes=[(16,16),(24,24),(32,32),(48,48),(64,64),(128,128),(256,256)]);d=b.getvalue()
_,_,n=struct.unpack('<HHH',d[:6]);ents=[]
for i in range(n):
    w_,h_,cc,r,pl,bc,sz,off=struct.unpack('<BBBBHHII',d[6+16*i:22+16*i]);ents.append((w_,h_,cc,r,pl,bc,d[off:off+sz]))
grp=struct.pack('<HHH',0,1,n)+b''.join(struct.pack('<BBBBHHIH',a,b_,cc,r,pl,bc,len(img),i+1) for i,(a,b_,cc,r,pl,bc,img) in enumerate(ents))
manifest=b'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
<assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="K.Woottangi" type="win32"/>
<dependency><dependentAssembly><assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/></dependentAssembly></dependency>
<application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings>
<dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true</dpiAware>
</windowsSettings></application>
</assembly>'''

def _pad(b):
    return b + b'\0' * ((4 - len(b) % 4) % 4)
def _blk(key, val=b'', typ=1, children=b'', vlen=None):
    k = (key + '\0').encode('utf-16-le')
    hdr_len = 6 + len(k)
    body = _pad(b'\0' * 6 + k)[6:]
    data = _pad(val) if val else b''
    if vlen is None:
        vlen = len(val) // 2 if typ == 1 else len(val)
    total = 6 + len(body) + len(data) + len(children)
    return struct.pack('<HHH', total, vlen, typ) + body + data + children
def _s(key, v):
    return _pad(_blk(key, (v + '\0').encode('utf-16-le')))
VER = "1.0.0.0"
strs = b''.join(_s(k, v) for k, v in [("CompanyName", "K"), ("FileDescription", DESC), ("FileVersion", APPVER),
    ("InternalName", "Woottangi"), ("OriginalFilename", ORIG), ("ProductName", "우땅이"), ("ProductVersion", APPVER)])
st = _blk("041204b0", typ=1, children=strs, vlen=0)
sfi = _blk("StringFileInfo", typ=1, children=st, vlen=0)
var = _blk("Translation", struct.pack('<HH', 0x0412, 0x04b0), typ=0)
vfi = _blk("VarFileInfo", typ=1, children=_pad(var), vlen=0)
ffi = struct.pack('<13I', 0xFEEF04BD, 0x00010000, (_mj<<16)|_mn, 0, (_mj<<16)|_mn, 0, 0x3F, 0, 0x40004, 1, 0, 0, 0)
VERSION_RES = _blk("VS_VERSION_INFO", ffi, typ=0, children=_pad(sfi) + _pad(vfi), vlen=len(ffi))

types=[(3,[(i+1,e[6]) for i,e in enumerate(ents)]),(14,[(1,grp)]),(16,[(1,VERSION_RES)]),(24,[(1,manifest)])]
def dirhdr(k): return struct.pack('<IIHHHH',0,0,0,0,0,k)
root_size=16+8*len(types);type_sizes=[16+8*len(items) for _,items in types]
leaves=[it for _,items in types for it in items];lang_size=24
off=root_size;type_offs=[]
for s in type_sizes: type_offs.append(off);off+=s
lang_offs=[]
for _ in leaves: lang_offs.append(off);off+=lang_size
de_offs=[]
for _ in leaves: de_offs.append(off);off+=16
data_offs=[]
for _,blob in leaves: off=(off+7)&~7;data_offs.append(off);off+=len(blob)
total=(off+7)&~7;raw=bytearray(total);relocs=[]
raw[0:root_size]=dirhdr(len(types))+b''.join(struct.pack('<II',t,0x80000000|type_offs[i]) for i,(t,_) in enumerate(types))
li=0
for i,(t,items) in enumerate(types):
    ent=b''
    for (rid,_) in items: ent+=struct.pack('<II',rid,0x80000000|lang_offs[li]);li+=1
    raw[type_offs[i]:type_offs[i]+type_sizes[i]]=dirhdr(len(items))+ent
for k,(rid,blob) in enumerate(leaves):
    raw[lang_offs[k]:lang_offs[k]+lang_size]=dirhdr(1)+struct.pack('<II',0x0409,de_offs[k])
    raw[de_offs[k]:de_offs[k]+16]=struct.pack('<IIII',data_offs[k],len(blob),0,0);relocs.append(de_offs[k])
    raw[data_offs[k]:data_offs[k]+len(blob)]=blob
ptr_raw=60;ptr_rel=ptr_raw+len(raw);ptr_sym=ptr_rel+10*len(relocs)
hdr=struct.pack('<HHIIIHH',0x8664,1,0,ptr_sym,1,0,0)
sec=struct.pack('<8sIIIIIIHHI',b'.rsrc',0,0,len(raw),ptr_raw,ptr_rel,0,len(relocs),0,0x40000040)
rel=b''.join(struct.pack('<IIH',r,0,3) for r in relocs)
sym=struct.pack('<8sIhHBB',b'.rsrc',0,1,0,3,0)
open(dst,'wb').write(hdr+sec+bytes(raw)+rel+sym+struct.pack('<I',4))
ico.save('icon_preview.png')
