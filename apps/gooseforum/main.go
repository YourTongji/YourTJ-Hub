// GooseForum is a self-hosted forum platform built with Go, Vue, and Tailwind CSS.
package main

import (
	_ "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/logging"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/console"
)

// --go:generate go run generatetool/generatetool.go
//
// -- go:generate npm run --prefix actor build --emptyOutDir
//
//go:generate pnpm --dir resource build
func main() {
	// 注册静态资源
	console.Execute()
}
