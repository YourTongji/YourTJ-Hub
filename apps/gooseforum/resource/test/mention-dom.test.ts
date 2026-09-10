// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { textBeforeCaret, isInsideFencedCode, replaceMentionTokenInElement } from '../src/runtime/mention-dom'
afterEach(() => { document.body.innerHTML = ''; vi.restoreAllMocks() })
test('caret at editor start cannot read preceding UI text', () => {
 document.body.innerHTML = '<span>@private</span><div id="editor" contenteditable></div>'
 const editor = document.querySelector<HTMLElement>('#editor')!
 expect(textBeforeCaret(editor, 0, 200, editor)).toBe('')
 const selection = window.getSelection()!
 const range = document.createRange(); range.setStart(editor,0); range.collapse(true)
 selection.removeAllRanges(); selection.addRange(range)
 Object.defineProperty(document,'execCommand',{configurable:true,value:vi.fn(() => true)})
 expect(replaceMentionTokenInElement(editor,8,'@public')).toBe(false)
 expect(document.execCommand).not.toHaveBeenCalled()
})
test('caret reads and replaces a token across inline text nodes', () => {
 document.body.innerHTML = '<div id="editor" contenteditable>hello @al<span>ice</span></div>'
 const editor = document.querySelector<HTMLElement>('#editor')!
 const text=editor.querySelector('span')!.firstChild!
 expect(textBeforeCaret(text,3,200,editor)).toBe('hello @alice')
 const range=document.createRange();range.setStart(text,3);range.collapse(true)
 window.getSelection()!.removeAllRanges();window.getSelection()!.addRange(range)
 Object.defineProperty(document,'execCommand',{configurable:true,value:vi.fn(() => {
  expect(window.getSelection()!.toString()).toBe('@alice');return true
 })})
 expect(replaceMentionTokenInElement(editor,6,'@bob')).toBe(true)
})

test('source mode matches the opening fence delimiter and inline run length', () => {
 expect(isInsideFencedCode('```text\n~~~\n@alice')).toBe(true)
 expect(isInsideFencedCode('`` code @alice')).toBe(true)
 expect(isInsideFencedCode('```text\nhello\n```\n@alice')).toBe(false)
})
