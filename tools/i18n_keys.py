#!/usr/bin/env python3
"""Lists every interface string Parley translates (T/Tf literals and a few tables)."""
import re, glob, json, os
ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
keys = []
def add(k):
    k = bytes(k, "utf-8").decode("unicode_escape").encode("latin-1").decode("utf-8")
    if k not in keys:
        keys.append(k)
for p in sorted(glob.glob(os.path.join(ROOT, "*.go"))):
    if p.endswith("_test.go") or os.path.basename(p).startswith("i18n_data"):
        continue
    s = open(p, encoding="utf-8").read()
    for m in re.finditer(r'\bTf?\("((?:[^"\\]|\\.)*)"', s):
        add(m.group(1))
    if p.endswith("i18n.go"):
        m = re.search(r'backendMessages = \[\]string\{(.*?)\n\}', s, re.S)
        for v in re.findall(r'"((?:[^"\\]|\\.)*)"', m.group(1)):
            add(v)
        for m in re.finditer(r'MustCompile\(`[^`]*`\), "((?:[^"\\]|\\.)*)"', s):
            add(m.group(1))
    if p.endswith("skins_windows.go"):
        for m in re.finditer(r'Label: "([^"]+)"', s):
            add(m.group(1))
    if p.endswith("overlay_windows.go"):
        m = re.search(r'chanNames\s*=\s*map\[string\]string\{(.*?)\}', s, re.S)
        for v in re.findall(r':\s*"([^"]+)"', m.group(1)):
            add(v)
    if p.endswith("settings_windows.go"):
        m = re.search(r'engineLabels\s*=\s*\[\]string\{(.*?)\}', s, re.S)
        for v in re.findall(r'"([^"]+)"', m.group(1)):
            add(v)
    if p.endswith("phrasebook.go"):
        for v in re.findall(r'^\t\{"([^"]+)", map', s, re.M):
            add(v)
if __name__ == "__main__":
    print(json.dumps(keys, ensure_ascii=False, indent=1))
