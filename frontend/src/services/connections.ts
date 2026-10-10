export interface SSHConnection {name:string;host:string;port:number;username:string;profileId?:string}
export interface SavedConnection extends SSHConnection {id:string}
const KEY='fnterminal.ssh-connections'
export function validSSHHost(raw:string):boolean {
 const host=raw.trim();if(!host||host.length>253)return false
 if(host.includes(':')||host.includes('[')||host.includes(']')){
  const ip=host.startsWith('[')&&host.endsWith(']')?host.slice(1,-1):host
  if(!/^[a-fA-F0-9:.]+$/.test(ip)||!ip.includes(':'))return false
  try{new URL(`http://[${ip}]/`);return true}catch{return false}
 }
 return /^[a-zA-Z0-9](?:[a-zA-Z0-9.-]{0,251}[a-zA-Z0-9])?$/.test(host)&&host.split('.').every(label=>label.length>=1&&label.length<=63&&!label.startsWith('-')&&!label.endsWith('-'))
}
export function formatSSHEndpoint(connection:SSHConnection):string {const host=connection.host.replace(/^\[|\]$/g,'');return `${host.includes(':')?`[${host}]`:host}:${connection.port}`}
export function validateConnection(connection:SSHConnection):string {
 if(typeof connection.name!=='string'||!connection.name.trim()||connection.name.trim().length>80)return '连接名称需要1到80个字符'
 if(typeof connection.host!=='string'||!validSSHHost(connection.host))return '请填写有效IP或域名；主机与SSH端口分别填写，不包含协议、路径或用户名'
 if(!Number.isInteger(connection.port)||connection.port<1||connection.port>65535)return 'SSH端口必须是1到65535的整数'
 if(typeof connection.username!=='string'||(connection.username.trim()&&!/^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,63}$/.test(connection.username.trim())))return 'SSH用户名格式无效；也可以留空在终端输入'
 return ''
}
export function readConnections():SavedConnection[]{try{const data=JSON.parse(localStorage.getItem(KEY)||'[]');if(!Array.isArray(data))return [];return data.filter(c=>c&&typeof c.id==='string'&&typeof c.name==='string'&&typeof c.host==='string'&&typeof c.username==='string'&&!validateConnection(c)).slice(0,30).map(clean)}catch{return []}}
function clean(c:SavedConnection):SavedConnection{return{id:c.id,name:c.name.trim(),host:c.host.trim(),port:c.port,username:c.username.trim()}}
export function writeConnections(list:SavedConnection[]){localStorage.setItem(KEY,JSON.stringify(list.slice(0,30).map(clean)))}

const LAST_KEY='fnterminal.last-ssh'
export function rememberConnection(connection:SSHConnection){try{if(connection.profileId)localStorage.setItem(LAST_KEY,connection.profileId);return true}catch{return false}}
export function readLastConnection():SSHConnection|undefined{try{const id=localStorage.getItem(LAST_KEY);const profile=readConnections().find(p=>p.id===id);return profile?{name:profile.name,host:profile.host,port:profile.port,username:profile.username,profileId:profile.id}:undefined}catch{return undefined}}
