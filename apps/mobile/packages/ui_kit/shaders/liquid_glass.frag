#include <flutter/runtime_effect.glsl>

// The first size and sampler are supplied by ImageFilter.shader (Impeller).
uniform vec2 u_size;
uniform vec4 u_rect;
uniform float u_radius;
uniform float u_pixelRatio;
uniform sampler2D u_backdrop;
out vec4 frag_color;

float roundedBox(vec2 p, vec2 halfSize, float radius) {
  vec2 q = abs(p) - halfSize + radius;
  return min(max(q.x, q.y), 0.0) + length(max(q, 0.0)) - radius;
}

vec4 sampleBackdrop(vec2 p) {
  vec2 uv = clamp(p / u_size, vec2(0.001), vec2(0.999));
  #ifdef IMPELLER_TARGET_OPENGLES
    uv.y = 1.0 - uv.y;
  #endif
  return texture(u_backdrop, uv);
}

void main() {
  vec2 pixel = FlutterFragCoord().xy;
  vec2 p = pixel - (u_rect.xy + u_rect.zw * 0.5);
  float radius = min(u_radius * u_pixelRatio, min(u_rect.z, u_rect.w) * 0.5);
  float d = roundedBox(p, u_rect.zw * 0.5, radius);
  // A convex rim bends the actual content inward. The middle stays quiet;
  // this is spatial refraction, not a moving decorative noise texture.
  vec2 normal = normalize(vec2(
    roundedBox(p + vec2(1.0, 0.0), u_rect.zw * 0.5, radius) - d,
    roundedBox(p + vec2(0.0, 1.0), u_rect.zw * 0.5, radius) - d
  ) + vec2(0.0001));
  float edge = 1.0 - smoothstep(0.0, 16.0 * u_pixelRatio, -d);
  vec2 refracted = pixel - normal * edge * edge * 9.0 * u_pixelRatio;
  // Diffusion is supplied by the compositor's separable Gaussian blur.
  frag_color = sampleBackdrop(refracted);
}
