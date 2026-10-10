import zh from './locales/zh-CN.json'
export function useI18n() {
  return {t:(key:string, values:Record<string,unknown>={}) => {
    let text=key==='startTab.defaultName'?'新建应用终端':((zh as Record<string,string>)[key]||key)
    for(const [k,v] of Object.entries(values)) text=String(text).replaceAll(`{${k}}`,String(v))
    return text
  }}
}
