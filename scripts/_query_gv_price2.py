import json
import sqlite3

con = sqlite3.connect(r"C:\work\token-factory\one-api.db")
for key in ("ModelPrice", "VideoPrice"):
    row = con.execute("SELECT value FROM options WHERE key=?", (key,)).fetchone()
    if row:
        d = json.loads(row[0])
        print(key, "GV-3.1-fast =", d.get("GV-3.1-fast"))
