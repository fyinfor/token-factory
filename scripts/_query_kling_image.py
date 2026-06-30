import json
import sqlite3

con = sqlite3.connect(r"C:\work\token-factory\one-api.db")
con.row_factory = sqlite3.Row

print("=== aff relation test1 markup ===")
for r in con.execute(
    "SELECT model_markup_discount_rate FROM aff_invite_relations WHERE invitee_user_id=3"
):
    print(r[0])

print("\n=== channel 3 ===")
for r in con.execute(
    "SELECT id, name, models, markup_discount_rate, price_discount_percent FROM channels WHERE id=3"
):
    print(dict(r))

print("\n=== Kling image pricing options ===")
for key in ("ChannelImagePricingRules", "ImagePricingRules", "ChannelImagePrice", "ImagePrice", "ModelPrice"):
    row = con.execute("SELECT value FROM options WHERE key=?", (key,)).fetchone()
    if not row:
        continue
    d = json.loads(row[0])
    for mk, mv in (d.items() if isinstance(d, dict) else []):
        if "Kling" in mk or "kling" in mk.lower():
            print("KEY", key, "model", mk)
            print(json.dumps(mv, ensure_ascii=False)[:1500])

print("\n=== profit share logs Kling ===")
for r in con.execute(
    """SELECT * FROM aff_invite_profit_share_logs
       WHERE invitee_user_id=3 AND model_name LIKE '%Kling%' ORDER BY id DESC LIMIT 5"""
):
    print(dict(r))

print("\n=== logs Kling test1 ===")
for r in con.execute(
    """SELECT id, quota, model_name, channel, other FROM logs
       WHERE user_id=3 AND model_name LIKE '%Kling%' ORDER BY id DESC LIMIT 3"""
):
    d = dict(r)
    o = json.loads(d.pop("other") or "{}")
    d["billing_mode"] = o.get("billing_mode")
    d["image_usd_per_image"] = o.get("image_usd_per_image")
    d["markup_discount_rate"] = o.get("markup_discount_rate")
    d["global_model_price"] = o.get("global_model_price")
    print(d)
