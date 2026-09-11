import re

with open("mime.src", "r", encoding="utf-8") as f:
    src = f.read()

f.close()
td = re.findall("<td>.*</td>", src)

for i in range(0, len(td), 1):
    if (i + 1) % 3 == 0:
        code = re.findall("<code>.*</code>", src)
        s = code[0]
        s.replace("</code>", "")
        s.replace("<code>", "")
        print("case", s + ":")