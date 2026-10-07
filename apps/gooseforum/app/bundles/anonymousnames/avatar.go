package anonymousnames

import (
	"fmt"
	"math"
)

// Avatar uses the course-review Beam geometry and palette with an independent
// persisted random seed. Geometry source: resource/src/site/utils/course-review-share.ts.
// Only geometry and colours enter the SVG, never names.
func Avatar(seed string) string {
	var hash int32
	for _, c := range seed {
		hash = hash*31 + c
	}
	n := int64(hash)
	if n < 0 {
		n = -n
	}
	digit := func(index int) int64 { return n / int64(math.Pow10(index)) % 10 }
	unit := func(size int64, index int) int64 {
		v := n % size
		if index != 0 && digit(index)%2 == 0 {
			v = -v
		}
		return v
	}
	colors := []string{"#0f172a", "#38bdf8", "#f8fafc", "#f59e0b", "#22c55e"}
	face := "#000000"
	if n%5 == 0 {
		face = "#FFFFFF"
	}
	x, y := unit(10, 1), unit(10, 2)
	if x < 5 {
		x += 4
	}
	if y < 5 {
		y += 4
	}
	fx, fy := float64(unit(8, 1)), float64(unit(7, 2))
	if x > 6 {
		fx = float64(x) / 2
	}
	if y > 6 {
		fy = float64(y) / 2
	}
	radius := 6
	if digit(1)%2 == 0 {
		radius = 18
	}
	mouth := fmt.Sprintf(`<path d="M13,%d a1,0.75 0 0,0 10,0" fill="%s"/>`, 19+n%3, face)
	if digit(2)%2 == 0 {
		mouth = fmt.Sprintf(`<path d="M15 %dc2 1 4 1 6 0" stroke="%s" fill="none" stroke-linecap="round"/>`, 19+n%3, face)
	}
	// Half-width radii match browser SVG clamping and avoid self-crossing
	// rounded-rect paths in mobile vector renderers.
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36" width="96" height="96"><defs><clipPath id="beam"><rect width="36" height="36" rx="18"/></clipPath></defs><g clip-path="url(#beam)"><rect width="36" height="36" fill="%s"/><rect width="36" height="36" fill="%s" rx="%d" transform="translate(%d %d) rotate(%d 18 18) scale(%g)"/><g transform="translate(%g %g) rotate(%d 18 18)">%s<rect x="%d" y="14" width="1.5" height="2" rx=".75" ry="1" fill="%s"/><rect x="%d" y="14" width="1.5" height="2" rx=".75" ry="1" fill="%s"/></g></g></svg>`, colors[(n+13)%5], colors[n%5], radius, x, y, n%360, 1+float64(n%3)/10, fx, fy, unit(10, 3), mouth, 14-n%5, face, 20+n%5, face)
}
