export interface MarkdownImagePreview {
  images: Array<{ src: string; alt: string }>
  index: number
}

/** Shared click/gallery policy for rendered post and Wiki Markdown. */
export function getMarkdownImagePreview(
  target: EventTarget | null,
  root: ParentNode = document,
): MarkdownImagePreview | null {
  if (!(target instanceof HTMLElement)) return null
  const image = target.closest('.gf-prose-post img')
  if (!(image instanceof HTMLImageElement)) return null
  if (isStickerImage(image)) return null

  const imageSrc = image.currentSrc || image.src
  if (!imageSrc) return null
  const anchor = image.closest('a')
  if (anchor && !sameUrl(anchor.href, imageSrc)) return null

  const images = Array.from(root.querySelectorAll<HTMLImageElement>('.gf-prose-post img'))
    .filter((item) => !isStickerImage(item))
    .map((item) => ({ src: item.currentSrc || item.src, alt: item.alt || '' }))
    .filter((item) => item.src)
  const index = images.findIndex((item) => sameUrl(item.src, imageSrc))
  return { images, index: index >= 0 ? index : 0 }
}

function isStickerImage(image: HTMLImageElement) {
  // The Markdown renderer emits this marker for both official and personal
  // stickers. Match the marker, not the URL: ordinary photos stay previewable.
  return image.hasAttribute('data-gf-sticker')
}

function sameUrl(left: string, right: string) {
  try {
    return new URL(left, window.location.href).href === new URL(right, window.location.href).href
  } catch {
    return left === right
  }
}
