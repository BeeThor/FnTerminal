<!-- Adapted from ys-ll/uniterm AppHeader.vue (Apache-2.0), FnTerminal 2026-10-08: fnOS actions replace desktop window/AI bindings. -->
<template>
  <header class="app-header">

    <div class="header-tabs"><TabsList @close-tab="closeTab" @close-tab-batch="closeBatch" /></div>
    <button class="header-btn" title="搜索终端内容" aria-label="搜索终端内容" @click="action('terminal:open-search')"><Search :size="16" /></button>
    <button class="header-btn" title="输入文字或粘贴" aria-label="输入文字或粘贴" @click="emit('paste',tabs.activeTabId)"><ClipboardPaste :size="16" /></button>
    <button class="header-btn" title="移动快捷键" aria-label="移动快捷键" @click="emit('toggle-keys')"><Keyboard :size="16" /></button>
    <div class="settings-wrap"><button ref="buttonRef" class="header-btn" aria-label="终端菜单" title="终端菜单" @click.stop="menuRef?.toggle(buttonRef!)"><MenuIcon :size="16" /></button>
    <Menu ref="menuRef" align="end" v-model:visible="menuVisible">
      <MenuSubmenu label="高级诊断"><MenuItem @click="create">独立应用账户 Shell</MenuItem></MenuSubmenu>
      <MenuItem @click="action('panel:reconnect')">重新连接当前终端</MenuItem>
      <MenuItem @click="ssh">新建 SSH 连接…</MenuItem><MenuItem v-if="tabs.tabs.find(t=>t.id===tabs.activeTabId)?.mode==='nas'" @click="editSsh">编辑当前 SSH 连接…</MenuItem>
      <MenuDivider />
      <MenuItem @click="emit('paste',tabs.activeTabId);menuVisible=false">输入文字 / 粘贴…</MenuItem>
      <MenuItem @click="action('terminal:export')">导出终端内容</MenuItem>
      <MenuSubmenu label="界面主题"><MenuItem v-for="value in ['dark','light']" :key="value" :class="{active:settings.settings.theme===value}" @click="setTheme(value)">{{themeLabels[value]}}</MenuItem></MenuSubmenu>
      <MenuSubmenu label="终端配色"><MenuItem v-for="theme in TERMINAL_THEMES" :key="theme.value" :class="{active:settings.settings.terminal.theme===theme.value}" @click="settings.settings.terminal.theme=theme.value;settings.save()">{{theme.label}}</MenuItem></MenuSubmenu>
      <MenuSubmenu label="字号"><MenuItem v-for="size in [12,13,14,16,18,20]" :key="size" :class="{active:settings.settings.terminal.fontSize===size}" @click="settings.settings.terminal.fontSize=size;settings.save()">{{size}} px</MenuItem></MenuSubmenu>
      <MenuDivider /><MenuItem @click="menuVisible=false;emit('diagnostics')">连接诊断与日志</MenuItem>
    </Menu>
    </div>
  </header>
</template>
<script setup lang="ts">
import {ref} from 'vue'
import {Search,ClipboardPaste,Keyboard,Menu as MenuIcon} from '@lucide/vue'
import TabsList from './TabsList.vue'
import Menu from './Menu.vue'
import MenuItem from './MenuItem.vue'
import MenuSubmenu from './MenuSubmenu.vue'
import MenuDivider from './MenuDivider.vue'
import {useTabStore} from '../stores/tabStore'
import {useSettingsStore} from '../stores/settingsStore'
import {TERMINAL_THEMES} from '../types/settings'


const emit=defineEmits<{paste:[id:string];'toggle-keys':[];diagnostics:[];ssh:[];editSsh:[id:string]}>()
const tabs=useTabStore(),settings=useSettingsStore(),menuVisible=ref(false)
const menuRef=ref<InstanceType<typeof Menu>|null>(null),buttonRef=ref<HTMLButtonElement>()
const themeLabels:Record<string,string>={dark:'深色',light:'浅色','deep-blue':'深蓝'}
function closeTab(id:string){tabs.closeTab(id)}
function closeBatch(ids:string[]){ids.forEach(id=>tabs.closeTab(id))}
function create(){menuVisible.value=false;tabs.createStartTab()}
function action(type:string){menuVisible.value=false;window.dispatchEvent(new CustomEvent(type,{detail:{panelId:tabs.activeTabId}}))}
function editSsh(){menuVisible.value=false;emit('editSsh',tabs.activeTabId)}
function ssh(){menuVisible.value=false;emit('ssh')}
function setTheme(value:string){settings.updateTheme(value);settings.settings.terminal.theme=value==='light'?'uniterm-light':'uniterm-dark';settings.save()}
</script>
<style scoped>
.app-header {
  display: flex;
  align-items: center;
  height: 2.75rem;
  padding: 0 0.5rem;
  gap: 0.125rem;
  background: var(--bg-elevated);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  position: relative;
  z-index: 10;

}

.header-tabs {
  display: flex;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  justify-content: flex-start;
  align-items: center;
}

.header-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 1.75rem;
  padding: 0.3125rem 0.5rem;
  font-family: var(--font-ui);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  transition: all 0.15s ease;
  white-space: nowrap;
  flex-shrink: 0;

}

.header-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.header-btn.active {
  background: var(--bg-active, var(--bg-hover));
  color: var(--accent, var(--text-primary));
}

.header-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.header-btn:disabled:hover {
  background: transparent;
  color: var(--text-secondary);
}

.header-btn .el-icon {
  font-size: 0.875rem;
}

.app-header :deep(.window-controls) {

}

/* ── Settings dropdown menu ── */
.settings-wrap {
  position: relative;
  flex-shrink: 0;

}

</style>
