import sqlite3
import json

c = sqlite3.connect(r"c:\work\token-factory\one-api.db")
v = json.loads(c.execute("SELECT value FROM options WHERE key='VideoPricingRules'").fetchone()[0])
c.close()

for m in [
    "happyhorse-1.0-i2v",
    "happyhorse-1.0-r2v",
    "happyhorse-1.0-t2v",
    "Seedance2.0",
    "GV-3.1-fast",
]:
    r = v.get(m, {})
    lanes = []
    for k, val in r.items():
        if not isinstance(val, list) or not val:
            continue
        if any(isinstance(x, dict) and (x.get("price") or x.get("video_price") or 0) > 0 for x in val):
            lanes.append(k)
    print(m, lanes or "NONE")
