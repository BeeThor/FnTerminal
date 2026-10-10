<!-- Modified by FnTerminal, 2026-10-08: browser-only terminal tab/store adapter; numeric icon sizes. Original: ys-ll/uniterm, Apache-2.0. -->
<!-- Modified by FnTerminal, 2026-10-08: numeric Lucide icon sizes for browser type checks. Original: ys-ll/uniterm, Apache-2.0. -->
<template>
  <div class="tabs-list" ref="tabsListRef" @wheel="onWheel" @dragover.prevent="onTabsContainerDragOver" @dragleave="onTabsDragLeave" @drop="onTabsContainerDrop">
    <template v-for="(tab, index) in tabs" :key="tab.id">
      <div
        v-if="(dragOverTabIndex === index && !dragOverInsertAfter) || (dragOverTabIndex === index - 1 && dragOverInsertAfter)"
        class="tab-drop-indicator"
      ></div>

      <TabItem
        :tab="tab"
        :shortcut-index="index + 1"
        :is-active="tab.id === activeTabId"
        @activate="setActiveTab"
        @close="(id: string) => $emit('close-tab', id)"
        @close-batch="(ids: string[]) => $emit('close-tab-batch', ids)"
        @dragstart="(e: DragEvent, tabId: string) => $emit('tab-dragstart', e, tabId)"
        @dragover.prevent="(e: DragEvent) => onTabDragOver(e, index)"
        @dragleave="onTabDragLeave"
        @drop="(e: DragEvent) => onTabDrop(e, tab.id, index)"
      />
    </template>
    <div
      v-if="dragOverTabIndex === tabs.length - 1 && dragOverInsertAfter"
      class="tab-drop-indicator"
    ></div>
    <!-- Drop indicator at end when dragging over empty tabs area (renders
         inline after the last tab, same spot as the per-tab insert-after
         indicator — no auto margin, which used to fling the + button to the
         far right edge, #922) -->
    <div
      v-if="dragOverContainer"
      class="tab-drop-indicator"
    ></div>
    <button
      v-if="!showMore"
      class="tab-add-btn"
      :title="t('startTab.defaultName') + shortcutSuffix('newConnection')"
      @click="onAddStartTab"
    >
      <Plus :size="14" />
    </button>
  </div>
  <div class="tab-more" v-if="showMore">
    <span class="tab-more-btn" :title="t('tab.more')" @click.stop="tabMoreRef?.toggle($event.currentTarget as HTMLElement)">
      <el-icon class="tab-more-icon"><MoreHorizontal :size="14" /></el-icon>
    </span>
    <Menu ref="tabMoreRef" v-model:visible="tabMoreVisible" align="end">
      <MenuItem
        v-for="tab in tabs"
        :key="tab.id"
        :class="{ active: tab.id === activeTabId }" :title="tab.name"
        @click="onTabMoreSelect(tab.id)"
      >
        {{ tab.name }}
      </MenuItem>
    </Menu>
    <button
      class="tab-add-btn"
      :title="t('startTab.defaultName') + shortcutSuffix('newConnection')"
      @click="onAddStartTab"
    >
      <Plus :size="14" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { MoreHorizontal, Plus } from '@lucide/vue'
import { useTabStore } from '../stores/tabStore'
import { useI18n } from '../i18n'
import { useSettingsStore } from '../stores/settingsStore'
import { formatKeyBinding } from '../composables/useKeyboardShortcuts'
import { isMobilePlatform } from '../utils/platform'
import TabItem from './TabItem.vue'
import Menu from './Menu.vue'
import MenuItem from './MenuItem.vue'

const tabStore = useTabStore()
const settingsStore = useSettingsStore()
const { t } = useI18n()

const isMac = /Mac|iPhone|iPad/.test(navigator.userAgent)

// " (Ctrl+Shift+N)" suffix for the new-tab tooltip, '' when unset. Reactive via
// settingsStore, so the tooltip updates when the user rebinds the shortcut.
// '' on mobile too — no hardware keyboard, so the hint is meaningless.
function shortcutSuffix(action: 'newConnection'): string {
  if (isMobilePlatform()) return ''
  const b = settingsStore.settings.keyboard[action]
  if (!b) return ''
  const key = formatKeyBinding(b, isMac)
  return key ? ` (${key})` : ''
}

const tabs = computed(() => tabStore.tabs)
const activeTabId = computed(() => tabStore.activeTabId)

const dragOverTabIndex = ref<number | null>(null)
const dragOverInsertAfter = ref(false)
const dragOverContainer = ref(false)

const tabsListRef = ref<HTMLElement | null>(null)
const showMore = ref(false)

function updateOverflow() {
  const el = tabsListRef.value
  if (!el) return
  showMore.value = el.scrollWidth > el.clientWidth + 1
}

// Watch tab changes and window resize to update overflow state

watch(() => tabs.value.length, () => nextTick(updateOverflow))
watch(activeTabId, () => nextTick(updateOverflow))

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  resizeObserver = new ResizeObserver(updateOverflow)
  if (tabsListRef.value) {
    resizeObserver.observe(tabsListRef.value)
  }
  nextTick(updateOverflow)
  window.addEventListener('dragend', clearDragState)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  window.removeEventListener('dragend', clearDragState)
})

