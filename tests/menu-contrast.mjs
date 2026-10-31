import assert from 'node:assert/strict';

// Use actual browser styles: source-token checks cannot catch translucent
// surfaces composited over an independently coloured terminal.
export async function checkMenuContrast(page, folder) {
 const original = await page.evaluate(() => ({theme:document.documentElement.dataset.theme, mode:document.body.getAttribute('theme-mode')}));
 const metrics=[];
 async function check(selector, state) {
  const samples=await page.locator(selector).evaluateAll(menus => menus.map(menu => {
   const rgba=value=>value.match(/[\d.]+/g).map(Number);
   const over=(fg,bg)=>{const a=fg[3]??1;return fg.slice(0,3).map((v,i)=>v*a+bg[i]*(1-a));};
   const luminance=rgb=>rgb.map(v=>{v/=255;return v<=.04045?v/12.92:((v+.055)/1.055)**2.4}).reduce((n,v,i)=>n+v*[.2126,.7152,.0722][i],0);
   const bg=rgba(getComputedStyle(menu).backgroundColor);
   return {background:bg, rows:[...menu.querySelectorAll('.menu-item:not(.disabled)')].map(row=>{
    const style=getComputedStyle(row), surface=over(rgba(style.backgroundColor),bg);
    const label=row.querySelector('.menu-label')||row;
    const ink=over(rgba(getComputedStyle(label).color),surface);
    const a=luminance(ink),b=luminance(surface);
    return {text:label.textContent,contrast:(Math.max(a,b)+.05)/(Math.min(a,b)+.05)};
   })};
  }));
  assert.ok(samples.length, state+' menu exists');
  for(const sample of samples){
   assert.equal(sample.background[3]??1,1,state+' surface must be opaque');
   for(const row of sample.rows)assert.ok(row.contrast>=4.5,`${state}: ${row.text} contrast ${row.contrast}`);
  }
  metrics.push({state,samples});
 }
 try {
  for(const theme of ['light','dark'])for(const backdrop of ['#000','#fff']){
   await page.evaluate(({theme,backdrop})=>{
    document.documentElement.dataset.theme=theme;document.body.setAttribute('theme-mode',theme);
    let underlay=document.getElementById('contrast-underlay');
    if(!underlay){underlay=document.createElement('div');underlay.id='contrast-underlay';document.body.append(underlay);}
    Object.assign(underlay.style,{position:'fixed',inset:'0',zIndex:'1000',pointerEvents:'none',background:backdrop});
   },{theme,backdrop});
   await page.getByRole('button',{name:'终端菜单'}).click();
   await check('.conn-context-menu:visible',`${theme}/${backdrop}/toolbar`);
   await page.getByText('高级诊断',{exact:true}).hover();
   await page.locator('.menu-submenu:visible').waitFor();
   await check('.menu-submenu:visible',`${theme}/${backdrop}/submenu`);
   await page.getByText('独立应用账户 Shell',{exact:true}).hover();
   await page.waitForTimeout(150);
   await check('.menu-submenu:visible',`${theme}/${backdrop}/hover`);
   await page.screenshot({path:`${folder}/menu-${theme}-${backdrop==='#000'?'black':'white'}.png`});
   await page.keyboard.press('Escape');
   await page.locator('.tab-item').first().click({button:'right'});
   await check('.conn-context-menu:visible',`${theme}/${backdrop}/tab-context`);
   await page.keyboard.press('Escape');
  }
 } finally {
  await page.evaluate(original=>{document.getElementById('contrast-underlay')?.remove();document.documentElement.dataset.theme=original.theme;document.body.setAttribute('theme-mode',original.mode);},original);
 }
 return metrics;
}
