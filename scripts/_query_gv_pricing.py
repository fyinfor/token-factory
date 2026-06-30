import json
import sqlite3

con = sqlite3.connect(r"C:\work\token-factory\one-api.db")
con.row_factory = sqlite3.Row

print("=== channels with GV-3.1-fast in models ===")
rows = con.execute(
    "SELECT id, name, type, models, price_discount_percent, markup_discount_rate FROM channels WHERE models LIKE ?",
    ("%GV-3.1-fast%",),
).fetchall()
for r in rows:
    print(dict(r))

print("\n=== options keys related video/image/GV ===")
opts = con.execute(
    "SELECT key, length(value) as len FROM options WHERE key LIKE '%video%' OR key LIKE '%image%' OR key LIKE '%GV%' OR key LIKE '%Video%' OR key LIKE '%Image%'"
).fetchall()
for o in opts:
    print(o["key"], o["len"])

for key_pat in ["VideoPricingRules", "video_pricing", "ImagePricing", "ModelPrice", "Ratio"]:
    print(f"\n--- options like %{key_pat}% ---")
    for row in con.execute(
        "SELECT key, value FROM options WHERE key LIKE ? LIMIT 20",
        (f"%{key_pat}%",),
    ):
        val = row["value"]
        if "GV" in val or "gv" in val.lower():
            print("KEY:", row["key"])
            try:
                data = json.loads(val)
                if isinstance(data, dict):
                    for mk, mv in data.items():
                        if "GV" in mk or "gv" in mk.lower():
                            print("  model:", mk)
                            print("  snippet:", json.dumps(mv, ensure_ascii=False)[:800])
                else:
                    print("  type", type(data), str(data)[:400])
            except Exception as e:
                print("  parse err", e, val[:200])

print("\n=== supplier channel video rules (channel id=1) ===")
for row in con.execute("SELECT key, value FROM options WHERE key LIKE '%ChannelVideo%' OR key LIKE '%channel_video%' LIMIT 30"):
    val = row["value"]
    if "GV" not in val:
        continue
    print("KEY:", row["key"])
    try:
        data = json.loads(val)
        if isinstance(data, dict):
            for k, v in data.items():
                if "1" in k and ("GV" in str(v) or True):
                    pass
            # try nested channel id 1
            ch = data.get("1") or data.get(1)
            if ch and isinstance(ch, dict):
                for mk, rules in ch.items():
                    if "GV" in mk:
                        print("  channel model:", mk)
                        print(json.dumps(rules, ensure_ascii=False)[:1200])
        print("  top keys sample:", list(data.keys())[:5] if isinstance(data, dict) else type(data))
    except Exception as e:
        print(e)
