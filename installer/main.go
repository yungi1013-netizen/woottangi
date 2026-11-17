package main

// 우땅이 설치 프로그램
//
//   Woottangi_Setup.exe             설치 화면
//   Woottangi_Setup.exe /S          조용히 설치 (바탕화면 바로가기 O, 자동 실행은 지금 설정 유지)
//   uninstall.exe /uninstall        삭제 (설치 폴더 안에 자동으로 생김. 설정 > 앱 에서도 실행됨)
//   uninstall.exe /uninstall /S     조용히 삭제 (저장 파일은 남김, /PURGE 를 붙이면 저장 파일도 삭제)

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed payload/Woottangi.exe
var payload []byte

//go:embed art.png
var artPNG []byte

func main() {
	runtime.LockOSThread()
	pSetProcessDPIAware.Call()
	args := map[string]bool{}
	for _, a := range os.Args[1:] {
		args[strings.ToUpper(a)] = true
	}
	silent := args["/S"]
	p := defaultPaths()
	self, _ := os.Executable()

	if args["/UNINSTALL"] || strings.EqualFold(filepath.Base(self), uninstName) {
		runUninstall(p, silent, args["/PURGE"])
		return
	}
	if silent {
		o := options{desktop: true, autostart: p.autostartOn()}
		if err := install(p, o, payload); err != nil {
			os.Exit(1)
		}
		return
	}
	runSetupWindow(p)
}

func runUninstall(p paths, silent, purge bool) {
	if silent {
		if uninstall(p, purge) != nil {
			os.Exit(1)
		}
		return
	}
	if msgBox(0, "우땅이를 이 컴퓨터에서 삭제할까요?", mbYesNo|mbIconQuest) != idYes {
		return
	}
	removeData := msgBox(0, "우땅이의 이름·바나나·레벨·업적 기록도 함께 지울까요?\n\n"+
		"[아니요]를 누르면 기록은 남아서, 나중에 다시 설치했을 때 그대로 이어서 키울 수 있어요.",
		mbYesNo|mbIconQuest|mbDefButton2) == idYes
	if err := uninstall(p, removeData); err != nil {
		msgBox(0, "삭제하지 못했어요.\n\n"+err.Error(), mbOK|mbIconError)
		return
	}
	msgBox(0, "우땅이를 삭제했어요. 그동안 고마웠어! 꾸벅", mbOK|mbIconInfo)
}
