#!/usr/bin/env python3
"""Disposable Linux VM preparation. Never follows guest absolute symlinks on the host."""
import gzip,hashlib,json,shutil,struct,tarfile,urllib.request
from pathlib import Path
base=Path('.test-linux');downloads=base/'downloads';downloads.mkdir(parents=True,exist_ok=True)
root=(base/'rootfs').resolve();root.mkdir(parents=True,exist_ok=True)
def download(url,name):
    path=downloads/name
    if not path.exists():urllib.request.urlretrieve(url,path)
    return path
root_archive=download('https://dl-cdn.alpinelinux.org/alpine/v3.22/releases/aarch64/alpine-minirootfs-3.22.6-aarch64.tar.gz','rootfs.tgz')
# A fresh rootfs avoids absolute busybox links from prior package extraction.
if (root/'bin').exists():shutil.rmtree(root);root.mkdir()
with tarfile.open(root_archive) as archive:archive.extractall(root,filter='fully_trusted')
index_archive=download('https://dl-cdn.alpinelinux.org/alpine/v3.22/main/aarch64/APKINDEX.tar.gz','APKINDEX.tar.gz')
with tarfile.open(index_archive) as archive:content=archive.extractfile('APKINDEX').read().decode()
index={}
for block in content.split('\n\n'):
    package=dict(line.split(':',1) for line in block.splitlines() if ':' in line)
    if 'P' in package:index[package['P']]=package
names=['bash','readline','libncursesw','ncurses-terminfo-base','flock','openssh-client-common','openssh-client-default','openssh-keygen','openssh-server','openssh-server-common','libcrypto3','zlib','libedit','runuser','linux-pam','libeconf','libintl','utmps-libs','skalibs-libs']
for name in names:
    package=index[name];filename=name+'-'+package['V']+'.apk'
    path=download('https://dl-cdn.alpinelinux.org/alpine/v3.22/main/aarch64/'+filename,filename)
    with tarfile.open(path,'r:gz',ignore_zeros=True) as archive:
        for member in archive.getmembers():
            if member.name.startswith('.') or not member.name:continue
            target=root/member.name
            if '..' in Path(member.name).parts or member.name.startswith('/') or not target.parent.resolve().is_relative_to(root):raise RuntimeError('archive path escapes test rootfs')
            if target.is_symlink():target.unlink()
            archive.extract(member,root,filter='fully_trusted')
kernel=download('https://dl-cdn.alpinelinux.org/alpine/v3.22/releases/aarch64/netboot-3.22.6/vmlinuz-virt','vmlinuz')
data=kernel.read_bytes()
if data[4:8]==b'zimg':
    offset,size=struct.unpack_from('<II',data,8);data=gzip.decompress(data[offset:offset+size])
assert data[56:60]==b'ARMd','invalid ARM64 Linux kernel'
(base/'kernel').write_bytes(data)
for source,destination in [('scripts/linux/init-test.sh','init'),('scripts/linux/lifecycle.sh','lifecycle.sh'),('packaging/cmd/main','package-main'),('packaging/app/bin/nas-login','nas-login')]:
    shutil.copyfile(source,root/destination);(root/destination).chmod(0o755)
(base/'downloads.json').write_text(json.dumps({p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in downloads.iterdir() if p.is_file()},indent=2)+'\n')
