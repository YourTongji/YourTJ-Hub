// Package stickerpresets owns the built-in sticker catalog shared by seeding
// and upgrades. Images and license notices remain embedded by console/stickerpresets.
package stickerpresets

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed manifest.json
var manifest []byte

// Preset identifies a global token, its embedded image, and source pack.
// File is relative to console/stickerpresets/preset_stickers.
type Preset struct {
	Name string `json:"name"`
	File string `json:"file"`
	Pack string `json:"pack"`
}

func Manifest() ([]Preset, error) {
	var presets []Preset
	if err := json.Unmarshal(manifest, &presets); err != nil {
		return nil, fmt.Errorf("parse preset manifest: %w", err)
	}
	return presets, nil
}
