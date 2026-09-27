// Package stickerpresets embeds the built-in preset sticker pack used by the
// seed-stickers command. Sources and licenses are listed in
// preset_stickers/NOTICE.md (flowerhd CC BY 4.0, EmojiPackage Apache-2.0,
// WXMemeStickers MIT) — the NOTICE must stay embedded with the binary.
package stickerpresets

import (
	"embed"
	"path"
)

//go:embed preset_stickers
var presetFS embed.FS

// Load reads one preset image from the embedded filesystem.
func Load(file string) ([]byte, error) {
	return presetFS.ReadFile(path.Join("preset_stickers", file))
}
