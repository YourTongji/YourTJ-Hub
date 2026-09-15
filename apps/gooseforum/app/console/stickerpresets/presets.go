// Package stickerpresets embeds the built-in preset sticker pack used by the
// seed-stickers command. Sources and licenses are listed in
// preset_stickers/NOTICE.md (flowerhd CC BY 4.0, EmojiPackage Apache-2.0,
// WXMemeStickers MIT) — the NOTICE must stay embedded with the binary.
package stickerpresets

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
)

//go:embed preset_stickers
var presetFS embed.FS

// Preset is one manifest entry: name is the global sticker token name,
// file is the preset_stickers-relative image path, pack is the source pack.
type Preset struct {
	Name string `json:"name"`
	File string `json:"file"`
	Pack string `json:"pack"`
}

const manifestPath = "preset_stickers/manifest.json"

// Manifest parses the embedded preset manifest.
func Manifest() ([]Preset, error) {
	data, err := presetFS.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read preset manifest: %w", err)
	}
	var presets []Preset
	if err := json.Unmarshal(data, &presets); err != nil {
		return nil, fmt.Errorf("parse preset manifest: %w", err)
	}
	return presets, nil
}

// Load reads one preset image from the embedded filesystem.
func Load(file string) ([]byte, error) {
	return presetFS.ReadFile(path.Join("preset_stickers", file))
}

// NOTICE returns the bundled attribution notice text.
func NOTICE() ([]byte, error) {
	return presetFS.ReadFile("preset_stickers/NOTICE.md")
}
