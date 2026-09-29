#!/usr/bin/env python3
"""Lists the addon's translatable strings: T("...") calls plus preset and group names."""
import re, os, json
ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "addon", "Parley")
keys = []
def add(k):
    if k not in keys:
        keys.append(k)
for f in ("Parley.lua", "ParleyUI.lua"):
    s = open(os.path.join(ROOT, f), encoding="utf-8").read()
    for m in re.finditer(r'\bT\("((?:[^"\\]|\\.)*)"\)', s):
        add(m.group(1).replace('\\"', '"'))
    for m in re.finditer(r'name = "([^"]+)", desc = "([^"]+)"', s):
        add(m.group(1)); add(m.group(2))
    for m in re.finditer(r'label = "([^"]+)"', s):
        add(m.group(1))
if __name__ == "__main__":
    print(json.dumps(keys, ensure_ascii=False, indent=1))
