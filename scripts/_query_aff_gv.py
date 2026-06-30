import json
import sqlite3

con = sqlite3.connect(r"C:\work\token-factory\one-api.db")
con.row_factory = sqlite3.Row

print("=== users test / test1 ===")
for u in con.execute(
    "SELECT id, username, role, aff_quota, inviter_id, is_distributor, distributor_commission_bps FROM users WHERE username IN ('test','test1')"
):
    print(dict(u))

print("\n=== aff_invite_relations ===")
for r in con.execute(
    "SELECT * FROM aff_invite_relations WHERE inviter_id IN (SELECT id FROM users WHERE username='test') OR invitee_user_id IN (SELECT id FROM users WHERE username='test1')"
):
    d = dict(r)
    print("id", d.get("id"), "inviter", d.get("inviter_id"), "invitee", d.get("invitee_user_id"))
    raw = d.get("model_markup_discount_rate") or "[]"
    if raw and raw != "[]":
        print("  model_markup_discount_rate:", raw[:2000])

print("\n=== aff_invite_profit_share_logs (recent) ===")
for r in con.execute(
    """SELECT l.*, u.username as invitee_name FROM aff_invite_profit_share_logs l
       LEFT JOIN users u ON u.id=l.invitee_user_id
       ORDER BY l.id DESC LIMIT 15"""
):
    print(dict(r))

print("\n=== recent consume logs test1 ===")
for r in con.execute(
    """SELECT id, user_id, type, quota, model_name, content, other, created_at FROM logs
       WHERE user_id=(SELECT id FROM users WHERE username='test1')
       ORDER BY id DESC LIMIT 8"""
):
    d = dict(r)
    other = d.get("other")
    if other:
        try:
            o = json.loads(other)
            d["billing_mode"] = o.get("billing_mode")
            d["markup_discount_rate"] = o.get("markup_discount_rate")
            d["actual_quota"] = o.get("actual_quota")
            d["video_billed_quota"] = o.get("video_billed_quota")
        except Exception:
            pass
    del d["other"]
    print(d)

print("\n=== distributor commission mode ===")
for r in con.execute("SELECT key, value FROM options WHERE key LIKE '%distributor%' OR key LIKE '%commission%' OR key LIKE '%profit%'"):
    print(r["key"], r["value"][:200] if r["value"] else "")
