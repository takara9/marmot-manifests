#!/usr/bin/env python3
"""README.md の構成図 (mail-server-architecture.png) を生成するスクリプト。

再作成の方法:
    sudo apt install python3-pil fonts-noto-cjk     # または pip install pillow
    python3 82-mail-server/image/draw_architecture.py

同じディレクトリに mail-server-architecture.png を上書き出力します。
図を変更するときは、このファイルのボックス座標・矢印・ラベルを編集して再実行してください。
座標は 1500x880 の論理ピクセルで、S 倍（既定 2 倍）の解像度で描画します。
フォントは Noto Sans CJK（日本語）を使います。別の場所にある場合は FP / FB を変更してください。
"""
import math
import os

from PIL import Image, ImageDraw, ImageFont

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "mail-server-architecture.png")
FP = "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"
FB = "/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc"

S = 2
W, H = 1500, 880
img = Image.new("RGB", (W * S, H * S), "white")
d = ImageDraw.Draw(img)


def font(path, size):
    # .ttc には SC / TC / JP などが含まれるため、日本語 (JP) の字形を選ぶ
    for i in range(8):
        try:
            f = ImageFont.truetype(path, size * S, index=i)
            if "JP" in "".join(f.getname()):
                return f
        except Exception:
            break
    return ImageFont.truetype(path, size * S)


f_title = font(FB, 20)
f_sub = font(FP, 14)
f_lab = font(FP, 14)
f_frame = font(FB, 18)

BLUE, ORANGE, GREEN, GRAY = (31, 90, 180), (214, 105, 10), (30, 140, 70), (90, 90, 90)


def P(x, y):
    return (x * S, y * S)


def box(x1, y1, x2, y2, title, sub=None, fill=(235, 242, 252), outline=(60, 90, 150)):
    d.rounded_rectangle([P(x1, y1), P(x2, y2)], radius=10 * S, fill=fill, outline=outline, width=2 * S)
    cx = (x1 + x2) / 2
    lines = [(title, f_title, (20, 20, 20))] + [(t, f_sub, (70, 70, 70)) for t in (sub or [])]
    hs = [d.textbbox((0, 0), t, font=f)[3] / S + 6 for t, f, _ in lines]
    y = (y1 + y2) / 2 - sum(hs) / 2
    for (t, f, c), h in zip(lines, hs):
        w = d.textlength(t, font=f) / S
        d.text(P(cx - w / 2, y), t, font=f, fill=c)
        y += h


def arrowhead(p, q, color):
    a = math.atan2(q[1] - p[1], q[0] - p[0])
    L, B = 12, 6
    pts = [q,
           (q[0] - L * math.cos(a) + B * math.sin(a), q[1] - L * math.sin(a) - B * math.cos(a)),
           (q[0] - L * math.cos(a) - B * math.sin(a), q[1] - L * math.sin(a) + B * math.cos(a))]
    d.polygon([P(*t) for t in pts], fill=color)


def line(pts, color, both=False, width=3):
    d.line([P(*p) for p in pts], fill=color, width=width * S, joint="curve")
    arrowhead(pts[-2], pts[-1], color)
    if both:
        arrowhead(pts[1], pts[0], color)


def label(x, y, text, color, anchor="c"):
    # 線に重なっても読めるよう、白い背景を敷く
    w = d.textlength(text, font=f_lab) / S
    h = d.textbbox((0, 0), text, font=f_lab)[3] / S
    x0 = x - w / 2 if anchor == "c" else x
    d.rectangle([P(x0 - 4, y - 2), P(x0 + w + 4, y + h + 4)], fill="white")
    d.text(P(x0, y), text, font=f_lab, fill=color)


# メールサーバーの枠
d.rounded_rectangle([P(330, 30), P(1150, 850)], radius=14 * S, fill=(250, 250, 250), outline=(120, 120, 120), width=2 * S)
d.text(P(350, 40), "メールサーバー mail.labo.local (192.168.1.12)", font=f_frame, fill=(60, 60, 60))

