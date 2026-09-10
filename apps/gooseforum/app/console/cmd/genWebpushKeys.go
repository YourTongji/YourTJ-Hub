package cmd

import (
	"fmt"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "webpush-keys",
		Short: "Generate a VAPID key pair for the [webpush] config section",
		RunE:  runGenWebpushKeys,
	}
	appendCommand(cmd)
}

// runGenWebpushKeys 生成 VAPID 密钥对并打印配置值。
// 私钥是实例级 secret：写入 config.toml 的 [webpush] 段（本地），
// 部署实例经 GitHub Environments secret VAPID_PRIVATE_KEY 注入，
// 永不提交 git、不进 DB。
func runGenWebpushKeys(_ *cobra.Command, _ []string) error {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return fmt.Errorf("generate VAPID keys: %w", err)
	}
	fmt.Println("Web Push (VAPID) key pair generated. Add to config.toml:")
	fmt.Println()
	fmt.Println("[webpush]")
	fmt.Printf("vapid_public_key = %q\n", publicKey)
	fmt.Printf("vapid_private_key = %q\n", privateKey)
	fmt.Println()
	fmt.Println("Deployment: set GitHub Environments secrets VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY,")
	fmt.Println("then render config.toml via deploy/render_config.py (see docs/operations/deployment.md).")
	return nil
}
