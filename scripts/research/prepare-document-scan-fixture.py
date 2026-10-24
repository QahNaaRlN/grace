#!/usr/bin/env python3
"""Reproducible public scan: lossless images of S1.22 synthetic full32 pages."""
from pathlib import Path
from reportlab.pdfgen import canvas
from reportlab.lib.utils import ImageReader
import hashlib
import json

ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / 'internal/statementextract/testdata/direct-page-reading/full32'
OUTPUT = ROOT / 'internal/pdfextract/testdata/full32-scan.pdf'

def main():
    pages = [SOURCE / f'page-{i}.png' for i in (1, 2, 3)]
    pdf = canvas.Canvas(str(OUTPUT), invariant=1, pageCompression=1)
    pdf.setTitle('Synthetic full32 image-only statement fixture')
    for path in pages:
        image = ImageReader(str(path))
        width, height = image.getSize()
        pdf.setPageSize((width, height))
        pdf.drawImage(image, 0, 0, width=width, height=height)
        pdf.showPage()
    pdf.save()
    manifest = {'source': 'public synthetic S1.22 full32', 'pages': 3, 'rows': 32,
                'inputSHA256': [hashlib.sha256(p.read_bytes()).hexdigest() for p in pages],
                'pdfSHA256': hashlib.sha256(OUTPUT.read_bytes()).hexdigest()}
    OUTPUT.with_suffix('.json').write_text(json.dumps(manifest, indent=2) + '\n')

if __name__ == '__main__':
    main()
