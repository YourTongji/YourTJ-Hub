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

// NicknameContrastMin is WCAG's large-text floor (3:1), not the 4.5:1 ordinary-text one.
// Web (NICKNAME_CONTRAST_MIN) and Flutter (nicknameContrastMin) mirror this value.
const NicknameContrastMin = 3.0

type Style = decoration.Definition

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
var decorationAssetPattern = regexp.MustCompile(`^/static/decorations/[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.(png|webp|gif|svg)$`)

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

// readable grades the gradient's endpoint stops only; the ramp between them is not sampled (#1312).
func readable(colors, backgrounds []string) bool {
	for _, color := range []string{colors[0], colors[len(colors)-1]} {
		fg := luminance(rgb(color))
		for _, background := range backgrounds {
			bg := luminance(rgb(background))
			if (math.Max(fg, bg)+0.05)/(math.Min(fg, bg)+0.05) < NicknameContrastMin {
				return false
			}
		}
	}
	return true
}
