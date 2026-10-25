// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { readCaretTextWindow } from '../src/runtime/caret-text'
import { deleteStickerAtCaret, stickerDeleteDirection } from '../src/runtime/sticker-atomic'

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
  // execCommand 是 defineProperty 覆写上的自有属性，restoreAllMocks 恢复不了
  delete (document as unknown as { execCommand?: unknown }).execCommand
})

function editorElement(): HTMLElement {
  return document.querySelector<HTMLElement>('#editor')!
}

function placeCaret(node: Node, offset: number) {
  const range = document.createRange()
  range.setStart(node, offset)
  range.collapse(true)
  const selection = window.getSelection()!
  selection.removeAllRanges()
  selection.addRange(range)
}

function mockDeleteCommand() {
  const spy = vi.fn(() => true)
  Object.defineProperty(document, 'execCommand', { configurable: true, value: spy })
  return spy
}

test('Backspace after a token split across inline text nodes selects the whole token', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p>你好 <span>[:sticker:hu</span><b>aji:]</b></p></div>'
  const tail = editorElement().querySelector('b')!.firstChild!
  placeCaret(tail, 5)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(true)
  expect(window.getSelection()!.toString()).toBe('[:sticker:huaji:]')
  expect(execCommand).toHaveBeenCalledTimes(1)
  expect(execCommand).toHaveBeenCalledWith('delete')
})

test('Delete at the token start selects the whole token', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p><span>[:sticker:</span><i>huaji:]</i> 后文</p></div>'
  placeCaret(editorElement().querySelector('span')!.firstChild!, 0)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'forward')).toBe(true)
  expect(window.getSelection()!.toString()).toBe('[:sticker:huaji:]')
  expect(execCommand).toHaveBeenCalledTimes(1)
})

test('an unterminated fragment is selected as one block', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p>图 <span>[:sticker:na</span></p></div>'
  placeCaret(editorElement().querySelector('span')!.firstChild!, 12)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(true)
  expect(window.getSelection()!.toString()).toBe('[:sticker:na')
  expect(execCommand).toHaveBeenCalledTimes(1)
})

test('plain text and a token in the previous block are left to the default deletion', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p>[:sticker:huaji:]</p><p>后文</p></div>'
  placeCaret(editorElement().querySelectorAll('p')[1].firstChild!, 0)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(false)
  expect(deleteStickerAtCaret(editorElement(), 'forward')).toBe(false)
  expect(execCommand).not.toHaveBeenCalled()
})

test('a non-collapsed selection or a caret outside the editor is ignored', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p>[:sticker:huaji:]</p></div><span id="outside">[:sticker:huaji:]</span>'
  const execCommand = mockDeleteCommand()

  const inside = editorElement().querySelector('p')!.firstChild!
  const range = document.createRange()
  range.setStart(inside, 0)
  range.setEnd(inside, 14)
  window.getSelection()!.removeAllRanges()
  window.getSelection()!.addRange(range)
  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(false)

  placeCaret(document.querySelector('#outside')!.firstChild!, 14)
  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(false)
  expect(execCommand).not.toHaveBeenCalled()
})

test('a token in another list item is out of scope', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><ul><li>[:sticker:huaji:]</li><li>后文</li></ul></div>'
  placeCaret(editorElement().querySelectorAll('li')[1].firstChild!, 0)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(false)
  expect(deleteStickerAtCaret(editorElement(), 'forward')).toBe(false)
  expect(execCommand).not.toHaveBeenCalled()
})

test('a <br> right after the token stays outside the deleted range', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p>[:sticker:x:]<br>后</p></div>'
  placeCaret(editorElement().querySelector('p')!.firstChild!, 13)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(true)
  const selected = window.getSelection()!.getRangeAt(0).cloneContents()
  expect(selected.querySelector('br')).toBeNull()
  expect(selected.textContent).toBe('[:sticker:x:]')
  expect(execCommand).toHaveBeenCalledTimes(1)
})

test('a caret sitting on the editor root itself is not intercepted', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p>[:sticker:huaji:]</p></div>'
  placeCaret(editorElement(), 0)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(false)
  expect(deleteStickerAtCaret(editorElement(), 'forward')).toBe(false)
  expect(execCommand).not.toHaveBeenCalled()
})

test('a caret in text attached to the editor root reads only that root child', () => {
  document.body.innerHTML = '<div id="editor" contenteditable>[:sticker:na<div>huaji:]</div></div>'
  const text = editorElement().firstChild!
  placeCaret(text, 12)
  const execCommand = mockDeleteCommand()

  expect(deleteStickerAtCaret(editorElement(), 'backward')).toBe(true)
  expect(window.getSelection()!.toString()).toBe('[:sticker:na')
  expect(execCommand).toHaveBeenCalledTimes(1)
})

test('a collapsed character range is null', () => {
  document.body.innerHTML = '<div id="editor" contenteditable><p>[:sticker:huaji:]</p></div>'
  const text = editorElement().querySelector('p')!.firstChild!
  const textWindow = readCaretTextWindow(text, 13, 200, editorElement())!

  expect(textWindow.range(textWindow.caret, textWindow.caret)).toBeNull()
  expect(textWindow.range(3, 1)).toBeNull()
  expect(textWindow.range(0, 1)?.toString()).toBe('[')
})

test('IME composition and Ctrl/Meta/Alt combinations are not intercepted; Shift still deletes the token', () => {
  const key = (init: KeyboardEventInit) => new KeyboardEvent('keydown', init)
  expect(stickerDeleteDirection(key({ key: 'Backspace' }))).toBe('backward')
  expect(stickerDeleteDirection(key({ key: 'Delete' }))).toBe('forward')
  expect(stickerDeleteDirection(key({ key: 'Backspace', shiftKey: true }))).toBe('backward')
  expect(stickerDeleteDirection(key({ key: 'Delete', shiftKey: true }))).toBe('forward')
  expect(stickerDeleteDirection(key({ key: 'Backspace', isComposing: true }))).toBeNull()
  expect(stickerDeleteDirection(key({ key: 'Backspace', ctrlKey: true }))).toBeNull()
  expect(stickerDeleteDirection(key({ key: 'Delete', metaKey: true }))).toBeNull()
  expect(stickerDeleteDirection(key({ key: 'Delete', altKey: true }))).toBeNull()
  expect(stickerDeleteDirection(key({ key: 'a' }))).toBeNull()
})
