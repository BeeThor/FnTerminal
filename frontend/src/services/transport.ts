import type {SSHConnection} from './connections'
const renewalInterval = 20_000
const renewalRetryDelay = 5_000
// Browser-initiated failures use application close codes.
const clientPolicyClose = 4008

// fnOS transport: prepare over HTTP, attach once, then wait for actual PTY readiness.
export interface ConnectionProblem {code:string;message:string;detail?:string;phase:string}
export type TerminalMode='application'|'nas'
export class TerminalTransport {
 socket:WebSocket|null=null
 private renewal:ReturnType<typeof setTimeout>|undefined
 private controller:AbortController|undefined
 private generation=0
 private failure:ConnectionProblem|null=null
 private ready=false
 private sshPort=22
 private sshHost='127.0.0.1'
 private sshUser=''
 private mode:TerminalMode='application'
 private size={cols:100,rows:28}
 constructor(private output:(bytes:Uint8Array,ack:()=>void)=>void,private state:(status:string,message?:string,problem?:ConnectionProblem)=>void){}
 private problem(code:string,message:string,phase:string,detail=''){const p={code,message,phase,detail};this.failure=p;this.state('error',message,p)}
 private async request(path:string,options:RequestInit,generation:number){
  const controller=this.controller!;const timer=setTimeout(()=>controller.abort(),10000)
  try{const response=await fetch('/app/fnterminal'+path,{credentials:'same-origin',cache:'no-store',redirect:'error',...options,signal:controller.signal});if(generation!==this.generation)throw Error('cancelled');const data=await response.json().catch(()=>({code:'invalid_response',message:'服务响应异常，请刷新页面或重新登录NAS'}));if(!response.ok)throw data;return data}finally{clearTimeout(timer)}
 }
 async connect(mode:TerminalMode=this.mode,cols=this.size.cols,rows=this.size.rows,sshPort=this.sshPort,connection?:SSHConnection){
  this.stop();const generation=this.generation;this.mode=mode;this.sshPort=connection?.port??sshPort;if(connection){this.sshHost=connection.host;this.sshUser=connection.username}this.size={cols:Math.max(2,Math.min(500,cols)),rows:Math.max(2,Math.min(200,rows))};this.failure=null;this.ready=false;this.controller=new AbortController();this.state('preparing')
  let phase='登录状态检查'
  try{
   const bootstrap=await this.request('/api/bootstrap',{method:'GET'},generation)
   if(bootstrap.protocol!==2)throw {code:'version_mismatch',message:'前后端版本不一致，请刷新页面或重新安装应用'}
   phase='连接准备'
   const spec=await this.request('/api/connect',{method:'POST',headers:{'Content-Type':'application/json','X-Fnterminal-Csrf':bootstrap.csrf},body:JSON.stringify({mode,...this.size,...(mode==='nas'?{sshPort:this.sshPort,sshHost:this.sshHost,sshUser:this.sshUser}: {})})},generation)
   if(generation!==this.generation)return
   const ws=new WebSocket(`${location.protocol==='https:'?'wss:':'ws:'}//${location.host}/app/fnterminal/ws?ticket=${encodeURIComponent(spec.ticket)}`)
   this.socket=ws;ws.binaryType='arraybuffer';this.state('connecting')
   let sessionID = ''
   let serverRenewal = false, renewing = false, renewalFailed = false, finished = false
   const startupTimer=setTimeout(()=>{if(generation===this.generation&&!sessionID){this.problem('startup_timeout','终端启动超时，请查看连接诊断','启动终端');ws.close(clientPolicyClose,'startup timeout')}},10000)
   ws.onopen=()=>{if(this.socket===ws)this.state('starting')}
   const renew=async()=>{
    if (!sessionID || renewing || !this.ready || this.socket !== ws || ws.readyState !== WebSocket.OPEN) return
    clearTimeout(this.renewal)
    renewing = true
    let status: number | undefined
    try {
     const response = await fetch('/app/fnterminal/lease', {
      method: 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
      headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: sessionID}),
      signal: AbortSignal.timeout(10_000),
     })
     status = response.status
    } catch {
     // The service enforces the existing lease deadline while we retry.
    } finally {
     renewing = false
    }
    if (this.socket !== ws || !this.ready || ws.readyState !== WebSocket.OPEN) return
    if (status === 401 || status === 403) {
     this.problem('lease_expired', 'NAS登录态已失效，请重新登录', '会话续期')
     ws.close(clientPolicyClose, 'authentication failed')
     return
    }
    renewalFailed = status !== 204
    if (renewalFailed || !serverRenewal) {
     this.renewal = setTimeout(() => void renew(), renewalFailed ? renewalRetryDelay : renewalInterval)
    }
   }
   ws.onmessage=event=>{
    if(this.socket!==ws)return
    if(event.data instanceof ArrayBuffer){const bytes=new Uint8Array(event.data);this.output(bytes,()=>{if(this.socket===ws)this.send({type:'ack',bytes:bytes.byteLength})});return}
    try {
     const message = JSON.parse(event.data)
     if (message.type === 'ready') {
      if (sessionID) return
      if (typeof message.id !== 'string' || !message.id) throw Error('Missing session ID')
      clearTimeout(startupTimer)
      sessionID = message.id
      serverRenewal = message.renewal === 'server'
      this.ready = true
      this.failure = null
      this.state('connected')
      void renew()
     } else if (message.type === 'renew' && serverRenewal) {
      void renew()
     } else if (message.type === 'exit') {
      this.ready = false
      clearTimeout(this.renewal)
      if (message.exitCode !== 0) {
       this.problem(message.mode==='nas'?'ssh_exit':'shell_exit',message.mode==='nas'?'SSH会话异常结束，请查看终端中的具体提示':'终端进程异常退出',message.mode==='nas'?'SSH会话':'终端进程',`退出码 ${message.exitCode}；终端输出保留了具体提示。`)
      } else {
       finished = true
       this.state('disconnected', '终端已正常退出')
      }
     } else if (message.type === 'error') {
      clearTimeout(startupTimer)
      this.problem(message.code||'startup_failed',message.message||'终端启动失败','启动终端',message.detail||'')
     }
    } catch {
     this.problem('protocol', '服务协议响应异常', 'WebSocket')
     ws.close(4002, 'protocol error')
    }
   }
   ws.onerror=()=>{if(this.socket===ws&&!sessionID&&!this.failure)this.problem('websocket_failed','WebSocket连接失败，请打开连接诊断','WebSocket握手')}
   ws.onclose=event=>{
    clearTimeout(startupTimer);if(this.socket!==ws)return;clearTimeout(this.renewal);this.socket=null
    this.ready=false;if(finished||this.failure)return // retain the real error, never replace it with a generic close
    if(!sessionID){this.problem('startup_closed','服务在终端准备完成前结束了连接','启动终端',`WebSocket close ${event.code}`);return}
    const reason=event.reason||''
    const cause:Record<string,[string,string,string]>={'idle timeout':['idle_timeout','会话因长时间没有输入或输出而结束','空闲超时'],'session timeout':['session_timeout','会话达到最长使用时间','会话超时'],'authentication lease expired':renewalFailed?['lease_unavailable','无法续期NAS登录态，会话已结束，请检查网络和登录状态','会话续期']:['lease_expired','NAS登录态续期已过期，请重新登录','会话续期'],'service shutdown':['service_stopped','应用服务已停止或正在升级','应用服务']}
    if(cause[reason]){const [code,message,phase]=cause[reason];this.problem(code,message,phase,'重新连接将创建新会话，不恢复已结束的进程。');return}
    if(event.code===1009){this.problem('frame_limit','传输数据超出限制，会话已关闭','传输限制');return}
    if(event.code===1008){this.problem('policy_closed','会话因安全策略或流量限制被关闭，请查看诊断','会话限制');return}
    if(event.code===1006){this.problem('connection_lost','应用服务或网络连接中断，请重试或查看诊断','服务连接')}else this.state('disconnected','终端已结束，重新连接会创建新的会话。')
   }
  }catch(error:any){if(generation!==this.generation)return;this.problem(error.code||'network',error.message||'无法连接服务，请检查NAS登录状态',phase,error.detail||'')}
 }
 send(frame:object){const ws=this.socket;if(!ws||ws.readyState!==WebSocket.OPEN)return false;if(ws.bufferedAmount>128*1024){this.problem('input_backpressure','输入队列已满，会话已关闭','发送输入');ws.close(clientPolicyClose,'input backpressure');return false}ws.send(JSON.stringify(frame));return true}
 input(data:string){
  if(!this.ready||this.failure||this.socket?.readyState!==WebSocket.OPEN){this.state('notice','终端尚未连接，输入未发送',this.failure||{code:'not_connected',message:'终端尚未连接',phase:'发送输入'});return false}
  if(new TextEncoder().encode(data).byteLength>64*1024){this.state('notice','单次输入最多64 KiB，请分段粘贴',{code:'input_limit',message:'输入过大，未发送',phase:'发送输入'});return false}
  let chunk='';for(const char of data){chunk+=char;if(chunk.length>=2048){if(!this.send({type:'input',data:chunk}))return false;chunk=''}};return !chunk||this.send({type:'input',data:chunk})
 }
 resize(cols:number,rows:number){const size={cols:Math.max(2,Math.min(500,cols)),rows:Math.max(2,Math.min(200,rows))};if(size.cols===this.size.cols&&size.rows===this.size.rows)return;this.size=size;this.send({type:'resize',...size})}
 stop(){this.ready=false;this.generation++;this.controller?.abort();clearTimeout(this.renewal);const ws=this.socket;this.socket=null;ws?.close(1000,'terminal closed')}
}
