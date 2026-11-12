#!/usr/bin/env python3
"""tokens.json -> tokens.css. Run from this folder after any token change: python3 gen_css.py"""
import json

t = json.load(open("tokens.json"))
c = t["color"]

def block(sel, d, scheme=None, indent=""):
    lines = ["  --%s: %s;" % (k, v) for k, v in d.items()] + (["  color-scheme: %s;" % scheme] if scheme else [])
    return "".join(indent + l + "\n" for l in ["%s {" % sel] + lines + ["}"])

base = {}
base.update({"font-" + k: v for k, v in t["font"].items()})
for k, v in t["type"].items():
    base["text-%s" % k] = "%dpx/%dpx" % (v["size"], v["line"])
    base["weight-%s" % k] = v["weight"]
base.update({"space-%d" % i: "%dpx" % v for i, v in enumerate(t["space"])})
base.update({"radius-" + k: "%dpx" % v for k, v in t["radius"].items()})
base.update({"shadow-" + k: v for k, v in t["shadow"].items()})
base.update({"motion-" + k: v for k, v in t["motion"].items()})
base.update({k: "%dpx" % v for k, v in t["layout"].items() if not k.startswith("bp-")})
base.update({"viewer-" + k: v for k, v in c["viewer"].items()})

css = "/* Generated from tokens.json by gen_css.py. Do not edit by hand. */\n"
css += "/* Breakpoints (CSS vars can't be used in media queries): md %(bp-md)dpx, lg %(bp-lg)dpx */\n\n" % t["layout"]
css += block(":root", {**base, **c["light"]}, "light")
css += "\n@media (prefers-color-scheme: dark) {\n" + block(':root:not([data-theme="light"])', c["dark"], "dark", "  ") + "}\n"
css += "\n" + block(':root[data-theme="dark"]', c["dark"], "dark")
open("tokens.css", "w").write(css)

# self-check: every light colour role is redefined for dark, and the file uses every role
assert set(c["light"]) == set(c["dark"]), set(c["light"]) ^ set(c["dark"])
assert all("--%s:" % k in css for k in c["light"])
assert "--color-scheme" not in css and css.count("color-scheme: dark;") == 2
print("tokens.css: %d lines" % css.count("\n"))
