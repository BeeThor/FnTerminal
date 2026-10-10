// Module singleton: unlike <script setup>, this runs once for the application.
export interface ActiveMenu {el:()=>HTMLElement|null;close:()=>void}
let active:ActiveMenu|null=null
function outside(e:Event){const target=e.target as Element|null;if(target&&(!active?.el()?.contains(target))&&!target.closest('.menu-submenu'))closeActiveMenu()}
function keyboard(e:KeyboardEvent){if(e.key==='Escape')closeActiveMenu()}
export function activateMenu(menu:ActiveMenu){if(active&&active!==menu)active.close();if(!active){document.addEventListener('pointerdown',outside,true);document.addEventListener('keydown',keyboard);window.addEventListener('resize',closeActiveMenu)}active=menu}
export function deactivateMenu(menu:ActiveMenu){if(active!==menu)return;active=null;document.removeEventListener('pointerdown',outside,true);document.removeEventListener('keydown',keyboard);window.removeEventListener('resize',closeActiveMenu)}
export function closeActiveMenu(){const menu=active;if(menu){menu.close();deactivateMenu(menu)}}
