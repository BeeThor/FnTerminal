import os,stat,struct,gzip
from pathlib import Path
root=Path('.test-linux/rootfs')
output=bytearray()
def entry(name,mode,data,ino):
 fields=[ino,mode,0,0,1,0,len(data),0,0,0,0,len(name.encode())+1,0]
 output.extend(b'070701'+b''.join(f'{x:08x}'.encode() for x in fields));output.extend(name.encode()+b'\0');output.extend(b'\0'*((-len(output))%4));output.extend(data);output.extend(b'\0'*((-len(output))%4))
for i,p in enumerate(sorted(root.rglob('*')),1):
 s=p.lstat();name=str(p.relative_to(root))
 if p.is_symlink():data=os.readlink(p).encode()
 elif p.is_file():data=p.read_bytes()
 elif p.is_dir():data=b''
 else:continue
 entry(name,s.st_mode,data,i)
entry('TRAILER!!!',0,b'',999999)
Path('.test-linux/initramfs.cpio.gz').write_bytes(gzip.compress(output,compresslevel=3))
print('Initramfs:',len(output),'bytes')
