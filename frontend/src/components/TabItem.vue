<!-- Adapted from ys-ll/uniterm (Apache-2.0) by FnTerminal, 2026-10-08: browser/fnOS transport, terminal-only controls. -->
<template>
  <div
    class="tab-item" role="tab" :aria-selected="isActive" :tabindex="isActive ? 0 : -1"
    :class="{ active: isActive }"
    :data-tab-id="tab.id"
    @click="$emit('activate', tab.id)"
    @mouseenter="hovered = true"
    @mouseleave="hovered = false"
    draggable="true"
    @dragstart="onDragStart"
    @dragend="onDragEnd"
    @contextmenu="onContextMenu"
  >
    <button
      v-if="!tabCloseRight && (hovered || isTouch) && !tab.locked"
      class="tab-close" :aria-label="`关闭 ${tab.name}`"
      @click.stop="$emit('close', tab.id)"
    ><X /></button>
    <span
      v-if="tabCloseRight || (!hovered && !isTouch) || tab.locked"
      class="tab-icon-wrapper"
    >
      <component
        :is="tab.locked ? Lock : tabIcon"
        class="tab-type-icon"

      />
    </span>
    <span v-if="!editing" class="tab-name" :class="{ 'tab-disconnected': isDisconnected }" :title="tab.name + (tabShortcut ? ` (${tabShortcut})` : '')" @dblclick.stop="startEdit">
      <span class="tab-name-text">{{ tab.name }}</span>
    </span>
    <input
      v-else
      ref="editInputRef"
      v-model="editName"
      class="tab-name-input"
      @keydown.enter="confirmEdit"
      @keydown.escape="cancelEdit"
      @blur="confirmEdit"
      @click.stop
    />
    <span v-if="tabShortcut && settingsStore.settings.showTabShortcutHints" class="tab-shortcut">{{ tabShortcut }}</span>
    <!-- OSC 9;4 progress indicator: a 2px bar along the tab's bottom edge.
         state 1 = normal (accent), 2 = error (red), 3 = indeterminate
         (animated), 4 = paused/warning (orange). -->
    <button
      v-if="tabCloseRight"
      class="tab-close tab-close-right" :aria-label="`关闭 ${tab.name}`"
      :class="{ 'tab-close-right-ghost': (!hovered && !isTouch) || tab.locked }"
      @click.stop="$emit('close', tab.id)"
    ><X /></button>
    <Menu ref="ctxMenuRef" v-model:visible="ctxMenuVisible">
      <!-- ⓪ 工作区操作 -->




      <!-- ① 标签类操作 -->
      <MenuItem v-if="canDuplicate" :shortcut="menuShortcut('duplicateSession')" @click="onDuplicate">
        {{ t('tab.duplicate') }}
      </MenuItem>

      <MenuItem v-if="tab.mode==='nas'" @click="editConnection">编辑连接…</MenuItem>
      <MenuItem v-if="canReconnect" @click="onReconnect">{{ t('tab.reconnect') }}</MenuItem>


      <MenuItem @click="startEdit">{{ t('tab.rename') }}</MenuItem>

      <MenuItem @click="toggleLock">
        {{ tab.locked ? t('tab.unlock') : t('tab.lock') }}
      </MenuItem>


      <!-- ② 会话文本操作 -->
      <MenuDivider />
      <MenuItem v-if="tab.type === 'terminal'" :shortcut="menuShortcut('terminalSearch')" @click="triggerSearch">
        {{ t('terminal.searchText') }}
      </MenuItem>
      <MenuItem v-if="tab.type === 'terminal'" @click="triggerExport">{{ t('terminal.export') }}</MenuItem>
      <MenuItem v-if="tab.type === 'terminal'" @click="triggerResetOutput">{{ t('terminal.resetOutput') }}</MenuItem>
      <MenuItem v-if="tab.type === 'terminal'" @click="triggerClearScrollback">{{ t('terminal.clearScrollback') }}</MenuItem>



      <!-- ③ 连接功能（ssh / rdp） -->
      <MenuDivider />






      <!-- ④ 关闭标签操作 -->
      <MenuDivider />
      <MenuItem :class="{ disabled: tab.locked }" :shortcut="menuShortcut('closePanel')" @click="tab.locked ? null : closeTab()">
        {{ t('tab.close') }}
      </MenuItem>
      <MenuItem @click="closeOther">{{ t('tab.closeOther') }}</MenuItem>
      <MenuItem @click="closeRight">{{ t('tab.closeRight') }}</MenuItem>
    </Menu>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { SquareTerminal, X, Lock } from '@lucide/vue'
