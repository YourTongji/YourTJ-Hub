package wikiservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
)

// DeleteNamespace 删除 namespace（贡献者记录清理失败必须中止删除，
// review：吞错会留下孤儿贡献者行）。
func DeleteNamespace(name string) error {
	if err := wikiNamespaceEditors.DeleteByNamespace(name); err != nil {
		return err
	}
	return wikiNamespaces.DeleteByName(name)
}
