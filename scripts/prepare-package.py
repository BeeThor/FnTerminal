#!/usr/bin/env python3
"""Prepare root logo sizes with Pillow. No generated/replaced artwork."""
import sys
from pathlib import Path
from PIL import Image
stage, arch = Path(sys.argv[1]), sys.argv[2]
p = stage / 'manifest'
p.write_text('\n'.join('platform='+arch if line.startswith('platform=') else line for line in p.read_text().splitlines())+'\n')
with Image.open('logo.png') as source:
    if source.width != source.height:
        raise SystemExit('logo.png must be square')
    for size, name in [(64, 'ICON.PNG'), (256, 'ICON_256.PNG')]:
        resized = source.convert('RGBA').resize((size, size), Image.Resampling.LANCZOS)
        resized.save(stage / name)
        resized.save(stage / 'app/ui/images' / f'icon_{size}.png')

for path in stage.rglob(".DS_Store"):
    path.unlink()