import { useTabStore } from '../stores/tabStore'
import { useSettingsStore } from '../stores/settingsStore'
import { useI18n } from '../i18n'
import { isMobilePlatform } from '../utils/platform'
import type { TerminalTab } from '../types/terminal'
import Menu from './Menu.vue'
import MenuItem from './MenuItem.vue'
import MenuDivider from './MenuDivider.vue'
const props = defineProps<{ tab: TerminalTab; shortcutIndex?: number; isActive: boolean;  }>()
const emit = defineEmits<{activate:[id:string];close:[id:string];closeBatch:[ids:string[]]}>()
const tabStore = useTabStore()
const settingsStore = useSettingsStore()
const { t } = useI18n()
const hovered = ref(false), editing = ref(false), editName = ref(''), ctxMenuVisible = ref(false)
const editInputRef = ref<HTMLInputElement>(), ctxMenuRef = ref<InstanceType<typeof Menu> | null>(null)
const tabCloseRight = computed(() => settingsStore.settings.tabCloseButton === 'right')
const isTouch = window.matchMedia('(pointer: coarse)').matches
const tabIcon = SquareTerminal
const isDisconnected = computed(() => props.tab.status !== 'connected')
const canDuplicate = true, canReconnect = true
const tabShortcut = computed(() => !isMobilePlatform() && props.shortcutIndex && props.shortcutIndex<=9 ? `⌃⌥${props.shortcutIndex}` : '')
function menuShortcut(action:string) { return ({duplicateSession:'Ctrl+Shift+T',closePanel:'Ctrl+Shift+W',terminalSearch:'Ctrl+Shift+F'} as Record<string,string>)[action] || '' }
function onContextMenu(e:MouseEvent) {e.preventDefault();e.stopPropagation();ctxMenuRef.value?.openAt(e.clientX,e.clientY,props.tab)}
function closeContextMenu() {ctxMenuVisible.value=false}
function editConnection(){closeContextMenu();window.dispatchEvent(new CustomEvent('fnterminal:edit-connection',{detail:props.tab.id}))}
function onDuplicate() {closeContextMenu();tabStore.createStartTab(props.tab.mode,props.tab.sshPort,props.tab.connection)}
function onDragStart(e:DragEvent) { e.dataTransfer?.setData('application/tab-id',props.tab.id);e.dataTransfer?.setData('application/tab-type','terminal');if(e.dataTransfer)e.dataTransfer.effectAllowed='move';tabStore.setDraggingTabId(props.tab.id) }
function onDragEnd() {tabStore.setDraggingTabId(null)}
function startEdit() {
  closeContextMenu()
  editName.value = props.tab.name
  editing.value = true
  nextTick(() => {
    editInputRef.value?.focus()
    editInputRef.value?.select()
  })
}
function confirmEdit() {
  if (!editing.value) return
  editing.value = false
  const newName = editName.value.trim()
  if (newName && newName !== props.tab.name) {
    tabStore.renameTab(props.tab.id, newName)
  }
}
function cancelEdit() {
  editing.value = false
}
function toggleLock() {
  tabStore.toggleTabLock(props.tab.id)
  closeContextMenu()
}
function closeTab() {
  emit('close', props.tab.id)
  closeContextMenu()
}
function closeOther() {
  const allTabs = tabStore.tabs
  const currentIdx = allTabs.findIndex(t => t.id === props.tab.id)
  const ids = allTabs.filter((t, i) => i !== currentIdx && !t.locked).map(t => t.id)
  if (ids.length) emit('closeBatch', ids)
  closeContextMenu()
}
function closeRight() {
  const allTabs = tabStore.tabs
  const currentIdx = allTabs.findIndex(t => t.id === props.tab.id)
  const ids = allTabs.slice(currentIdx + 1).filter(t => !t.locked).map(t => t.id)
  if (ids.length) emit('closeBatch', ids)
  closeContextMenu()
}
function onReconnect() {
  closeContextMenu()
  const panelId = 'panelId' in props.tab ? props.tab.panelId : ''
  if (!panelId) return
  window.dispatchEvent(new CustomEvent('panel:reconnect', { detail: { panelId } }))
}
function triggerSearch() {
  window.dispatchEvent(new CustomEvent('terminal:open-search', { detail: { panelId: props.tab.panelId } }))
  closeContextMenu()
}
function triggerExport() {
  window.dispatchEvent(new CustomEvent('terminal:export', { detail: { panelId: props.tab.panelId } }))
  closeContextMenu()
}
function triggerResetOutput() {
  window.dispatchEvent(new CustomEvent('terminal:reset-output', { detail: { panelId: props.tab.panelId } }))
  closeContextMenu()
}
function triggerClearScrollback() {
  window.dispatchEvent(new CustomEvent('terminal:clear-scrollback', { detail: { panelId: (props.tab as TerminalTab).panelId } }))
  closeContextMenu()
}

