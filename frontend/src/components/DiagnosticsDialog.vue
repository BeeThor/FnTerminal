<template>
 <ElDialog :model-value="visible" title="连接诊断与日志" destroy-on-close class="diagnostics-dialog" width="min(760px,94vw)" :close-on-click-modal="false" @update:model-value="emit('update:visible',$event)">
  <section class="diagnostic-current"><span class="diagnostic-eyebrow">当前终端</span><strong>{{problem?.message||statusText}}</strong><span v-if="connection" class="diagnostic-endpoint">{{formatSSHEndpoint(connection)}}<template v-if="connection.username"> · {{connection.username}}</template></span><p v-if="problem?.detail">{{problem.detail}}</p><span v-if="problem" class="diagnostic-phase">{{problem.phase}} · {{problem.code}}</span></section>
  <p v-if="loading" class="dialog-hint" role="status">读取服务诊断…</p><p v-if="error" class="connection-error" role="alert">{{error}}</p>
  <div v-if="server" class="diagnostic-summary"><div><span>服务版本</span><strong>{{server.version||'未知'}}</strong></div><div><span>运行平台</span><strong>{{server.runtime||'未知'}}</strong></div><div><span>服务账户</span><strong>{{server.account||'未知'}}</strong></div><div><span>我的活动会话</span><strong>{{Number.isInteger(server.activeSessions)?server.activeSessions:'未知'}}</strong></div></div>
  <section v-if="server" class="diagnostic-events"><div class="diagnostic-events-heading"><strong>最近连接事件</strong><small>{{events.length}} 条</small></div><ol v-if="events.length"><li v-for="(event,index) in events" :key="index"><span class="diagnostic-event-dot" :class="{failed:/failed|rejected/.test(event.event||'')}"/><div><div><strong>{{eventName(event.event)}}</strong><time>{{formatTime(event.time)}}</time></div><span>{{event.code}}</span><p v-if="event.detail">{{event.detail}}</p></div></li></ol><p v-else class="dialog-hint">暂无事件；创建连接后可刷新查看。</p></section>
  <details v-if="record" class="diagnostic-raw"><summary>原始诊断数据</summary><pre class="diagnostic-record" role="log">{{record}}</pre></details>
  <template #footer><span class="diagnostic-privacy">不记录命令、密码或终端输出</span><ElButton :disabled="loading" @click="load">刷新诊断</ElButton><ElButton :disabled="!record" @click="download">下载诊断记录</ElButton></template>
 </ElDialog>
</template>
<script setup lang="ts">
import {computed,onBeforeUnmount,ref,watch} from 'vue'
import {ElDialog,ElButton} from 'element-plus'
import type {ConnectionProblem} from '../services/transport'
import {formatSSHEndpoint,type SSHConnection} from '../services/connections'
const props=defineProps<{visible:boolean;connection?:SSHConnection;problem?:ConnectionProblem;statusText:string}>()
const emit=defineEmits<{'update:visible':[value:boolean]}>()
const loading=ref(false),error=ref(''),record=ref(''),server=ref<Record<string,any>|null>(null)
const events=computed(()=>Array.isArray(server.value?.events)?server.value!.events.filter((e:any)=>e&&typeof e==='object'&&typeof e.event==='string'&&typeof e.time==='string').slice(-200).reverse():[])
let controller:AbortController|undefined;let generation=0
async function load(){controller?.abort();controller=new AbortController();const requestController=controller;const signal=requestController.signal;const current=++generation;loading.value=true;error.value='';const timer=setTimeout(()=>requestController.abort(),10000)
 try{const response=await fetch('/app/fnterminal/api/diagnostics',{credentials:'same-origin',cache:'no-store',redirect:'error',signal});const data=await response.json();if(current!==generation)return;if(!response.ok)throw Error(data.message||'读取服务诊断失败，请检查NAS登录状态');if(!data||typeof data!=='object'||Array.isArray(data))throw Error('服务返回了无效诊断数据');server.value=data;record.value=JSON.stringify({client:{origin:location.origin,opaqueOrigin:self.origin==='null',connection:props.problem||props.statusText,sshTarget:props.connection},server:data},null,2)}catch(e:any){if(current===generation){error.value=signal.aborted?'读取诊断超时，请检查网络后刷新':e.message||'读取诊断失败';server.value=null;record.value=''}}finally{clearTimeout(timer);if(current===generation)loading.value=false}}
function eventName(value:string){return ({service_start:'服务启动',connect_prepared:'连接准备完成',connect_failed:'连接检查失败',websocket_rejected:'连接被拒绝',session_ready:'终端通道就绪',session_exit:'终端进程退出',session_end:'会话结束'} as Record<string,string>)[value]||value||'连接事件'}
function formatTime(value:string){const date=new Date(value);return Number.isNaN(date.getTime())?'时间未知':date.toLocaleString()}
function download(){const url=URL.createObjectURL(new Blob([record.value],{type:'application/json;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download='FnTerminal-diagnostics.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)}
function cancel(){generation++;controller?.abort();loading.value=false}
watch(()=>props.visible,value=>{if(value){server.value=null;record.value='';void load()}else cancel()})
onBeforeUnmount(cancel)
</script>
