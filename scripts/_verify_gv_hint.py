"""模拟 BuildVideoFlatClipHint 对 GV-3.1-fast 渠道1 的最低价（文生视频按秒 854x480）。"""
import json
import sqlite3

con = sqlite3.connect(r"C:\work\token-factory\one-api.db")
ch_rules = json.loads(
    con.execute("SELECT value FROM options WHERE key='ChannelVideoPricingRules'").fetchone()[0]
)["1"]["GV-3.1-fast"]
gl_rules = json.loads(
    con.execute("SELECT value FROM options WHERE key='VideoPricingRules'").fetchone()[0]
)["GV-3.1-fast"]
ch = 1.0
gl = 2.0
for row in ch_rules["text_to_video_per_second"]:
    if row["resolution"] == "854x480" and row["has_audio"] is False:
        ch = row["price"]
for row in gl_rules["text_to_video_per_second"]:
    if row["resolution"] == "854x480" and row["has_audio"] is False:
        gl = row["price"]
markup = con.execute(
    "SELECT markup_discount_rate FROM channels WHERE id=1"
).fetchone()[0]
eff = ch * 1.0 + gl * (markup / 100.0)
print("channel 854x480 $/s", ch)
print("global 854x480 $/s", gl)
print("markup_discount_rate", markup, "%")
print("display min $/s (cost 100% + markup)", eff)
print("before fix showed only", ch, "$/s (cost only)")
