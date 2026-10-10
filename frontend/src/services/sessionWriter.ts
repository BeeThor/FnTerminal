// Browser replacement for Uniterm's Wails session writer; no local credentials.
const writers=new Map<string,(data:string)=>boolean>()
export function registerWriter(id:string,write:(data:string)=>boolean){writers.set(id,write);return()=>writers.delete(id)}
export function queuedSessionWrite(id:string,data:string){return writers.get(id)?.(data)??false}
