// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

const storageKey = 'goose:flash-messages'
let flashModule: typeof import('../src/runtime/flash-message')

beforeEach(async () => {
  vi.useFakeTimers()
  vi.resetModules()
  sessionStorage.removeItem(storageKey)
  flashModule = await import('../src/runtime/flash-message')
})

afterEach(() => {
  const flash = flashModule.useFlashMessages()
  flash.messages.value.forEach(item => flash.dismiss(item.id))
  sessionStorage.removeItem(storageKey)
  vi.useRealTimers()
})

test.each([3, 5])('dismissing all %s visible messages during iteration clears every timer', count => {
  const flash = flashModule.useFlashMessages()
  for (let index = 0; index < count; index++) flash.push(`Message ${index}`)
  expect(flash.messages.value).toHaveLength(count)
  expect(vi.getTimerCount()).toBe(count)

  // dismiss replaces the ref's array; it does not splice the array being traversed.
  flash.messages.value.forEach(item => flash.dismiss(item.id))

  expect(flash.messages.value).toEqual([])
  expect(vi.getTimerCount()).toBe(0)
})

test('hydration consumes the navigation queue once per document', async () => {
  flashModule.queueFlashMessage('First navigation', 'success')
  flashModule.hydrateFlashMessages()
  const flash = flashModule.useFlashMessages()
  expect(flash.messages.value.map(item => item.message)).toEqual(['First navigation'])
  expect(sessionStorage.getItem(storageKey)).toBeNull()

  flashModule.queueFlashMessage('Next navigation', 'info')
  flashModule.hydrateFlashMessages()
  expect(flash.messages.value.map(item => item.message)).toEqual(['First navigation'])
  expect(JSON.parse(sessionStorage.getItem(storageKey)!)).toEqual([
    { type: 'info', message: 'Next navigation' },
  ])

  flash.messages.value.forEach(item => flash.dismiss(item.id))
  // A new module instance models the next document; hydration is not a test reset.
  vi.resetModules()
  flashModule = await import('../src/runtime/flash-message')
  flashModule.hydrateFlashMessages()
  flashModule.hydrateFlashMessages()
  expect(flashModule.useFlashMessages().messages.value.map(item => item.message)).toEqual(['Next navigation'])
  expect(sessionStorage.getItem(storageKey)).toBeNull()
})
