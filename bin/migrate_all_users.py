#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""全量迁移用户及其令牌(tokens)到另一个库。

适用场景：把源库的「全部用户」连同其 API 令牌整体迁到目标库。

迁移内容：
  - users 表全部用户：用户名、密码(哈希原样复制)、角色、状态、邮箱、手机号、分组、
    权限/边栏设置、各 OAuth 绑定、备注、标签、注册时间等（默认连同额度等一并迁移）。
  - tokens 表：每个被迁用户名下的 API 令牌（key、名称、过期时间、额度上限、分组等），
    令牌的 user_id 会重映射到目标库新主键。令牌 remain_quota 是「使用上限」而非余额，照原值迁移。
  - inviter_id：在迁移用户集合内重映射，保持邀请关系。
  - 账户余额默认不迁移：用户 quota（余额）、aff_quota / aff_history（邀请收益）一律置 0。
    令牌额度上限、用量统计等正常迁移。（如确需连余额一起迁，加 --keep-amount）

冲突规则（按需求）：
  - 目标库已存在「相同用户名」-> 跳过该用户（其令牌也不迁），列入「待人工确认」清单。
  - 令牌 key 在目标库已存在 -> 跳过该令牌并记录（极少见，两环境 key 理论不重复）。

唯一约束兜底：
  - access_token：目标库已占用则置空。
  - aff_code：目标库已占用则随机重新生成。

统计输出：用户总数、导入成功数、相同用户名跳过数(=待确认数)、令牌迁移数、令牌跳过数。

默认是 dry-run（只预览不写入）。确认无误后加 --execute 真正写入目标库。

依赖：pip install -r bin/requirements-migrate.txt

示例(PowerShell 写一行)：
  python bin/migrate_all_users.py --source-dsn "postgres://root:KGzKjZpWBp4R4RSa@43.132.223.112:5432/token-factory" 
  --target-dsn "postgres://root:KGzKjZpWBp4R4RSa@43.138.142.105:5432/token-factory" --report migrate_all_report.json
  # 确认后：
  python bin/migrate_all_users.py --source-dsn "postgres://root:KGzKjZpWBp4R4RSa@43.132.223.112:5432/token-factory" --target-dsn "postgres://root:KGzKjZpWBp4R4RSa@43.138.142.105:5432/token-factory" --report migrate_all_report.json --execute
  # 默认不迁账户余额(置 0)；若需连余额一起迁：加 --keep-amount
