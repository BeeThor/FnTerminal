<template>
  <ElDialog :model-value="visible" :title="editing?'编辑 SSH 连接':'新建 SSH 连接'" class="ssh-connection-dialog" width="min(760px,94vw)" :close-on-click-modal="false" @update:model-value="closeDialog">
    <div class="connection-workspace">
      <aside class="connection-library" :class="{'is-empty':!profiles.length}">
        <div class="connection-library-heading"><span class="connection-library-title">已保存连接 <small>{{profiles.length}}</small></span><button class="connection-library-toggle" :aria-label="libraryExpanded?'收起已保存连接':'展开已保存连接'" :aria-expanded="libraryExpanded" @click="libraryExpanded=!libraryExpanded">已保存连接 <small>{{profiles.length}}</small><ChevronDown :size="15" :class="{expanded:libraryExpanded}" /></button><button class="connection-icon-button" aria-label="新建连接配置" title="新建连接配置" @click="newProfile"><Plus :size="16" /></button></div>
        <div class="connection-library-content" :class="{expanded:libraryExpanded}"><div class="connection-library-tools"><button class="connection-icon-button" aria-label="导入连接配置" title="合并导入连接配置" @click="importRef?.click()"><Upload :size="15" /></button><button class="connection-icon-button" :disabled="!profiles.length" aria-label="导出连接配置" title="导出连接配置，不含密码" @click="downloadProfiles"><Download :size="15" /></button><input ref="importRef" type="file" accept=".json,application/json" hidden aria-label="连接配置文件" @change="importProfiles" /></div>
        <ElInput v-if="profiles.length" v-model="profileSearch" placeholder="搜索名称或地址" aria-label="搜索已保存的连接" clearable class="connection-profile-search"><template #prefix><Search :size="14" /></template></ElInput>
        <div v-if="profiles.length" class="saved-connection-list">
          <button v-for="profile in filteredProfiles" :key="profile.id" class="saved-connection" :class="{'is-selected':selected===profile.id}" :aria-label="`选择连接 ${profile.name}`" :aria-pressed="selected===profile.id" :title="`${profile.name} · ${formatSSHEndpoint(profile)}${profile.username?' · '+profile.username:''}`" @click="selectProfile(profile.id)">
            <span class="saved-connection-icon"><Server :size="17" /></span><span class="saved-connection-copy"><strong>{{profile.name}}</strong><small :title="formatSSHEndpoint(profile)">{{formatSSHEndpoint(profile)}}</small><small v-if="profile.username">{{profile.username}}</small></span><Check v-if="selected===profile.id" :size="14" class="saved-connection-check" />
          </button>
          <p v-if="!filteredProfiles.length" class="connection-list-empty">没有匹配的连接</p>
        </div>
        <div v-else class="connection-library-empty"><Server :size="26" /><strong>连接收藏</strong><p>保存常用连接，下次直接选择。密码不会保存。</p></div>
        </div>
      </aside>
      <section class="connection-editor">
        <div class="connection-editor-heading"><div><span class="connection-section-title">连接设置</span><span class="connection-draft-badge" :class="{'is-dirty':dirty}">{{dirty?'有未保存修改':selected?'已保存':'新配置'}}</span></div><button v-if="selected" class="connection-icon-button connection-delete" aria-label="删除选中连接" title="删除选中连接" @click="removeProfile"><Trash2 :size="15" /></button></div>
        <div class="connection-form">
          <label for="connection-name">连接名称</label><ElInput id="connection-name" v-model="form.name" maxlength="80" aria-label="连接名称" placeholder="例如：我的 NAS" />
          <label for="ssh-host">SSH 主机 / IP</label><ElInput id="ssh-host" v-model="form.host" aria-label="SSH主机" placeholder="127.0.0.1" />
          <div class="connection-row"><div><label for="ssh-port">SSH端口</label><ElInputNumber id="ssh-port" v-model="form.port" :min="1" :max="65535" :precision="0" controls-position="right" aria-label="NAS SSH端口" /></div><div><label for="ssh-user">用户名（可选）</label><ElInput id="ssh-user" v-model="form.username" aria-label="SSH用户名" autocomplete="off" placeholder="留空则在终端输入" /></div></div>
        </div>
        <p class="dialog-hint">本机 NAS 使用 <code>127.0.0.1</code>，端口填写实际 SSH 端口，例如 <code>10022</code>。地址须从 NAS 可达；密码与主机指纹在终端确认。</p>
        <label class="connection-option"><input type="checkbox" v-model="settings.settings.autoConnect" @change="settings.save()" />打开应用时自动连接最近使用的配置</label>
        <p v-if="editing" class="dialog-hint">保存配置不影响现有会话；重新连接只结束当前标签的旧会话。</p>
        <div v-if="pendingAction" class="connection-draft-guard" role="alert"><strong>当前修改尚未保存</strong><span>继续编辑，或放弃修改后继续操作。</span><div><ElButton @click="pendingAction=null">继续编辑</ElButton><ElButton @click="discardDraft">放弃修改</ElButton></div></div>
        <div v-if="deleted" class="connection-undo" role="status"><span>已删除 {{deleted.name}}</span><button @click="undoDelete">撤销删除</button></div>
        <p v-if="transferNote" class="dialog-hint" role="status">{{transferNote}}</p>
        <p v-if="error" class="connection-error" role="alert">{{error}}</p>
        <div v-if="saved" class="connection-save-feedback" role="status" aria-live="polite"><span class="connection-feedback-icon"><Check :size="18" /></span><div><strong>配置已保存</strong><span>{{form.name}} · {{formatSSHEndpoint(form)}}</span><small>已加入连接列表，现有会话未改变。</small></div></div>
      </section>
    </div>
    <template #footer><ElButton @click="closeDialog(false)">取消</ElButton><ElButton @click="save">保存配置</ElButton><ElButton type="primary" @click="connect">{{editing?'保存并重新连接':'连接'}}</ElButton></template>
  </ElDialog>
