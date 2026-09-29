// goMail-injector — 基于 github.com/wneessen/go-mail 的邮件注入程序
// 功能与 pmta-injector / PHPMailer-injector 对齐：同一套 config.yaml、模板、头策略与调度；
// SMTP 传输层使用 go-mail（TLS/AUTH），MIME 仍由本仓库 email.Builder 构建以保留反指纹细节。
package main

import (
	"fmt"
	"os"

	"__MODULE_PLACEHOLDER__/cmd"
)

var (
	Version   = "__VERSION_PLACEHOLDER__"
	BuildTime = "__BUILDTIME_PLACEHOLDER__"
	GitCommit = "__GITCOMMIT_PLACEHOLDER__"
)

func main() {
	cmd.SetVersionInfo(Version, BuildTime, GitCommit)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