</script>
<style scoped>
.tab-item {
  display: flex;
  align-items: center;
  gap: 0.125rem;
  height: 1.75rem;
  min-width: 9rem;
  padding: 0 0.75rem;
  margin: 0 1px;
  user-select: none;
  border-radius: var(--radius-sm);
  position: relative;
  color: var(--text-secondary);
  font-size: 0.75rem;
  transition: background 0.15s ease, color 0.15s ease;
  flex-shrink: 0;
}
.tab-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.tab-item.active {
  background: var(--bg-hover);
  color: var(--text-primary);
  box-shadow: inset 0 0 0 1px var(--accent);
}
/* AI-locked tabs are signalled by the tab icon alone: a solid warning-coloured
   icon with a soft glow. No surface tint — low-alpha warning colour over the
   dark background turns muddy (issue #928), and at higher alpha it reads as a
   flat orange block. The accent ring stays the sole "which tab is selected"
   signal. */
.tab-name {
  font-size: 0.75rem;
  white-space: nowrap;
  overflow: hidden;
  /* Keep a full IPv6 address visible; longer custom names still use ellipsis. */
  max-width: 18.75rem;
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-weight: 500;
}
.tab-name-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tab-shortcut {
  flex-shrink: 0;
  margin-left: 0.25rem;
  color: var(--text-muted);
  font-size: 0.625rem;
  font-weight: 500;
  /* UI font so macOS modifier symbols (⌘⌥⇧) render with their native
     system-font shapes instead of a mono fallback. */
  font-family: var(--font-ui);
}
.tab-disconnected {
  opacity: 0.5;
}
.tab-icon-wrapper {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
  margin-right: 0.25rem;
}
.tab-type-icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 0.875rem;
  height: 0.875rem;
  color: var(--text-muted);
}
/* Broadcast status icon occupies the exact same far-right slot as the right
   close button: with margin-left:auto it pushes to the edge, and the right X
   is v-if'd out of the DOM (see template) whenever the icon is shown, so the
   two are never siblings — the icon sits precisely where the X would. In the
   default (left-close) layout there is no right X, so the icon always takes
   the slot. Reserving real layout space (not absolute positioning) means it
   never overlaps the tab name. */
/* OSC 9;4 progress bar along the tab's bottom edge (see template). */
@keyframes tab-progress-indeterminate {
  from { left: -40%; }
  to { left: 100%; }
}
.tab-item.active .tab-type-icon {
  color: var(--accent);
}
/* Locked wins over the active accent: the AI state must stay visible when the
   tab is selected. */
.tab-item.active .tab-type-icon.ai-locked-icon {
  color: var(--warning);
}
.transfer-indicator {
  color: var(--accent);
  flex-shrink: 0;
  line-height: 1;
}
.tab-name-input {
  font-size: 0.75rem;
  font-family: inherit;
  color: var(--text-primary);
  background: var(--bg-base);
  border: 1px solid var(--accent);
  border-radius: var(--radius-sm);
  padding: 0.125rem 0.375rem;
  width: 7.5rem;
  outline: none;
}
.tab-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 0.875rem;
  height: 0.875rem;
  margin-right: 0.25rem;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-size: 0.875rem;
  transition: all 0.12s ease;
}
.tab-close:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
/* Close button on the right side of the tab (appearance setting).
   margin-left:auto pushes it flush to the far right edge of the tab
   (inside the 0.75rem horizontal padding), instead of hugging the name.

   The button is always present in the layout when the right-side setting is on
   — ghosted (visibility:hidden) while not hovered OR the tab is locked — so its
   slot is always reserved; showing the X only toggles visibility instead of
   inserting/removing an element, which would otherwise re-truncate long names
   and cause jitter (including the instant the tab is locked). */
.tab-close.tab-close-right {
  margin-left: auto;
  margin-right: 0;
}
.tab-close-right-ghost {
  visibility: hidden;
  /* Hide instantly on mouse-leave: `.tab-close` carries transition:all, and
     visibility is a transitionable property, so a visible→hidden ghost would
     otherwise linger for the transition duration. */
  transition: visibility 0s;
}
.tab-item:focus-within .tab-close-right-ghost {visibility:visible}
</style>
