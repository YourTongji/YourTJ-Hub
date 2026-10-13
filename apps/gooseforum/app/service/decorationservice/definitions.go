package decorationservice

import (
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/decoration"
)

const DefaultAssetScale = 1.25

const AvatarFrame = "avatarFrame"
const NicknameStyle = "nicknameStyle"

type Style = decoration.Definition

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
var decorationAssetPattern = regexp.MustCompile(`^/static/decorations/[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.(png|webp)$`)

func ValidateStyle(style Style) error {
	if math.IsNaN(style.AssetScale) || math.IsInf(style.AssetScale, 0) || (style.AssetScale != 0 && (style.AssetScale < 0.5 || style.AssetScale > 2)) {
		return ErrInvalid
	}
	if style.AssetURL != "" && (style.Kind != AvatarFrame || path.Clean(style.AssetURL) != style.AssetURL || !decorationAssetPattern.MatchString(style.AssetURL)) {
		return ErrInvalid
	}
	if !codePattern.MatchString(style.Code) || strings.TrimSpace(style.Name) == "" || utf8.RuneCountInString(style.Name) > 80 || (style.Kind != AvatarFrame && style.Kind != NicknameStyle) || style.SortOrder < 0 || style.SortOrder > 100000 {
		return ErrInvalid
	}
	for _, colors := range [][]string{style.LightColors, style.DarkColors} {
		if len(colors) < 1 || len(colors) > 2 {
			return ErrInvalid
		}
		for _, color := range colors {
			if !colorPattern.MatchString(color) {
				return ErrInvalid
			}
		}
	}
	if style.Kind == NicknameStyle && (!readable(style.LightColors, []string{"#FFFFFF", "#F3F4F6"}) || !readable(style.DarkColors, []string{"#111827", "#1F2937"})) {
		return ErrInvalid
	}
	return nil
}

func rgb(color string) [3]float64 {
	n, _ := strconv.ParseUint(color[1:], 16, 32)
	return [3]float64{float64(n >> 16), float64((n >> 8) & 255), float64(n & 255)}
}
func luminance(c [3]float64) float64 {
	for i, v := range c {
		v /= 255
		if v <= 0.04045 {
			c[i] = v / 12.92
		} else {
			c[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return c[0]*0.2126 + c[1]*0.7152 + c[2]*0.0722
}
func readable(colors, backgrounds []string) bool {
	a, b := rgb(colors[0]), rgb(colors[len(colors)-1])
	for step := 0; step <= 32; step++ {
		t := float64(step) / 32
		fg := luminance([3]float64{a[0]*(1-t) + b[0]*t, a[1]*(1-t) + b[1]*t, a[2]*(1-t) + b[2]*t})
		for _, background := range backgrounds {
			bg := luminance(rgb(background))
			if (math.Max(fg, bg)+0.05)/(math.Min(fg, bg)+0.05) < 4.5 {
				return false
			}
		}
	}
	return true
}
