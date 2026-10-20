#!/usr/bin/env python3
"""Public synthetic RU development fixture; no owner input. Requires reportlab + Noto Sans."""
from pathlib import Path
import argparse
from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont

parser = argparse.ArgumentParser()
parser.add_argument('--font', default='/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf')
args = parser.parse_args()
pdfmetrics.registerFont(TTFont('FixtureSans', args.font))
out = Path(__file__).resolve().parents[2] / 'internal/pdfextract/testdata/turnover-roles-ru.pdf'
c = canvas.Canvas(str(out), pagesize=(600, 800), invariant=1, pageCompression=0)
c.setTitle('Synthetic RU turnover roles regression')
def line(x, y, text):
    c.setFont('FixtureSans', 10)
    c.drawString(x, y, text)
line(30, 760, 'СИНТЕТИЧЕСКАЯ ВЫПИСКА - НЕ РЕАЛЬНЫЙ БАНК')
line(30, 730, 'Доступный остаток RUB 0,00')
line(30, 690, 'Дата операции')
line(210, 690, 'Сумма RUB')
line(30, 665, '12.10.2026')
line(210, 665, '-2486,00')
line(30, 640, 'ТЕСТОВЫЙ МАГАЗИН - ПОКУПКА')
line(30, 40, 'Страница 1')
c.showPage()
line(30, 760, 'СИНТЕТИЧЕСКАЯ ВЫПИСКА - ПРОДОЛЖЕНИЕ')
line(30, 725, 'Дата операции')
line(210, 725, 'Сумма RUB')
line(30, 700, '13.10.2026')
line(210, 700, '+3105,00')
line(30, 675, 'ТЕСТОВЫЙ ПЕРЕВОД - ПРОВЕРИТЬ СМЫСЛ')
line(30, 620, 'Расходы: RUB 2 486,00')
line(30, 595, 'Пополнения: RUB 3 105,00')
line(30, 120, 'БИК 044000999')
line(30, 95, 'КПП 550001111')
line(30, 40, 'Страница 2')
c.save()
print(out)
