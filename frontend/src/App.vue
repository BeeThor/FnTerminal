<template>
  <div class="app-container">
    <AppHeader @paste="openComposer" @toggle-keys="showKeys=!showKeys" @diagnostics="openDiagnostics" @ssh="openSsh" @edit-ssh="editSsh" />
    <main class="main-content">
      <BaseTerminal v-for="tab in tabs.tabs" :key="tab.id" :id="tab.id" :active="tab.id===tabs.activeTabId" />
      <div v-if="!tabs.tabs.length" class="empty-terminal"><img :src="appIcon" class="welcome-icon" alt="FnTerminal" /><h1>FnTerminal</h1><p>连接 NAS SSH，使用自己的 NAS 用户权限。</p><button @click="openSsh">配置 SSH 连接</button></div>
    </main>
    <MobileKeyBar v-if="showKeys&&tabs.activeTabId" :session-id="tabs.activeTabId" />
    <footer class="status-bar"><span class="status-dot" :class="active?.status" /><span class="status-label" :title="statusText">{{statusText}}</span><span v-if="active?.mode==='nas'&&active.connection" class="connection-target" :title="formatSSHEndpoint(active.connection)">{{formatSSHEndpoint(active.connection)}}</span><span class="account">{{active?.mode==='nas'?'SSH 二次认证':active?'高级诊断账户':'未连接'}} · {{tabs.tabs.length}} / 6</span><span class="encoding">UTF-8</span></footer>
    <ElDialog v-model="composer" title="输入文字 / 粘贴" @closed="focusTerminal" width="min(520px,92vw)" :close-on-click-modal="false">
      <p class="dialog-hint">支持中文输入和多行粘贴。文字会发送到打开此窗口时的终端，不自动添加回车。</p>
      <ElInput v-model="composeText" type="textarea" :rows="6" placeholder="输入文字，或长按使用系统粘贴菜单" aria-label="待发送文字" />
      <template #footer><ElButton @click="composer=false">取消</ElButton><ElButton type="primary" @click="sendText">发送文字</ElButton></template>
    </ElDialog>
    <ElDialog v-model="copyDialog" title="复制终端文字" width="min(520px,92vw)"><textarea ref="copyRef" class="copy-text" :value="copyText" readonly aria-label="可复制的终端文字"></textarea><template #footer><ElButton @click="copyRef?.select()">全选</ElButton><ElButton @click="copyFallback">复制</ElButton></template></ElDialog>
    <ConnectionDialog v-model:visible="sshOpen" :editing="editingConnection" :default-port="settings.settings.nasSshPort" @connect="connectSsh" />
    <DiagnosticsDialog v-model:visible="diagnosticOpen" :connection="active?.mode==='nas'?active.connection:undefined" :problem="active?.problem" :status-text="statusText" />
    <div v-if="notice" class="notice" role="status">{{notice}}</div>
  </div>
