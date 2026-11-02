import { ref } from 'vue'

// AppShell mounts a single MyStickersDialog. Pickers close themselves before
// opening it, because a dialog inside a picker would vanish with the picker.
export const myStickersDialogOpen = ref(false)
/** AppShell loads the dialog only after its first use. */
export const myStickersDialogUsed = ref(false)

export function openMyStickers() {
  myStickersDialogUsed.value = true
  myStickersDialogOpen.value = true
}
