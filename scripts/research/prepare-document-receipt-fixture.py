#!/usr/bin/env python3
"""Public image-only receipt; invariant ReportLab output, no source text layer."""
from pathlib import Path
from reportlab.pdfgen import canvas
from reportlab.lib.utils import ImageReader
ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / 'internal/documentqualification/testdata/receipt-02.pdf'
image = ImageReader(str(ROOT / 'testdata/ai-evaluation/images/receipt-02.png'))
width, height = image.getSize()
pdf = canvas.Canvas(str(OUTPUT), pagesize=(width, height), invariant=1, pageCompression=1)
pdf.setTitle('Public synthetic receipt-02, image-only')
pdf.drawImage(image, 0, 0, width=width, height=height)
pdf.showPage()
pdf.save()
