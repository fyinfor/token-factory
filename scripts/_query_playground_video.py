import sqlite3
import json

db = r"c:\work\token-factory\one-api.db"
conn = sqlite3.connect(db)
conn.row_factory = sqlite3.Row
cur = conn.cursor()

cur.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
tables = [r[0] for r in cur.fetchall()]
print("tables:", [t for t in tables if any(x in t.lower() for x in ["task", "log", "option", "channel"])])

for sql in [
    "SELECT id, user_id, channel_id, model_name, action, status, progress, submit_time, data FROM tasks ORDER BY id DESC LIMIT 10",
    "SELECT id, user_id, model_name, type, quota, content, other FROM logs ORDER BY id DESC LIMIT 15",
]:
    print("\n===", sql[:70], "===")
    try:
        cur.execute(sql)
        for row in cur.fetchall():
            d = dict(row)
            for k in ("other", "data"):
                if d.get(k):
                    try:
                        d[k] = json.loads(d[k])
                    except Exception:
                        pass
            print(json.dumps(d, ensure_ascii=False, default=str)[:1500])
    except Exception as e:
        print("ERR", e)

# video pricing options snippet
for key in ["VideoPricingRules", "ChannelVideoPricingRules"]:
    cur.execute("SELECT key, substr(value,1,800) FROM options WHERE key LIKE ?", (f"%{key}%",))
    rows = cur.fetchall()
    if rows:
        print(f"\n=== option {key} ===")
        for r in rows:
            print(r[0], ":", r[1][:800])

conn.close()
