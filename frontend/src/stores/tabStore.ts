import {reactive} from 'vue'
import type {TerminalTab} from '../types/terminal'
import type {SSHConnection} from '../services/connections'
import {mobileCtrlArmed} from '../utils/mobileCtrlKey'
let next=0
const store=reactive({tabs:[] as TerminalTab[],activeTabId:'',draggingTabId:null as string|null,
 createStartTab(mode:'application'|'nas'='application',sshPort=22,connection?:SSHConnection){if(mode==='nas'&&(!Number.isInteger(sshPort)||sshPort<1||sshPort>65535)){window.dispatchEvent(new CustomEvent('fnterminal:notice',{detail:'SSH端口必须是1到65535的整数'}));return null}if(store.tabs.length>=6){window.dispatchEvent(new CustomEvent('fnterminal:notice',{detail:'每位管理员最多打开 6 个终端'}));return null}const id=`terminal-${++next}`;const tab:TerminalTab={id,name:mode==='nas'?(connection?.name||`SSH :${sshPort} · ${next}`):`终端 ${next}`,mode,connection:connection?{...connection}:undefined,connectionRevision:0,sshPort:mode==='nas'?sshPort:undefined,problem:undefined,type:'terminal',panelId:id,locked:false,status:'connecting'};store.tabs.push(tab);store.setActiveTab(id);return tab},
 updateConnection(id:string,connection:SSHConnection){const tab=store.tabs.find(t=>t.id===id&&t.mode==='nas');if(!tab)return false;tab.connection={...connection};tab.sshPort=connection.port;tab.name=connection.name;tab.connectionRevision++;return true},
 setActiveTab(id:string){if(store.tabs.some(t=>t.id===id)){store.activeTabId=id;mobileCtrlArmed.value=false}},
 renameTab(id:string,name:string){const t=store.tabs.find(t=>t.id===id);if(t)t.name=name.slice(0,80)},
 toggleTabLock(id:string){const t=store.tabs.find(t=>t.id===id);if(t)t.locked=!t.locked},
 setDraggingTabId(id:string|null){store.draggingTabId=id},
 moveTab(from:number,to:number){if(from<0||from>=store.tabs.length||to<0||to>=store.tabs.length)return;const [t]=store.tabs.splice(from,1);store.tabs.splice(to,0,t)},
 closeTab(id:string){const i=store.tabs.findIndex(t=>t.id===id);if(i<0||store.tabs[i].locked)return;store.tabs.splice(i,1);if(store.activeTabId===id)store.setActiveTab(store.tabs[Math.min(i,store.tabs.length-1)]?.id||'');if(!store.tabs.length)store.activeTabId=''},


})
export const useTabStore=()=>store
