package server

import "embed"

// WebFS 嵌入 web 前端构建产物（见 ADR-0001：部署合并单二进制）。
// 注意：embed 路径相对本文件目录（module 根），且不能包含 ".."，
// 因此必须放在这里而不是 cmd/server/ 下。
// all: 前缀确保 fresh clone 时目录内仅有 .gitkeep 也能编译。
//
//go:embed all:webdist
var WebFS embed.FS
