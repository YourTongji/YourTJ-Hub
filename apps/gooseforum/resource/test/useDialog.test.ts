// @vitest-environment happy-dom
import { afterEach, describe, expect, test } from 'vitest'
import { nextTick, ref } from 'vue'
import { useDialog } from '../src/site/composables/useDialog'

function createDialogHost() {
  const host = document.createElement('div')
  host.innerHTML = `
    <div data-dialog>
      <button data-first>close</button>
      <button data-second>submit</button>
    </div>
  `
  document.body.appendChild(host)
  const dialog = host.querySelector<HTMLElement>('[data-dialog]')!
  const first = host.querySelector<HTMLButtonElement>('[data-first]')!
  const second = host.querySelector<HTMLButtonElement>('[data-second]')!
  return { host, dialog, first, second }
}

afterEach(() => {
  document.body.innerHTML = ''
  document.body.style.overflow = ''
})

async function settle() {
  await nextTick()
  await new Promise((resolve) => requestAnimationFrame(resolve))
}

describe('useDialog', () => {
  test('打开后焦点移入弹窗内首个可聚焦元素，body 滚动锁定', async () => {
    const { host, dialog, first } = createDialogHost()
    const visible = ref(false)
    const { dialogRef } = useDialog({ visible })
    dialogRef.value = dialog

    visible.value = true
    await settle()

    expect(document.activeElement).toBe(first)
    expect(document.body.style.overflow).toBe('hidden')
    host.remove()
  })

  test('关闭后滚动解锁，焦点恢复到触发元素', async () => {
    const { host, dialog } = createDialogHost()
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    trigger.focus()

    const visible = ref(false)
    const { dialogRef, closeDialog } = useDialog({ visible })
    dialogRef.value = dialog

    visible.value = true
    await settle()
    expect(document.body.style.overflow).toBe('hidden')

    closeDialog()
    await nextTick()
    expect(document.body.style.overflow).toBe('')
    expect(document.activeElement).toBe(trigger)
    host.remove()
    trigger.remove()
  })

  test('打开期间 Tab 焦点圈禁在弹窗内（循环到首位）', async () => {
    const { host, dialog, second } = createDialogHost()
    const visible = ref(false)
    const { dialogRef } = useDialog({ visible })
    dialogRef.value = dialog

    visible.value = true
    await settle()

    second.focus()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(second)
    host.remove()
  })

  test('Shift+Tab 在首个元素时循环到末尾', async () => {
    const { host, dialog, first, second } = createDialogHost()
    const visible = ref(false)
    const { dialogRef } = useDialog({ visible })
    dialogRef.value = dialog

    visible.value = true
    await settle()

    first.focus()
    document.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true }),
    )
    window.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true }),
    )
    expect(document.activeElement).toBe(second)
    host.remove()
  })

  test('Esc 关闭弹窗并恢复焦点', async () => {
    const { host, dialog } = createDialogHost()
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    trigger.focus()

    const visible = ref(false)
    const { dialogRef } = useDialog({ visible })
    dialogRef.value = dialog

    visible.value = true
    await settle()

    // 浏览器中 Esc 由 document 上的 keydown 监听处理；happy-dom 的
    // document.dispatchEvent 会命中 document 监听（见 shiftTab 用例的兜底），此处直接派发。
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    if (document.activeElement !== trigger) {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    }

    // 打开状态由真实组件以 v-model 同步 visible；useDialog 的 Esc 只负责关闭动作本身
    // （触发 closeDialog → 焦点恢复 + 滚动解锁），visible 由组件侧 watcher 翻转。
    expect(document.activeElement).toBe(trigger)
    expect(document.body.style.overflow).toBe('')
    host.remove()
    trigger.remove()
  })

  test('initialFocusSelector 指定打开后的聚焦元素', async () => {
    const { host, dialog, second } = createDialogHost()
    const visible = ref(false)
    const { dialogRef } = useDialog({ visible, initialFocusSelector: '[data-second]' })
    dialogRef.value = dialog

    visible.value = true
    await settle()

    expect(document.activeElement).toBe(second)
    host.remove()
  })

  test('弹窗内无可聚焦元素时，焦点落到弹窗本身（Tab 不逃逸）', async () => {
    const { host } = createDialogHost()
    host.querySelector('[data-dialog]')!.innerHTML = '<p>no focusable</p>'
    const dialog = host.querySelector<HTMLElement>('[data-dialog]')!

    const visible = ref(false)
    const { dialogRef } = useDialog({ visible })
    dialogRef.value = dialog

    visible.value = true
    await settle()

    expect(document.activeElement).toBe(dialog)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(dialog)
    host.remove()
  })
})
