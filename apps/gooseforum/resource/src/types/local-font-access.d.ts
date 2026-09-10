// Local Font Access API (Chrome/Edge 103+). Not in TS 6.0 dom lib yet.
interface FontMetadata {
  family: string
  fullName: string
  postscriptName: string
  style: string
}

interface Window {
  queryLocalFonts?: (options?: { postscriptNames?: string[] }) => Promise<FontMetadata[]>
}
