import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'

function checkPermission(value: string | string[]): boolean {
  const store = usePermissionStore()
  const codes = Array.isArray(value) ? value : [value]
  if (!codes.length || !codes[0]) return true
  return codes.some((c) => store.hasPermission(c))
}

function apply(el: HTMLElement, binding: DirectiveBinding) {
  const ok = checkPermission(binding.value)
  el.style.display = ok ? '' : 'none'
}

export const permissionDirective: Directive = {
  mounted(el, binding) {
    apply(el as HTMLElement, binding)
  },
  updated(el, binding) {
    apply(el as HTMLElement, binding)
  },
}

export default permissionDirective