"""

import argparse
import json
import os
import re
import secrets
import string
import sys
from datetime import datetime
from urllib.parse import quote_plus

try:
    from sqlalchemy import MetaData, Table, create_engine, insert, select, update
except ImportError:
    sys.stderr.write(
        "缺少依赖 SQLAlchemy，请先执行：pip install -r bin/requirements-migrate.txt\n"
    )
    sys.exit(1)

USERS_TABLE = "users"
TOKENS_TABLE = "tokens"

# 默认清零的「账户余额 / 收益」列（属于钱，不迁移）。
# 注意：不含 tokens.remain_quota —— 那是令牌的使用上限，不是余额，需照原值迁移。
USER_ZERO_COLUMNS = {
    "quota",       # 账户余额
    "aff_quota",   # 邀请剩余收益（可提现/可用收益）
    "aff_history", # 邀请历史累计收益
}
# 令牌额度属于「使用限制」而非余额，默认全部按原值迁移，无需清零。
TOKEN_ZERO_COLUMNS = set()

REPORT_FIELDS = ["old_id", "username", "display_name", "email", "phone", "role"]
IN_CHUNK = 500


def log(msg):
    print(msg, flush=True)


def convert_dsn(raw):
    raw = (raw or "").strip()
    if not raw:
        raise ValueError("DSN 为空")
    if re.match(r"^(mysql\+\w+|postgresql(\+\w+)?|sqlite)://", raw):
        return raw
    if raw.startswith("postgres://"):
        return "postgresql://" + raw[len("postgres://") :]
    if raw.startswith("postgresql://"):
        return raw
    m = re.match(
        r"^(?P<user>[^:]+):(?P<pw>[^@]*)@tcp\((?P<host>[^:)]+)(?::(?P<port>\d+))?\)/(?P<db>[^?]+)",
        raw,
    )
    if m:
        user = quote_plus(m.group("user"))
        pw = quote_plus(m.group("pw"))
        host = m.group("host")
        port = m.group("port") or "3306"
        db = m.group("db")
        return f"mysql+pymysql://{user}:{pw}@{host}:{port}/{db}?charset=utf8mb4"
    if raw == "local":
        raw = "one-api.db"
    if raw.endswith(".db") or raw.endswith(".sqlite") or os.path.exists(raw):
        return "sqlite:///" + os.path.abspath(raw)
    raise ValueError(f"无法识别的 DSN 格式：{raw}")


def chunked(items, size=IN_CHUNK):
    items = list(items)
    for i in range(0, len(items), size):
        yield items[i : i + size]


def make_report_row(row):
    item = {}
    for f in REPORT_FIELDS:
        key = "id" if f == "old_id" else f
        item[f] = row.get(key)
    return item


def gen_aff_code():
    alphabet = string.ascii_letters + string.digits
    return "".join(secrets.choice(alphabet) for _ in range(6))


class AllUsersMigrator:
    def __init__(self, src_engine, tgt_engine, include_deleted, zero_amount, execute):
        self.src_engine = src_engine
        self.tgt_engine = tgt_engine
        self.include_deleted = include_deleted
        self.zero_amount = zero_amount
        self.execute = execute

        self.users_src = Table(USERS_TABLE, MetaData(), autoload_with=src_engine)
        self.users_tgt = Table(USERS_TABLE, MetaData(), autoload_with=tgt_engine)
        self.src_cols = set(self.users_src.c.keys())
        self.tgt_cols = list(self.users_tgt.c.keys())

        for required in ("id", "username"):
            if required not in self.src_cols:
                raise RuntimeError(f"源库 users 表缺少必要列：{required}")

        self._tgt_aff_codes = set()
        self._tgt_access_tokens = set()

        self.inserted = []  # [(old_id, new_id)]
        self.old_to_new = {}  # old_id -> 目标库新 id（仅成功插入）
        self.resolved_old_ids = set()  # 已/将插入的源 id（两种模式都填充）
        self.skipped_same_username = []  # 待人工确认清单
        self.tokens_migrated = 0
        self.tokens_skipped = 0
        self.tokens_candidate = 0

    def _has_col(self, name):
        return name in self.src_cols

    # ---------- 目标库唯一值预载 ----------
    def preload_unique(self, conn):
        if "aff_code" in self.tgt_cols:
            for r in conn.execute(select(self.users_tgt.c.aff_code)):
                if r[0]:
                    self._tgt_aff_codes.add(r[0])
        if "access_token" in self.tgt_cols:
            for r in conn.execute(select(self.users_tgt.c.access_token)):
                if r[0]:
                    self._tgt_access_tokens.add(r[0])

    def resolve_aff_code(self, conn, code):
        if "aff_code" not in self.tgt_cols:
            return None
        code = (code or "").strip()
        if code and code not in self._tgt_aff_codes:
            self._tgt_aff_codes.add(code)
            return code
        for _ in range(20):
            new_code = gen_aff_code()
            if new_code not in self._tgt_aff_codes:
                self._tgt_aff_codes.add(new_code)
                return new_code
        fallback = gen_aff_code() + secrets.token_hex(2)
        self._tgt_aff_codes.add(fallback)
        return fallback

    def resolve_access_token(self, value):
        value = (value or "")
        if not value:
            return None
        if value in self._tgt_access_tokens:
            return None  # 目标库已占用 -> 置空
        self._tgt_access_tokens.add(value)
        return value

    def find_username(self, conn, username):
        r = conn.execute(
            select(self.users_tgt.c.id).where(self.users_tgt.c.username == username)
        ).first()
        return r[0] if r else None

    def build_user_values(self, conn, row):
        data = {}
        for col in self.tgt_cols:
            if col == "id":
                continue
            if col == "inviter_id":
                data[col] = 0  # 第二阶段重映射
            elif col == "deleted_at":
                data[col] = row.get("deleted_at")
            elif self.zero_amount and col in USER_ZERO_COLUMNS:
                data[col] = 0
            elif col == "access_token":
                data[col] = self.resolve_access_token(row.get("access_token"))
            elif col == "aff_code":
                data[col] = self.resolve_aff_code(conn, row.get("aff_code"))
            elif col in row:
                data[col] = row[col]
        return data

    # ---------- 主流程 ----------
    def fetch_all_users(self, conn):
        u = self.users_src
        q = select(u)
        if not self.include_deleted and self._has_col("deleted_at"):
            q = q.where(u.c.deleted_at.is_(None))
        rows = {}
        for r in conn.execute(q):
            d = dict(r._mapping)
            rows[d["id"]] = d
        return rows

    def run(self):
        with self.src_engine.connect() as src_conn:
            source_rows = self.fetch_all_users(src_conn)

        ordered_ids = sorted(source_rows.keys())
        tgt_conn = self.tgt_engine.connect()
        trans = tgt_conn.begin()
        try:
            self.preload_unique(tgt_conn)

            for old_id in ordered_ids:
                row = source_rows[old_id]
                username = row.get("username")
                if username and self.find_username(tgt_conn, username) is not None:
                    self.skipped_same_username.append(make_report_row(row))
                    continue
                values = self.build_user_values(tgt_conn, row)
                self.resolved_old_ids.add(old_id)
                if self.execute:
                    res = tgt_conn.execute(insert(self.users_tgt).values(**values))
                    new_id = res.inserted_primary_key[0]
                    self.old_to_new[old_id] = new_id
                    self.inserted.append((old_id, new_id))
                else:
                    self.inserted.append((old_id, None))

            # 第二阶段：重映射 inviter_id
            if self.execute and "inviter_id" in self.tgt_cols:
                for old_id, new_id in self.inserted:
                    src_inviter = source_rows[old_id].get("inviter_id") or 0
                    if src_inviter and self.old_to_new.get(src_inviter):
                        tgt_conn.execute(
                            update(self.users_tgt)
                            .where(self.users_tgt.c.id == new_id)
                            .values(inviter_id=self.old_to_new[src_inviter])
                        )

            # 第三阶段：迁移 tokens
            self.migrate_tokens(tgt_conn)

            if self.execute:
                trans.commit()
            else:
                trans.rollback()
        except Exception:
            trans.rollback()
            raise
        finally:
            tgt_conn.close()

        return {
            "total_users": len(source_rows),
            "imported": len(self.inserted),
            "skipped_same_username": len(self.skipped_same_username),
            "pending_confirmation": len(self.skipped_same_username),
            "skipped_list": self.skipped_same_username,
            "tokens_candidate": self.tokens_candidate,
            "tokens_migrated": self.tokens_migrated,
            "tokens_skipped_key_exists": self.tokens_skipped,
        }

    # ---------- tokens 迁移 ----------
    def migrate_tokens(self, tgt_conn):
        try:
            tok_src = Table(TOKENS_TABLE, MetaData(), autoload_with=self.src_engine)
        except Exception:
            log(f"源库不存在 {TOKENS_TABLE} 表，跳过令牌迁移")
            return
        try:
            tok_tgt = Table(TOKENS_TABLE, MetaData(), autoload_with=self.tgt_engine)
        except Exception:
            log(f"目标库不存在 {TOKENS_TABLE} 表，跳过令牌迁移")
            return

        tok_tgt_cols = list(tok_tgt.c.keys())
        tok_src_has_deleted = "deleted_at" in tok_src.c.keys()
        user_ids = sorted(self.resolved_old_ids)
        if not user_ids:
            return

        # 预载目标库已存在的 token key，避免唯一冲突
        existing_keys = set()
        if "key" in tok_tgt_cols:
            for r in tgt_conn.execute(select(tok_tgt.c.key)):
                if r[0]:
                    existing_keys.add(r[0])

        src_tokens = []
        with self.src_engine.connect() as src_conn:
            for batch in chunked(user_ids):
                q = select(tok_src).where(tok_src.c.user_id.in_(batch))
                if not self.include_deleted and tok_src_has_deleted:
                    q = q.where(tok_src.c.deleted_at.is_(None))
                for r in src_conn.execute(q):
                    src_tokens.append(dict(r._mapping))

        self.tokens_candidate = len(src_tokens)

        for tok in src_tokens:
            key = tok.get("key")
            if key and key in existing_keys:
                self.tokens_skipped += 1
                continue
            if key:
                existing_keys.add(key)

            new_user_id = self.old_to_new.get(tok.get("user_id"))
            if not self.execute:
                self.tokens_migrated += 1
                continue
            if not new_user_id:
                continue
            values = {}
            for col in tok_tgt_cols:
                if col == "id":
                    continue
                if col == "user_id":
                    values[col] = new_user_id
                elif self.zero_amount and col in TOKEN_ZERO_COLUMNS:
                    values[col] = 0
                elif col in tok:
                    values[col] = tok[col]
            tgt_conn.execute(insert(tok_tgt).values(**values))
            self.tokens_migrated += 1


def main():
    parser = argparse.ArgumentParser(
        description="全量迁移用户及其令牌到另一个库",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--source-dsn", default=os.getenv("SOURCE_DSN"), help="源库 DSN（默认读环境变量 SOURCE_DSN）")
    parser.add_argument("--target-dsn", default=os.getenv("TARGET_DSN"), help="目标库 DSN（默认读环境变量 TARGET_DSN）")
    parser.add_argument("--include-deleted", action="store_true", help="包含已软删除用户/令牌（默认不含）")
    parser.add_argument("--keep-amount", action="store_true", help="连同账户余额一起迁移（默认不迁余额：quota/aff_quota/aff_history 置 0）")
    parser.add_argument("--execute", action="store_true", help="真正写入目标库（默认 dry-run 仅预览）")
    parser.add_argument("--report", default="migrate_all_report.json", help="待确认清单输出文件（JSON）")
    args = parser.parse_args()

    if not args.source_dsn or not args.target_dsn:
        parser.error("必须提供 --source-dsn 和 --target-dsn（或环境变量 SOURCE_DSN / TARGET_DSN）")

    src_url = convert_dsn(args.source_dsn)
    tgt_url = convert_dsn(args.target_dsn)

    log("=" * 60)
    log(f"源库   : {re.sub(r':[^:@/]+@', ':***@', src_url)}")
    log(f"目标库 : {re.sub(r':[^:@/]+@', ':***@', tgt_url)}")
    log(f"余额   : {'连余额一起迁(--keep-amount)' if args.keep_amount else '不迁余额(quota/aff_quota/aff_history 置0)'}")
    log(f"模式   : {'执行写入(EXECUTE)' if args.execute else 'DRY-RUN 仅预览'}")
    log("=" * 60)

    src_engine = create_engine(src_url)
    tgt_engine = create_engine(tgt_url)

    migrator = AllUsersMigrator(
        src_engine=src_engine,
        tgt_engine=tgt_engine,
        include_deleted=args.include_deleted,
        zero_amount=not args.keep_amount,
        execute=args.execute,
    )
    result = migrator.run()

    log("")
    log("------ 迁移结果汇总 ------")
    log(f"用户总数              : {result['total_users']}")
    log(f"{'导入成功' if args.execute else '可导入(预览)'}            : {result['imported']}")
    log(f"相同用户名跳过        : {result['skipped_same_username']}")
    log(f"待人工确认            : {result['pending_confirmation']}")
    log(
        f"令牌(候选/{'已迁移' if args.execute else '可迁移'}/key已存在跳过): "
        f"{result['tokens_candidate']}/{result['tokens_migrated']}/{result['tokens_skipped_key_exists']}"
    )

    if result["skipped_list"]:
        log(f"\n[待人工确认-相同用户名] 共 {len(result['skipped_list'])} 条：")
        for it in result["skipped_list"]:
            log(
                f"  - username={it.get('username')} email={it.get('email')} "
                f"phone={it.get('phone')} role={it.get('role')}"
            )

    with open(args.report, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2, default=str)
    log(f"\n详细清单已写入: {os.path.abspath(args.report)}")

    if not args.execute:
        log("\n当前为 DRY-RUN，未对目标库做任何修改。确认无误后加 --execute 正式执行。")


if __name__ == "__main__":
    main()
