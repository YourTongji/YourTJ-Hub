// Native Web Crypto is available in both the Node collector and Workers, without
// a Node compatibility layer or JavaScript hashing in the request CPU budget.
export async function hash(value: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value))
  return Array.from(new Uint8Array(digest), byte => byte.toString(16).padStart(2, '0')).join('')
}
