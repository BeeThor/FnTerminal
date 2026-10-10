import {validSSHHost,validateConnection,type SavedConnection,type SSHConnection} from './connections'
export const MAX_BACKUP_BYTES=64*1024
const fields=new Set(['name','host','port','username'])
function portable(c:SSHConnection){return {name:c.name.trim(),host:c.host.trim(),port:c.port,username:c.username.trim()}}
export function exportProfiles(profiles:SavedConnection[]):string{return JSON.stringify({app:'FnTerminal',schemaVersion:1,connections:profiles.map(portable)},null,2)+'\n'}
function key(c:SSHConnection):string{const raw=c.host.trim().replace(/^\[|\]$/g,'');const host=raw.includes(':')?new URL(`http://[${raw}]/`).hostname:raw.toLowerCase();return JSON.stringify([host,c.port,c.username.trim()])}
export function parseProfileBackup(text:string):SSHConnection[]{
 if(new TextEncoder().encode(text).byteLength>MAX_BACKUP_BYTES)throw Error('配置文件最多64 KiB')
 let data:any;try{data=JSON.parse(text)}catch{throw Error('配置文件不是有效JSON')}
 if(!data||data.app!=='FnTerminal'||data.schemaVersion!==1||!Array.isArray(data.connections)||data.connections.length>30||Object.keys(data).some(k=>!['app','schemaVersion','connections'].includes(k)))throw Error('请选择FnTerminal导出的配置文件，最多30个连接')
 return data.connections.map((c:any,index:number)=>{if(!c||typeof c!=='object'||Array.isArray(c)||Object.keys(c).some(k=>!fields.has(k)))throw Error(`第${index+1}个连接包含不支持的字段；文件不得包含密码、令牌或密钥`);const error=validateConnection(c);if(error||!validSSHHost(c.host))throw Error(`第${index+1}个连接：${error}`);return portable(c)})
}
export function mergeProfiles(existing:SavedConnection[],imported:SSHConnection[],newId:()=>string):{profiles:SavedConnection[];added:number;skipped:number}{
 const result=existing.map(c=>({...c}));const known=new Set(result.map(key));let skipped=0
 for(const c of imported){const fingerprint=key(c);if(known.has(fingerprint)){skipped++;continue}if(result.length>=30)throw Error('合并后超过30个连接，请先删除不需要的配置');result.push({...portable(c),id:newId()});known.add(fingerprint)}
 return {profiles:result,added:result.length-existing.length,skipped}
}
