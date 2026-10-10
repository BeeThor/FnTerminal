#!/usr/bin/env python3
"""Create a source delivery archive without caches, binaries or fixture logs."""
import hashlib
import os
from pathlib import Path
import zipfile

root = Path(__file__).resolve().parent.parent
version = next(line.split('=', 1)[1] for line in (root / 'packaging/manifest').read_text().splitlines() if line.startswith('version='))
output = root / 'dist' / f'FnTerminal-{version}-source.zip'
output.parent.mkdir(exist_ok=True)
excluded = {'.git', '.agents', '.codex', '.cache', '.tools', '.upstream', '.build', '.test-linux', '.vscode', '.idea', 'node_modules', 'dist', 'test-results', '__pycache__'}
with zipfile.ZipFile(output, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
    for directory, folders, names in os.walk(root):
        folders[:] = sorted(folder for folder in folders if folder not in excluded)
        for name in sorted(names):
            path = Path(directory) / name
            relative = path.relative_to(root)
            if path.is_symlink() or name == '.DS_Store' or path.suffix in {'.pyc', '.bak', '.pem', '.key'}:
                continue
            if (name == '.env' or name.startswith('.env.')) and name != '.env.example':
                continue
            if relative.parts[:3] == ('packaging', 'app', 'web') or relative == Path('packaging/app/bin/fnterminal'):
                continue
            archive.write(path, Path(f'FnTerminal-{version}') / relative)
with zipfile.ZipFile(output) as archive:
    if archive.testzip():
        raise SystemExit('Source archive verification failed')
checksum = hashlib.sha256(output.read_bytes()).hexdigest()
(output.parent / 'SOURCE_SHA256SUMS').write_text(f'{checksum}  {output.name}\n')
print(f'{output.name}: {output.stat().st_size} bytes, sha256={checksum}')