</template>
<script setup lang="ts">
import {computed,reactive,ref,watch} from 'vue'
import {ElDialog,ElInput,ElInputNumber,ElButton} from 'element-plus'
import {Server,Plus,Search,Check,Trash2,Upload,Download,ChevronDown} from '@lucide/vue'
import {exportProfiles,parseProfileBackup,mergeProfiles,MAX_BACKUP_BYTES} from '../services/profileBackup'
import {useSettingsStore} from '../stores/settingsStore'
import {readConnections,writeConnections,validateConnection,formatSSHEndpoint,type SSHConnection,type SavedConnection} from '../services/connections'
const settings=useSettingsStore()
const props=defineProps<{visible:boolean;editing?:SSHConnection;defaultPort:number}>()
const emit=defineEmits<{'update:visible':[visible:boolean];connect:[connection:SSHConnection]}>()
const form=reactive<SSHConnection>({name:'本机 NAS',host:'127.0.0.1',port:22,username:''})
const profiles=ref<SavedConnection[]>(readConnections()),selected=ref(''),error=ref(''),saved=ref(false),profileSearch=ref(''),libraryExpanded=ref(false)
const filteredProfiles=computed(()=>{const query=profileSearch.value.trim().toLowerCase();return profiles.value.filter(p=>`${p.name} ${p.host} ${p.port} ${p.username}`.toLowerCase().includes(query))})
const baseline=ref(''),pendingAction=ref<null|(()=>void)>(null),deleted=ref<SavedConnection|null>(null),transferNote=ref(''),importRef=ref<HTMLInputElement>()
function fingerprint(){return JSON.stringify([form.name.trim(),form.host.trim(),form.port,form.username.trim()])}
const dirty=computed(()=>fingerprint()!==baseline.value)
function guard(action:()=>void){if(dirty.value){pendingAction.value=action;return}action()}
function discardDraft(){const action=pendingAction.value;pendingAction.value=null;action?.()}
function closeDialog(value:boolean){if(!value)guard(()=>emit('update:visible',false))}
function newId(){return crypto.randomUUID?.()||Array.from(crypto.getRandomValues(new Uint8Array(16)),v=>v.toString(16).padStart(2,'0')).join('')}
let dialogGeneration=0
function resetDraft(){baseline.value=fingerprint();pendingAction.value=null}
function downloadProfiles(){const url=URL.createObjectURL(new Blob([exportProfiles(profiles.value)],{type:'application/json'}));const a=document.createElement('a');a.href=url;a.download='FnTerminal-connections.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);transferNote.value='已导出连接地址与用户名，不包含密码或密钥'}
async function importProfiles(event:Event){const input=event.target as HTMLInputElement;const file=input.files?.[0];const generation=dialogGeneration;input.value='';if(!file)return;try{if(file.size>MAX_BACKUP_BYTES)throw Error('配置文件最多64 KiB');const imported=parseProfileBackup(await file.text());if(!props.visible||generation!==dialogGeneration)return;const result=mergeProfiles(readConnections(),imported,newId);writeConnections(result.profiles);profiles.value=result.profiles;transferNote.value=`已导入${result.added}个连接，跳过${result.skipped}个重复连接；原有连接保留`;error.value=''}catch(e:any){error.value=e.message||'导入失败，原配置保留'}}
function undoDelete(){const removed=deleted.value;if(!removed)return;try{const list=readConnections();if(list.length>=30)throw Error('连接列表已满，无法撤销');if(list.some(p=>p.id===removed.id))throw Error('连接已经恢复');writeConnections([...list,removed]);profiles.value=[...list,removed];deleted.value=null;transferNote.value='删除已撤销'}catch(e:any){error.value=e.message||'撤销失败，原配置保留'}}
watch(()=>props.visible,value=>{dialogGeneration++;if(!value)return;profiles.value=readConnections();selected.value=profiles.value.some(p=>p.id===props.editing?.profileId)?props.editing!.profileId!:'';profileSearch.value='';libraryExpanded.value=false;error.value='';saved.value=false;transferNote.value='';deleted.value=null;Object.assign(form,props.editing||{name:'本机 NAS',host:'127.0.0.1',port:props.defaultPort,username:''});resetDraft()})
watch(form,()=>{error.value='';saved.value=false})
function newProfile(){guard(()=>{selected.value='';saved.value=false;error.value='';Object.assign(form,{name:'本机 NAS',host:'127.0.0.1',port:props.defaultPort,username:''});resetDraft()})}
function selectProfile(id:string){if(id===selected.value)return;guard(()=>{const profile=profiles.value.find(p=>p.id===id);if(profile){selected.value=id;saved.value=false;error.value='';Object.assign(form,{name:profile.name,host:profile.host,port:profile.port,username:profile.username});resetDraft()}})}
function save(){
 error.value=validateConnection(form);if(error.value)return false
 try{
  const id=selected.value||newId()
  const profile={id,name:form.name.trim(),host:form.host.trim(),port:form.port,username:form.username.trim()}
  const next=profiles.value.map(p=>({...p}));const index=next.findIndex(p=>p.id===id)
  if(index<0){if(next.length>=30){error.value='最多保存30个连接，请先删除不需要的配置';return false}next.push(profile)}else next[index]=profile
  writeConnections(next);profiles.value=next;selected.value=id;saved.value=true;resetDraft();return true
 }catch{error.value='浏览器无法保存配置；仍可点击连接，使用临时配置';return false}
}
function removeProfile(){guard(()=>{const removed=profiles.value.find(p=>p.id===selected.value);const next=profiles.value.filter(p=>p.id!==selected.value);try{writeConnections(next);profiles.value=next;selected.value='';deleted.value=removed||null;saved.value=false;error.value='';resetDraft()}catch{error.value='删除失败：浏览器无法保存配置，原配置仍保留'}})}
function connect(){
 error.value=validateConnection(form);if(error.value)return
 const persistent=save()
 emit('connect',{name:form.name.trim(),host:form.host.trim(),port:form.port,username:form.username.trim(),profileId:persistent?selected.value:undefined})
}
</script>