</template>
<script setup lang="ts">
import {ref,computed,onMounted,onBeforeUnmount,nextTick} from 'vue'
import {ElDialog,ElInput,ElButton} from 'element-plus'
import appIcon from './assets/app-icon.png'
import ConnectionDialog from './components/ConnectionDialog.vue'
import DiagnosticsDialog from './components/DiagnosticsDialog.vue'
import {rememberConnection,readLastConnection,formatSSHEndpoint,type SSHConnection} from './services/connections'
import AppHeader from './components/AppHeader.vue'
import BaseTerminal from './components/BaseTerminal.vue'
import MobileKeyBar from './components/MobileKeyBar.vue'
import {useTabStore} from './stores/tabStore'
import {useSettingsStore} from './stores/settingsStore'
import {isMobilePlatform} from './utils/platform'
import {closeActiveMenu} from './services/menuCoordinator'
const settings=useSettingsStore();const sshOpen=ref(false),editingTab=ref(''),editingConnection=ref<SSHConnection>()
function openSsh(){closeActiveMenu();editingTab.value='';editingConnection.value=undefined;sshOpen.value=true}
function editSsh(id:string){const tab=tabs.tabs.find(t=>t.id===id&&t.mode==='nas');if(!tab)return;closeActiveMenu();editingTab.value=id;editingConnection.value=tab.connection?{...tab.connection}:{name:tab.name,host:'127.0.0.1',port:tab.sshPort||22,username:''};sshOpen.value=true}
function connectSsh(connection:SSHConnection){settings.settings.nasSshPort=connection.port;settings.save();const updated=editingTab.value?tabs.updateConnection(editingTab.value,connection):tabs.createStartTab('nas',connection.port,connection);if(updated){const remembered=rememberConnection(connection);sshOpen.value=false;if(!connection.profileId)showNotice('配置未保存，本次使用临时连接。');else if(!remembered)showNotice('连接配置已保存，但未能记住最近使用的连接。')}}
function onEditSsh(e:Event){editSsh((e as CustomEvent).detail)}
const tabs=useTabStore()
const active=computed(()=>tabs.tabs.find(t=>t.id===tabs.activeTabId))
const statusText=computed(()=>({preparing:'检查登录状态',connecting:'连接WebSocket',starting:'启动终端',connected:active.value?.mode==='nas'?'终端通道已连接':'已连接',disconnected:'会话已结束',error:'连接异常'} as Record<string,string>)[active.value?.status||'']||'尚未打开终端')
const showKeys=ref(isMobilePlatform()||window.matchMedia('(pointer:coarse)').matches),composer=ref(false),composeText=ref(''),target=ref(''),notice=ref(''),copyDialog=ref(false),copyText=ref(''),copyRef=ref<HTMLTextAreaElement>()
const diagnosticOpen=ref(false)
function openDiagnostics(){closeActiveMenu();diagnosticOpen.value=true}
let noticeTimer:ReturnType<typeof setTimeout>
async function openComposer(id:string){closeActiveMenu();if(!id)return;target.value=id;composeText.value='';composer.value=true;try{const text=await navigator.clipboard.readText();if(composer.value&&target.value===id)composeText.value=text}catch{}}
function focusTerminal(){document.querySelector<HTMLElement>('.terminal-pane:not([hidden]) .xterm-helper-textarea')?.focus()}
function sendText(){if(new TextEncoder().encode(composeText.value).length>64*1024){showNotice('单次输入最多 64 KiB，请分段发送');return}window.dispatchEvent(new CustomEvent('fnterminal:send-text',{detail:{id:target.value,text:composeText.value}}));composer.value=false}
function showNotice(text:string){notice.value=text;clearTimeout(noticeTimer);noticeTimer=setTimeout(()=>notice.value='',4000)}
function onNotice(e:Event){showNotice((e as CustomEvent).detail)}
function onPaste(e:Event){void openComposer((e as CustomEvent).detail)}
function onCopy(e:Event){closeActiveMenu();copyText.value=(e as CustomEvent).detail;copyDialog.value=true;nextTick(()=>copyRef.value?.select())}
function copyFallback(){copyRef.value?.select();showNotice(document.execCommand('copy')?'已复制':'请使用系统复制菜单')}
function onKey(e:KeyboardEvent){if(e.isComposing||composer.value||copyDialog.value||diagnosticOpen.value||sshOpen.value)return;if(e.ctrlKey&&e.altKey&&/^[1-6]$/.test(e.key)){e.preventDefault();const tab=tabs.tabs[Number(e.key)-1];if(tab)tabs.setActiveTab(tab.id)}else if(e.ctrlKey&&e.shiftKey&&e.key.toLowerCase()==='t'){e.preventDefault();openSsh()}else if(e.ctrlKey&&e.shiftKey&&e.key.toLowerCase()==='w'){e.preventDefault();tabs.closeTab(tabs.activeTabId)}}
function resize(){const vv=window.visualViewport;if(vv?.scale===1)document.documentElement.style.setProperty('--fn-visible-height',`${vv.height}px`)}
onMounted(()=>{for(const [name,fn] of [['fnterminal:notice',onNotice],['fnterminal:paste',onPaste],['fnterminal:copy-text',onCopy]] as const)window.addEventListener(name,fn);window.addEventListener('keydown',onKey);window.visualViewport?.addEventListener('resize',resize);window.addEventListener('fnterminal:diagnostics',openDiagnostics);window.addEventListener('fnterminal:edit-connection',onEditSsh);window.addEventListener('fnterminal:new-connection',openSsh);resize();const last=readLastConnection();if(last&&settings.settings.autoConnect)tabs.createStartTab('nas',last.port,last);else openSsh()})
onBeforeUnmount(()=>{clearTimeout(noticeTimer);for(const [name,fn] of [['fnterminal:notice',onNotice],['fnterminal:paste',onPaste],['fnterminal:copy-text',onCopy]] as const)window.removeEventListener(name,fn);window.removeEventListener('keydown',onKey);window.visualViewport?.removeEventListener('resize',resize);window.removeEventListener('fnterminal:diagnostics',openDiagnostics);window.removeEventListener('fnterminal:edit-connection',onEditSsh);window.removeEventListener('fnterminal:new-connection',openSsh)})
</script>