defineEmits<{
  'close-tab': [id: string]
  'close-tab-batch': [ids: string[]]
  'tab-dragstart': [e: DragEvent, tabId: string]
}>()

function onAddStartTab() {
  window.dispatchEvent(new CustomEvent('fnterminal:new-connection'))
}

function onWheel(e: WheelEvent) {
  if (!tabsListRef.value) return
  tabsListRef.value.scrollLeft += e.deltaY
}

const tabMoreRef = ref<InstanceType<typeof Menu> | null>(null)
const tabMoreVisible = ref(false)
function onTabMoreSelect(id: string) {
  tabMoreVisible.value = false
  tabStore.setActiveTab(id)
  scrollToTab(id)
}

function setActiveTab(id: string) {
  tabStore.setActiveTab(id)
  scrollToTab(id)
}

function scrollToTab(tabId: string) {
  if (!tabsListRef.value) return
  const el = tabsListRef.value.querySelector(`[data-tab-id="${tabId}"]`) as HTMLElement | null
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
  }
}

// FnTerminal extracts terminal tab reordering; no desktop workspace/connection drops.
function onTabDragOver(e:DragEvent,index:number){if(!e.dataTransfer?.types.includes('application/tab-id'))return;const rect=(e.currentTarget as HTMLElement).getBoundingClientRect();dragOverContainer.value=false;dragOverTabIndex.value=index;dragOverInsertAfter.value=e.clientX>=rect.left+rect.width/2;e.dataTransfer.dropEffect='move'}
function onTabDragLeave(_e:DragEvent){}
function onTabsDragLeave(e:DragEvent){const el=e.currentTarget as HTMLElement;if(!e.relatedTarget||!el.contains(e.relatedTarget as Node))clearDragState()}
function clearDragState(){dragOverTabIndex.value=null;dragOverInsertAfter.value=false;dragOverContainer.value=false}
function onTabsContainerDragOver(e:DragEvent){if(!e.dataTransfer?.types.includes('application/tab-id'))return;if((e.target as HTMLElement).closest('.tab-item'))return;dragOverContainer.value=true;dragOverTabIndex.value=null;dragOverInsertAfter.value=false;e.dataTransfer.dropEffect='move'}
function onTabsContainerDrop(e:DragEvent){e.stopPropagation();if((e.target as HTMLElement).closest('.tab-item')){clearDragState();return}const id=e.dataTransfer?.getData('application/tab-id');clearDragState();const from=tabs.value.findIndex(t=>t.id===id);if(from>=0)tabStore.moveTab(from,tabs.value.length-1)}
function onTabDrop(e:DragEvent,targetId:string,index:number){e.stopPropagation();const after=dragOverInsertAfter.value;clearDragState();const id=e.dataTransfer?.getData('application/tab-id');if(!id||id===targetId)return;const from=tabs.value.findIndex(t=>t.id===id);if(from<0)return;let to=index+(after?1:0);if(to>from)to--;tabStore.moveTab(from,to)}
</script>

<style scoped>
.tabs-list {
  display: flex;
  flex: 1;
  overflow-x: auto;
  overflow-y: hidden;
  align-items: center;
  scrollbar-width: none;
  min-width: 0;
}
.tabs-list::-webkit-scrollbar {
  display: none;
}

.tab-more {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  padding: 0 0.25rem;
  height: 1.75rem;
}
.tab-more-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 1px;
  user-select: none;
  transition: all 0.15s;
}
.tab-more-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.tab-drop-indicator {
  width: 0.125rem;
  min-width: 0.125rem;
  align-self: stretch;
  background: var(--accent);
  opacity: 0.8;
  margin: 0.25rem 0;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}
/* 拖到标签条空白处时的"插到末尾"指示条。不带 margin-left:auto——那会把
   自由空间全吸到自己前面，连着后面的新增 + 按钮一起甩到最右边 (#922)。
   不加 auto 时它落在最后一个标签之后、+ 按钮之前，与"拖过末尾标签右半"
   的指示条（onTabDragOver 的 insert-after 分支）位置一致。 */
.tab-add-btn {
  flex-shrink: 0;
  width: 1.75rem;
  height: 1.75rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
  margin-left: 0.125rem;
}
.tab-add-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
</style>
