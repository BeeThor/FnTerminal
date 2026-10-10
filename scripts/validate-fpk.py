#!/usr/bin/env python3
"""Inspect FPKs without executing or extracting untrusted archive paths."""
import hashlib
import io
import json
import struct
import sys
import tarfile
from pathlib import Path, PurePosixPath
from PIL import Image

path, arch = Path(sys.argv[1]), sys.argv[2]
def check(condition, message):
    if not condition:
        raise SystemExit('FPK verification failed: '+message)
def safe(archive):
    names = set()
    for member in archive.getmembers():
        check(not PurePosixPath(member.name).is_absolute() and '..' not in PurePosixPath(member.name).parts, 'unsafe archive path')
        check(not member.issym() and not member.islnk(), 'unexpected link')
        check(member.name not in names, 'duplicate archive path')
        names.add(member.name)
    return names
with tarfile.open(path,'r:*') as package:
    names = safe(package)
    for required in ['manifest','config/privilege','config/resource','ICON.PNG','ICON_256.PNG','app.tgz','cmd/main','wizard']:
        check(required in names, 'missing '+required)
    read = lambda name: package.extractfile(name).read()
    manifest = dict(line.split('=',1) for line in read('manifest').decode().splitlines() if '=' in line)
    manifest = {k.strip():v.strip() for k,v in manifest.items()}
    check(manifest['appname']=='fnterminal' and manifest['platform']==arch, 'manifest architecture/name')
    check('service_port' not in manifest, 'TCP service port declared')
    privilege = json.loads(read('config/privilege'))
    check(privilege=={'defaults':{'run-as':'package'},'username':'fnterminal','groupname':'fnterminal'}, 'privilege expansion')
    check(json.loads(read('config/resource'))=={}, 'unexpected resources')
    app = read('app.tgz')
    check(manifest.get('checksum')==hashlib.md5(app).hexdigest(), 'fnpack app.tgz checksum mismatch')
    for size,name in [(64,'ICON.PNG'),(256,'ICON_256.PNG')]:
        with Image.open(io.BytesIO(read(name))) as icon:
            check(icon.size==(size,size) and icon.format=='PNG', 'icon dimensions/format')
            check(icon.mode=='RGBA' and icon.getchannel('A').getextrema()[0]==0, 'transparent logo alpha was lost')
        check(len(read(name))<=1024*1024,'icon size')
    for member in package.getmembers():
        if member.name.startswith('cmd/') and member.isfile(): check(member.mode&0o111, 'non-executable lifecycle script')
    with tarfile.open(fileobj=io.BytesIO(app),mode='r:gz') as payload:
        appnames = safe(payload)
        # Official fnpack uses paths relative to app/ (some versions include ./).
        files = {m.name.removeprefix('./'):m for m in payload.getmembers()}
        for file in ['bin/fnterminal','bin/nas-login','web/index.html','ui/config','licenses/creack-pty.txt','licenses/gorilla-websocket.txt','licenses/xterm.txt','licenses/addon-fit.txt','licenses/uniterm-Apache-2.0.txt','licenses/jetbrains-mono-OFL.txt','licenses/FnTerminal-NOTICE.txt','licenses/npm/douyinfe__semi-theme-default/LICENSE']:
            check(file in files,'missing payload '+file)
        # Every same-origin link emitted by Vite must be included in the archive.
        import re
        html=payload.extractfile(files['web/index.html']).read().decode()
        refs=re.findall(r'(?:src|href)="(/app/fnterminal/[^"]+)"',html)
        check(len(refs)>=2,'missing offline JS/CSS entry points')
        for url in refs: check('web/'+url.removeprefix('/app/fnterminal/') in files,'missing linked asset '+url)
        check(not re.search(r'(?:src|href)="https?://',html),'external entry asset')
        check(any(n.startswith('web/assets/') and n.endswith('.woff2') for n in files),'missing offline font')
        binary=payload.extractfile(files['bin/fnterminal']).read()
        check(binary[:4]==b'\x7fELF' and binary[4:6]==b'\x02\x01','not Linux ELF64 little endian')
        check(struct.unpack_from('<H',binary,18)[0]=={'x86':62,'arm':183}[arch], 'ELF architecture mismatch')
        # Program header type PT_INTERP would require an external dynamic loader.
        phoff=struct.unpack_from('<Q',binary,32)[0];phsize,phnum=struct.unpack_from('<HH',binary,54)
        check(all(struct.unpack_from('<I',binary,phoff+i*phsize)[0]!=3 for i in range(phnum)), 'binary is not static')
        for file in ['bin/fnterminal','bin/nas-login']: check(files[file].mode&0o111,'non-executable '+file)
        ui=json.loads(payload.extractfile(files['ui/config']).read())['.url']['fnterminal.main']
        check(ui['title']==manifest.get('display_name') and re.search(r'<title>(.*?)</title>',html).group(1)==ui['title'],'application/window/page title mismatch')
        check(ui['gatewayPrefix']=='/app/fnterminal' and ui['gatewaySocket']=='app.sock' and ui['allUsers'] is False,'gateway/visibility configuration')
        for size,name in [(64,'ICON.PNG'),(256,'ICON_256.PNG')]:
            check(payload.extractfile(files[f'ui/images/icon_{size}.png']).read()==read(name),'entry icon differs from package icon')
    report={'file':path.name,'architecture':arch,'version':manifest['version'],'size':path.stat().st_size,'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'app_md5':manifest['checksum'],'checks':'passed','fnos_installation':'pending'}
    path.with_suffix('.verification.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps(report,ensure_ascii=False))