# 枠の外
box(30, 40, 270, 120, "宛先の外部", ["メールサーバー"], fill=(240, 240, 240), outline=(110, 110, 110))
box(30, 160, 270, 250, "Brevo", ["smtp-relay.brevo.com"], fill=(255, 244, 230), outline=(190, 120, 40))
box(30, 310, 270, 410, "外部・他のサーバー", ["（インターネット / labo.local）"], fill=(240, 240, 240), outline=(110, 110, 110))
box(30, 520, 270, 640, "メールクライアント", ["（Thunderbird など）"], fill=(240, 240, 240), outline=(110, 110, 110))
box(1220, 60, 1460, 160, "ブラウザ", None, fill=(240, 240, 240), outline=(110, 110, 110))

# 枠の中
box(800, 60, 1100, 160, "Apache + PHP", ["Roundcube（Web メール）"], fill=(230, 245, 235), outline=(50, 130, 80))
box(380, 300, 640, 420, "Postfix", ["MTA（SMTP）", "内部アドレス → marmot.mydns.jp"], fill=(255, 244, 230), outline=(190, 120, 40))
box(800, 300, 1100, 420, "Dovecot", ["IMAP / LMTP / SASL 認証"])
box(800, 720, 1100, 810, "Maildir", ["/var/mail/vhosts（vmail）"], fill=(245, 245, 245), outline=(110, 110, 110))

# ブラウザ <-> Roundcube
line([(1220, 110), (1100, 110)], GREEN, both=True)
label(1160, 80, "443 HTTPS", GREEN)

# Roundcube <-> Dovecot
line([(950, 160), (950, 300)], BLUE, both=True)
label(950, 222, "993 IMAPS", BLUE)

# Roundcube -> Postfix
line([(800, 110), (510, 110), (510, 300)], ORANGE)
label(655, 82, "587 SMTP（認証 + STARTTLS）", ORANGE)

# Postfix <-> Dovecot
line([(640, 340), (800, 340)], BLUE, both=True)
label(720, 308, "SASL 認証", BLUE)
line([(640, 390), (800, 390)], BLUE)
label(720, 398, "LMTP", BLUE)

# Dovecot <-> Maildir
line([(1000, 420), (1000, 720)], BLUE, both=True)
label(1000, 560, "保存 / 読み出し", BLUE)

# 外部・他のサーバー -> Postfix
line([(270, 360), (380, 360)], ORANGE)
label(325, 328, "25", ORANGE)
d.text(P(40, 420), "MX レコードで mail.labo.local を参照", font=f_sub, fill=GRAY)

# Postfix -> Brevo -> 宛先の外部メールサーバー
line([(430, 300), (430, 205), (270, 205)], ORANGE)
label(350, 172, "587 SMTP（認証 + TLS）", ORANGE)
line([(150, 160), (150, 120)], ORANGE)
label(190, 126, "配送", ORANGE)

# メールクライアント -> Postfix
line([(270, 560), (510, 560), (510, 420)], ORANGE)
label(385, 530, "587 / 465 SMTP（認証）", ORANGE)

# メールクライアント <-> Dovecot
line([(270, 610), (850, 610), (850, 420)], BLUE, both=True)
label(560, 578, "993 IMAPS / 143 IMAP（STARTTLS）", BLUE)

# 凡例
d.rounded_rectangle([P(30, 700), P(290, 850)], radius=8 * S, outline=(160, 160, 160), width=S, fill="white")
d.text(P(42, 708), "凡例", font=f_lab, fill=(60, 60, 60))
for i, (c, t) in enumerate([(ORANGE, "SMTP（送信・配送）"), (BLUE, "IMAP / LMTP / SASL"), (GREEN, "HTTPS")]):
    y = 750 + i * 32
    d.line([P(42, y), P(92, y)], fill=c, width=3 * S)
    arrowhead((42, y), (92, y), c)
    d.text(P(104, y - 10), t, font=f_lab, fill=(40, 40, 40))

img.save(OUT, optimize=True)
print(f"wrote {OUT}")
