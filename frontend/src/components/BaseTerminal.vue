<!-- Adapted from ys-ll/uniterm (Apache-2.0), FnTerminal 2026-10-08: Wails calls replaced by authenticated fnOS WebSocket/PTY; desktop protocols and AI removed. -->
<template>
  <div class="base-terminal terminal-pane" :hidden="!active" :data-session-id="id">
    <div class="terminal-area" @contextmenu="onContextMenu"><div ref="terminalRef" class="terminal-host"></div></div>
    <div v-show="searchVisible" class="terminal-search-bar">
      <input ref="searchInputRef" v-model="searchText" class="search-input" placeholder="搜索终端内容" @input="searchNext" @keydown.enter.prevent="searchNext" @keydown.escape="closeSearch" />
      <button class="search-btn" @click="searchPrev" title="上一个"><ChevronUp :size="14" /></button>
      <button class="search-btn" @click="searchNext" title="下一个"><ChevronDown :size="14" /></button>
      <button class="search-btn" @click="closeSearch" title="关闭搜索"><X :size="14" /></button>
    </div>
    <Menu ref="menuRef" v-model:visible="menuVisible">
      <MenuItem :class="{disabled:!hasSelection}" shortcut="Ctrl+Shift+C" @click="copy">复制选区</MenuItem>
      <MenuItem shortcut="Ctrl+Shift+V" @click="paste">粘贴文字…</MenuItem>
      <MenuItem @click="selectAll">全选终端内容</MenuItem>
      <MenuDivider />
      <MenuItem shortcut="Ctrl+Shift+F" @click="openSearch">搜索文字</MenuItem>
      <MenuItem @click="exportContent">导出终端内容</MenuItem>
      <MenuItem @click="clear">清除滚动历史</MenuItem>
      <MenuDivider /><MenuItem @click="reconnect">重新连接当前会话</MenuItem>
    </Menu>
    <div v-if="status==='error'||status==='disconnected'" class="session-notice"><span class="session-message">{{reason}}</span><div class="session-actions"><button @click="reconnect">重新连接</button><button v-if="currentTab?.mode==='nas'" @click="editConnection">编辑连接</button><button @click="diagnostics">查看诊断</button></div></div>
  </div>
