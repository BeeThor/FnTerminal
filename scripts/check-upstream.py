#!/usr/bin/env python3
import hashlib,json
from pathlib import Path
meta=json.loads(Path('third_party/uniterm/UPSTREAM.json').read_text())
for file in meta['files']:
    path=Path('third_party/uniterm/original')/file['path'].removeprefix('frontend/src/')
    assert hashlib.sha256(path.read_bytes()).hexdigest()==file['sha256'],f'Original changed: {path}'
modified=[]
for file in meta['files']:
    p=Path(file['path'])
    if p.exists() and hashlib.sha256(p.read_bytes()).hexdigest()!=file['sha256']:
        assert 'FnTerminal' in p.read_text().splitlines()[0],f'Missing modification notice: {p}'
        modified.append(str(p))
Path('third_party/uniterm/MODIFICATIONS.json').write_text(json.dumps({'commit':meta['commit'],'modified_files':modified},indent=2)+'\n')
print(f"Uniterm provenance verified: {len(meta['files'])} originals, {len(modified)} adapted files")
