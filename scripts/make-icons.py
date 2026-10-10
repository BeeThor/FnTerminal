#!/usr/bin/env python3
from pathlib import Path
from PIL import Image
# Keep the browser header icon tied to the same user-supplied root logo.
with Image.open('logo.png') as source:
    if source.width != source.height:
        raise SystemExit('logo.png must be square')
    path=Path('frontend/src/assets/app-icon.png')
    path.parent.mkdir(parents=True,exist_ok=True)
    source.convert('RGBA').resize((64,64),Image.Resampling.LANCZOS).save(path)