</template>
<script setup lang="ts">
import {computed,ref,onMounted,onBeforeUnmount,watch,nextTick} from 'vue'
import {Terminal} from '@xterm/xterm'
import {FitAddon} from '@xterm/addon-fit'
import {SearchAddon} from '@xterm/addon-search'
import {ChevronUp,ChevronDown,X} from '@lucide/vue'
import '@xterm/xterm/css/xterm.css'
import Menu from './Menu.vue'
import MenuItem from './MenuItem.vue'
import MenuDivider from './MenuDivider.vue'
import {getXtermTheme} from '../composables/useTerminal'
import {applyMobileCtrl} from '../utils/mobileCtrlKey'
import {normalizePastedText} from '../utils/terminalPaste'
import {useSettingsStore} from '../stores/settingsStore'
import {useTabStore} from '../stores/tabStore'
import {TerminalTransport} from '../services/transport'
import {registerWriter} from '../services/sessionWriter'
const props=defineProps<{id:string;active:boolean}>()
const settings=useSettingsStore(),tabs=useTabStore()
const terminalRef=ref<HTMLDivElement>(),searchInputRef=ref<HTMLInputElement>(),menuRef=ref<InstanceType<typeof Menu>|null>(null)
const menuVisible=ref(false),hasSelection=ref(false),searchVisible=ref(false),searchText=ref(''),status=ref('connecting'),reason=ref('')
let term:Terminal,fitAddon:FitAddon,searchAddon:SearchAddon,transport:TerminalTransport,observer:ResizeObserver,unregister:()=>void
let reconnectGeneration=0
function applyTerminalTheme(name:string){const theme=getXtermTheme(name);term.options.theme=theme;terminalRef.value?.style.setProperty('--terminal-bg',theme.background)}
function resize(){if(!term||!props.active||!terminalRef.value?.clientWidth)return;fitAddon.fit();transport.resize(term.cols,term.rows)}
function reconnect(){
 menuVisible.value=false;reason.value='';status.value='preparing';const current=tabs.tabs.find(t=>t.id===props.id);if(current){current.status='preparing';current.problem=undefined}const generation=++reconnectGeneration;transport.stop()
 // Drain pending old output before a full reset, so the new Shell never appends
 // its prompt to the old Shell's cursor or inherits alternate-screen modes.
 term.write('',()=>{if(generation!==reconnectGeneration)return;term.reset();searchAddon.clearDecorations();searchText.value='';searchVisible.value=false;const tab=tabs.tabs.find(t=>t.id===props.id);if(!tab)return;fitAddon.fit();void transport.connect(tab.mode,term.cols,term.rows,tab.sshPort,tab.connection)})
}
const currentTab=computed(()=>tabs.tabs.find(t=>t.id===props.id))
function editConnection(){window.dispatchEvent(new CustomEvent('fnterminal:edit-connection',{detail:props.id}))}
function diagnostics(){window.dispatchEvent(new CustomEvent('fnterminal:diagnostics'))}
function onContextMenu(e:MouseEvent){e.preventDefault();hasSelection.value=!!term?.hasSelection();menuRef.value?.openAt(e.clientX,e.clientY)}
async function copy(){menuVisible.value=false;const text=term?.getSelection();if(!text)return;try{await navigator.clipboard.writeText(text)}catch{window.dispatchEvent(new CustomEvent('fnterminal:copy-text',{detail:text}))}}
function paste(){menuVisible.value=false;window.dispatchEvent(new CustomEvent('fnterminal:paste',{detail:props.id}))}
function pasteText(text:string){if(new TextEncoder().encode(text).length>64*1024){reason.value='单次输入最多 64 KiB';status.value='error';return}term.paste(normalizePastedText(text));term.scrollToBottom()}
function selectAll(){menuVisible.value=false;term.selectAll()}
function openSearch(){menuVisible.value=false;searchVisible.value=true;nextTick(()=>searchInputRef.value?.focus())}
function searchNext(){searchAddon.findNext(searchText.value)}
function searchPrev(){searchAddon.findPrevious(searchText.value)}
function closeSearch(){searchVisible.value=false;searchAddon.clearDecorations();term.focus()}
function clear(){menuVisible.value=false;term.clear()}
function exportContent(){menuVisible.value=false;const text=Array.from({length:term.buffer.active.length},(_,i)=>term.buffer.active.getLine(i)?.translateToString(true)||'').join('\n');const url=URL.createObjectURL(new Blob([text],{type:'text/plain;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download='FnTerminal.txt';a.click();URL.revokeObjectURL(url)}
function onAction(e:Event){const event=e as CustomEvent;if(event.detail?.panelId!==props.id)return;switch(event.type){case 'panel:reconnect':reconnect();break;case 'terminal:open-search':openSearch();break;case 'terminal:export':exportContent();break;case 'terminal:reset-output':term.clear();break;case 'terminal:clear-scrollback':clear();break}}
function onPaste(e:Event){const event=e as CustomEvent;if(event.detail?.id===props.id)pasteText(event.detail.text)}
onMounted(()=>{
 term=new Terminal({fontSize:settings.settings.terminal.fontSize,fontFamily:'"JetBrains Mono Variable", Menlo, Consolas, "Noto Sans Mono CJK SC", monospace',theme:getXtermTheme(settings.settings.terminal.theme),scrollback:3000,screenReaderMode:true,cursorBlink:true,allowProposedApi:false})
 fitAddon=new FitAddon();searchAddon=new SearchAddon();term.loadAddon(fitAddon);term.loadAddon(searchAddon);term.open(terminalRef.value!);applyTerminalTheme(settings.settings.terminal.theme);fitAddon.fit()
 transport=new TerminalTransport((bytes,ack)=>term.write(bytes,ack),(state,message,problem)=>{if(state==='notice'){window.dispatchEvent(new CustomEvent('fnterminal:notice',{detail:message}));return}status.value=state;if(message)reason.value=message;const tab=tabs.tabs.find(t=>t.id===props.id);if(tab){tab.status=state;tab.problem=problem;}if(state==='connected'){resize();if(props.active)term.focus()}})
 unregister=registerWriter(props.id,data=>transport.input(data))
 term.onData(data=>transport.input(applyMobileCtrl(data)??data))
 term.attachCustomKeyEventHandler(e=>{if(e.type!=='keydown')return true;if((e.ctrlKey||e.metaKey)&&e.shiftKey){switch(e.key.toLowerCase()){case 'c':void copy();return false;case 'v':paste();return false;case 'f':openSearch();return false}}return true})
 observer=new ResizeObserver(resize);observer.observe(terminalRef.value!);void transport.connect(tabs.tabs.find(t=>t.id===props.id)?.mode||'application',term.cols,term.rows,tabs.tabs.find(t=>t.id===props.id)?.sshPort,tabs.tabs.find(t=>t.id===props.id)?.connection);resize()
 for(const name of ['panel:reconnect','terminal:open-search','terminal:export','terminal:reset-output','terminal:clear-scrollback'])window.addEventListener(name,onAction)
 window.addEventListener('fnterminal:send-text',onPaste)
 void document.fonts.ready.then(()=>{if(term)resize()})
})
watch(()=>tabs.tabs.find(t=>t.id===props.id)?.connectionRevision,()=>{if(transport)reconnect()})
watch(()=>props.active,active=>{if(active)nextTick(()=>{resize();term?.focus()})})
watch(()=>settings.settings.terminal.fontSize,size=>{if(term){term.options.fontSize=size;nextTick(resize)}})
watch(()=>settings.settings.terminal.theme,theme=>{if(term)applyTerminalTheme(theme)})
onBeforeUnmount(()=>{reconnectGeneration++;transport?.stop();observer?.disconnect();unregister?.();term?.dispose();for(const name of ['panel:reconnect','terminal:open-search','terminal:export','terminal:reset-output','terminal:clear-scrollback'])window.removeEventListener(name,onAction);window.removeEventListener('fnterminal:send-text',onPaste)})
</script>
<style scoped>
.base-terminal {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
.terminal-area {
  flex: 1;
  display: flex;
  flex-direction: row;
  align-items: stretch;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
}
.terminal-host {
  flex: 1;
  min-width: 0;
  min-height: 0;
  /* `clip` (not `hidden`): a hidden box is still programmatically scrollable,
   * so a focus()-driven auto-scroll during a resize transition could leave a
   * stale scrollTop here and shove the whole .xterm up, leaving blank space
   * below (it is never scrolled deliberately — xterm scrolls its own layers).
   * `clip` makes the box a non-scroll-container, so the offset cannot exist. */
  overflow: clip;
  position: relative;
}

/* Search bar */
.terminal-search-bar {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.25rem;
  /* 跟应用主题走（与 Zmodem 面板、终端建议弹窗等悬浮控件一致）；
     保留 88% 不透明度，让 blur 透出一点终端内容。 */
  background: color-mix(in srgb, var(--bg-surface) 88%, transparent);
  backdrop-filter: blur(0.5rem);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 0.25rem 0.375rem;
  z-index: 50;
}
.search-input {
  width: 10rem;
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-family: var(--font-ui);
  font-size: 0.75rem;
  padding: 0.125rem 0.25rem;
}
.search-input::placeholder {
  color: var(--text-muted);
}
.search-count {
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  color: var(--text-muted);
  white-space: nowrap;
  min-width: 2rem;
  text-align: center;
}
.search-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 1.375rem;
  height: 1.375rem;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  transition: all 0.15s;
}
.search-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.search-btn.active {
  color: var(--accent);
  background: var(--accent-subtle);
}
.terminal-area :deep(.xterm) {
  width: 100%;
  height: 100%;
  display: block;
  box-sizing: border-box;
  /* 左/上/下 0.5rem，右侧不留：那 0.875rem 的滚动条轨道本身已把文本挡开
     （文本右缘与轨道间还有 0.125rem），右 padding 只会把整条滚动条往左推、
     在轨道外侧留一条空白。 */
  padding: 0.5rem 0 0.5rem 0.5rem;
}
/* 0.5rem padding（右 0）那圈用终端背景色，而不是应用主题色（--bg-base）。
   v5 时 xterm 把终端色内联在 .xterm-viewport 上，而它 absolute inset:0
   盖满 padding box，边缘因此自带终端色；v6 改成内联到 .xterm-scrollable-element，
   该元素止于 padding 内侧，边缘便露出 .xterm 自身的应用主题色 ——
   深色应用 + 浅色终端时就是一圈深色边框。--terminal-bg 由
   applyTerminalBgVar() 写在终端根元素上，取不到时回退旧行为。 */
.terminal-area :deep(.xterm),
.terminal-area :deep(.xterm-viewport) {
  background: var(--terminal-bg, var(--bg-base));
}
/* v6 起 .xterm-viewport 只是 overview ruler 的定位锚点，不再是滚动容器；
   留着 overflow-y: scroll 会多出一条空的原生轨道。终端滚动条样式见
   style.css 里的 .xterm-scrollable-element > .scrollbar 规则。 */
.terminal-area :deep(.xterm-viewport) {
  overflow: hidden;
}

.drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--scrim);
  pointer-events: none;
}
.drop-overlay span {
  font-size: 0.875rem;
  color: var(--text-primary);
  padding: 0.75rem 1.5rem;
  border: 0.125rem dashed var(--border-hover);
  border-radius: var(--radius-sm);
}
</style>
