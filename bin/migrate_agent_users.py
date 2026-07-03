#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""跨环境迁移「代理用户」及其名下整棵邀请树（不迁移金额/用量）。

适用场景：有多个部署环境，需要把源库中「某截止时间以前注册的代理用户」以及
这些代理名下的所有被邀请用户（按 inviter_id 递归整棵树）整体搬到目标库。

迁移内容：
  - 迁移：账户(username)、密码(已是哈希，原样复制)、显示名、角色(role)、状态(status)、
    分组(group)、权限/边栏设置(setting)、手机号(phone)、邮箱(email)、各 OAuth 绑定、
    代理身份(is_distributor)、学员身份、备注、标签、注册时间(created_at)等。
  - 不迁移（目标库置 0）：quota / used_quota / request_count / aff_count / aff_quota / aff_history。
  - 置空避免唯一冲突：access_token、stripe_customer。
  - aff_code：尽量保留；若目标库已被占用则随机重新生成一个唯一值。
  - inviter_id：迁移后会被重映射到目标库中的新主键（保持邀请关系），
    无法映射（邀请人未迁移/被跳过）时置 0。

冲突判定（写入目标库前对每个用户检查）：
  1. 目标库存在同名(username)，且 手机号或邮箱任一一致 -> 判定为同一人：跳过，记入「已存在(同一人)」清单；
     其下级邀请关系会挂到目标库已存在的这个用户上。
  2. 目标库存在同名，但 手机号与邮箱都不一致 -> 跳过，记入「同名待人工处理」清单。
  3. 目标库无同名，但 手机号或邮箱 已被「其他用户名」占用 -> 跳过，记入「手机/邮箱被他人占用」清单
     （额外安全检查，避免破坏应用层手机号/邮箱唯一性）。

默认是 dry-run（只预览不写入）。确认无误后加 --execute 真正写入目标库。

依赖：
  pip install -r bin/requirements-migrate.txt

DSN 既可用 SQLAlchemy URL，也可直接用项目里 Go 风格的 SQL_DSN：
  - MySQL(Go 风格)：user:pass@tcp(127.0.0.1:3306)/dbname  ->  自动转 mysql+pymysql://...
  - PostgreSQL：postgres://user:pass@host:5432/dbname     ->  自动转 postgresql://...
  - SQLite：/path/to/one-api.db 或 sqlite:////abs/path.db

示例：    # 先预览
python bin/migrate_agent_users.py --source-dsn "postgres://root:Fuyi_8888@43.132.223.112:5434/token_factory" 
--target-dsn "postgres://root:Fuyi_8888@43.138.142.105:5434/token_factory" --before "2026-06-08" --report migrate_report.json
 
   
  # 确认后：
python bin/migrate_agent_users.py --source-dsn "postgres://root:Fuyi_8888@43.132.223.112:5434/token_factory" --target-dsn "postgres://root:Fuyi_8888@43.138.142.105:5434/token_factory" --before "2026-06-08" --report migrate_report.json --execute
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
    from sqlalchemy import (
        MetaData,
        Table,
        and_,
        create_engine,
        insert,
        or_,
        select,
        update,
    )
except ImportError:
    sys.stderr.write(
        "缺少依赖 SQLAlchemy，请先执行：pip install -r bin/requirements-migrate.txt\n"
    )
    sys.exit(1)

USERS_TABLE = "users"

# 金额 / 用量字段：不迁移，目标库统一置 0
ZERO_COLUMNS = {
    "quota",
    "used_quota",
    "request_count",
    "aff_count",
    "aff_quota",
    "aff_history",
}

# 迁移易引发唯一冲突且与金额/会话相关，目标库置空
NULL_COLUMNS = {"access_token", "stripe_customer"}

# OAuth/第三方绑定列：两个环境相互独立，迁入时若目标库已存在相同绑定值则置空，保证目标库内不重复
BINDING_COLUMNS = {
    "github_id",
    "discord_id",
    "oidc_id",
    "wechat_id",
    "telegram_id",
    "linux_do_id",
}

# aff_invite_relations 中属于金额/累计收益的列：不迁移，置 0
RELATION_ZERO_COLUMNS = {"commission_earned_quota", "profit_share_earned_quota"}
RELATIONS_TABLE = "aff_invite_relations"

# 迁移到目标库时仅用于判重展示的关键字段
REPORT_FIELDS = ["old_id", "username", "display_name", "email", "phone", "role", "is_distributor"]

IN_CHUNK = 500


def log(msg):
    print(msg, flush=True)


def convert_dsn(raw):
    """把 Go 风格 SQL_DSN / postgres:// / sqlite 路径转换为 SQLAlchemy URL。"""
    raw = (raw or "").strip()
    if not raw:
        raise ValueError("DSN 为空")

    # 已是 SQLAlchemy URL（含 driver），直接用
    if re.match(r"^(mysql\+\w+|postgresql(\+\w+)?|sqlite)://", raw):
        return raw

    if raw.startswith("postgres://"):
        return "postgresql://" + raw[len("postgres://") :]
    if raw.startswith("postgresql://"):
        return raw

    # Go 风格 MySQL: user:pass@tcp(host:port)/db?params
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

    # SQLite：local 关键字或文件路径
    if raw == "local":
        raw = "one-api.db"
    if raw.endswith(".db") or raw.endswith(".sqlite") or os.path.exists(raw):
        abspath = os.path.abspath(raw)
        return "sqlite:///" + abspath

    raise ValueError(f"无法识别的 DSN 格式：{raw}")


def parse_cutoff(value):
    if not value:
        return None
    value = value.strip()
    for fmt in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%d"):
        try:
            return datetime.strptime(value, fmt)
        except ValueError:
            continue
    raise ValueError("--before 时间格式应为 YYYY-MM-DD 或 YYYY-MM-DD HH:MM:SS")


def chunked(items, size=IN_CHUNK):
    items = list(items)
    for i in range(0, len(items), size):
        yield items[i : i + size]


def make_report_row(row, reason):
    item = {}
    for f in REPORT_FIELDS:
        key = "id" if f == "old_id" else f
        item[f] = row.get(key)
    item["reason"] = reason
    return item


def gen_aff_code():
    alphabet = string.ascii_letters + string.digits
    return "".join(secrets.choice(alphabet) for _ in range(6))


class Migrator:
    def __init__(self, src_engine, tgt_engine, cutoff, include_deleted, execute):
        self.src_engine = src_engine
        self.tgt_engine = tgt_engine
        self.cutoff = cutoff
        self.include_deleted = include_deleted
        self.execute = execute

        self.users_src = Table(USERS_TABLE, MetaData(), autoload_with=src_engine)
        self.users_tgt = Table(USERS_TABLE, MetaData(), autoload_with=tgt_engine)
        self.src_cols = set(self.users_src.c.keys())
        self.tgt_cols = list(self.users_tgt.c.keys())

        for required in ("id", "username", "is_distributor", "inviter_id"):
            if required not in self.src_cols:
                raise RuntimeError(f"源库 users 表缺少必要列：{required}")

        self._tgt_aff_codes = set()
        # 迁入过程中已占用的绑定值（含目标库已有 + 本次已分配），避免目标库内重复
        self._assigned_bindings = {col: set() for col in BINDING_COLUMNS}

        # 结果清单
        self.inserted = []  # [(old_id, new_id)]
        self.skipped_existing = []  # 同名且手机/邮箱一致 -> 同一人
        self.conflict_manual = []  # 同名但手机/邮箱都不一致
        self.phone_email_taken = []  # 手机/邮箱被其他用户名占用
        self.old_to_new = {}  # old_id -> 目标库 id（含新插入与判定为同一人的；dry-run 下新插入为 None）
        self.resolved_old_ids = set()  # 已落库或将落库（新插入/同一人）的源 id，两种模式都填充
        self.dropped_bindings = []  # 迁入时因目标库已存在而被置空的绑定 [{username, column, value}]
        self.relations_migrated = 0
        self.relations_skipped = 0
        self.relations_candidate = 0

    # ---------- 选取待迁移用户 ----------
    def _has_col(self, name):
        return name in self.src_cols

    def collect_user_ids(self, conn):
        u = self.users_src
        q = select(u.c.id).where(u.c.is_distributor == 1)
        if self.cutoff is not None and self._has_col("created_at"):
            q = q.where(u.c.created_at < self.cutoff)
        if not self.include_deleted and self._has_col("deleted_at"):
            q = q.where(u.c.deleted_at.is_(None))
        agent_ids = [r[0] for r in conn.execute(q)]
        log(f"匹配到代理用户 {len(agent_ids)} 个")

        collected = set(agent_ids)
        frontier = set(agent_ids)
        while frontier:
            new_ids = set()
            for batch in chunked([fid for fid in frontier if fid]):
                if not batch:
                    continue
                sub = select(u.c.id).where(u.c.inviter_id.in_(batch))
                if not self.include_deleted and self._has_col("deleted_at"):
                    sub = sub.where(u.c.deleted_at.is_(None))
                for r in conn.execute(sub):
                    new_ids.add(r[0])
            new_ids -= collected
            collected |= new_ids
            frontier = new_ids
        log(f"含递归邀请树共需迁移 {len(collected)} 个用户")
        return agent_ids, collected

    def fetch_rows(self, conn, ids):
        u = self.users_src
        rows = {}
        for batch in chunked(ids):
            for r in conn.execute(select(u).where(u.c.id.in_(batch))):
                d = dict(r._mapping)
                rows[d["id"]] = d
        return rows

    # ---------- 目标库查询 ----------
    def preload_aff_codes(self, conn):
        if "aff_code" not in self.tgt_cols:
            return
        for r in conn.execute(select(self.users_tgt.c.aff_code)):
            if r[0]:
                self._tgt_aff_codes.add(r[0])

    def find_by_username(self, conn, username):
        r = conn.execute(
            select(self.users_tgt).where(self.users_tgt.c.username == username)
        ).first()
        return dict(r._mapping) if r else None

    def find_by_phone_or_email(self, conn, row, exclude_username):
        conds = []
        t = self.users_tgt
        phone = (row.get("phone") or "").strip()
        email = (row.get("email") or "").strip()
        if phone and "phone" in self.tgt_cols:
            conds.append(t.c.phone == phone)
        if email and "email" in self.tgt_cols:
            conds.append(t.c.email == email)
        if not conds:
            return None
        q = select(t).where(or_(*conds)).where(t.c.username != exclude_username)
        r = conn.execute(q).first()
        return dict(r._mapping) if r else None

    def resolve_aff_code(self, conn, code):
        if "aff_code" not in self.tgt_cols:
            return None
        code = (code or "").strip()
        if code and code not in self._tgt_aff_codes:
            self._tgt_aff_codes.add(code)
            return code
        # 冲突或为空 -> 生成唯一
        for _ in range(20):
            new_code = gen_aff_code()
            if new_code in self._tgt_aff_codes:
                continue
            exists = conn.execute(
                select(self.users_tgt.c.id).where(self.users_tgt.c.aff_code == new_code)
            ).first()
            if not exists:
                self._tgt_aff_codes.add(new_code)
                return new_code
        # 极端兜底
        fallback = gen_aff_code() + secrets.token_hex(2)
        self._tgt_aff_codes.add(fallback)
        return fallback

    def binding_value_taken(self, conn, col, value):
        if value in self._assigned_bindings.get(col, set()):
            return True
        exists = conn.execute(
            select(self.users_tgt.c[col]).where(self.users_tgt.c[col] == value)
        ).first()
        return bool(exists)

    def resolve_binding(self, conn, col, value, username):
        """两环境独立：若该绑定值已存在于目标库（或本次已分配），置空避免重复。"""
        value = (value or "").strip()
        if not value:
            return ""
        if self.binding_value_taken(conn, col, value):
            self.dropped_bindings.append(
                {"username": username, "column": col, "value": value}
            )
            return ""
        self._assigned_bindings[col].add(value)
        return value

    def build_insert_values(self, conn, row, now):
        data = {}
        username = row.get("username")
        for col in self.tgt_cols:
            if col == "id":
                continue
            if col in ZERO_COLUMNS:
                data[col] = 0
            elif col == "inviter_id":
                data[col] = 0  # 第二阶段重映射
            elif col == "deleted_at":
                data[col] = None
            elif col == "updated_at":
                data[col] = now
            elif col in NULL_COLUMNS:
                data[col] = None
            elif col == "aff_code":
                data[col] = self.resolve_aff_code(conn, row.get("aff_code"))
            elif col in BINDING_COLUMNS:
                data[col] = self.resolve_binding(conn, col, row.get(col), username)
            elif col in row:
                data[col] = row[col]
        return data

    # ---------- 主流程 ----------
    def run(self):
        with self.src_engine.connect() as src_conn:
            agent_ids, all_ids = self.collect_user_ids(src_conn)
            source_rows = self.fetch_rows(src_conn, all_ids)

        # 让代理/邀请人尽量先于其下级处理（按 id 升序近似拓扑，配合二次重映射可保证正确）
        ordered_ids = sorted(source_rows.keys())

        tgt_conn = self.tgt_engine.connect()
        trans = tgt_conn.begin()
        try:
            self.preload_aff_codes(tgt_conn)
            now = datetime.now()

            for old_id in ordered_ids:
                row = source_rows[old_id]
                username = row.get("username")
                existing = self.find_by_username(tgt_conn, username) if username else None
                if existing:
                    same = False
                    src_phone = (row.get("phone") or "").strip()
                    src_email = (row.get("email") or "").strip()
                    if src_phone and src_phone == (existing.get("phone") or "").strip():
                        same = True
                    if src_email and src_email == (existing.get("email") or "").strip():
                        same = True
                    if same:
                        self.skipped_existing.append(
                            make_report_row(row, "目标库已存在同名且手机号/邮箱一致，判定为同一用户")
                        )
                        self.old_to_new[old_id] = existing["id"]
                        self.resolved_old_ids.add(old_id)
                    else:
                        self.conflict_manual.append(
                            make_report_row(row, "目标库存在同名但手机号与邮箱均不一致，需人工处理")
                        )
                    continue

                other = self.find_by_phone_or_email(tgt_conn, row, exclude_username=username)
                if other:
                    self.phone_email_taken.append(
                        make_report_row(
                            row,
                            f"手机号或邮箱已被目标库其他账户(username={other.get('username')})占用",
                        )
                    )
                    continue

                values = self.build_insert_values(tgt_conn, row, now)
                self.resolved_old_ids.add(old_id)
                if self.execute:
                    res = tgt_conn.execute(insert(self.users_tgt).values(**values))
                    new_id = res.inserted_primary_key[0]
                    self.old_to_new[old_id] = new_id
                    self.inserted.append((old_id, new_id))
                else:
                    self.inserted.append((old_id, None))

            # 第二阶段：重映射 inviter_id
            if self.execute:
                for old_id, new_id in self.inserted:
                    src_inviter = source_rows[old_id].get("inviter_id") or 0
                    if src_inviter and self.old_to_new.get(src_inviter):
                        tgt_conn.execute(
                            update(self.users_tgt)
                            .where(self.users_tgt.c.id == new_id)
                            .values(inviter_id=self.old_to_new[src_inviter])
                        )

            # 第三阶段：迁移 aff_invite_relations（分成比例配置），重映射两端 id，金额置 0
            self.migrate_relations(tgt_conn, now)

            if self.execute:
                trans.commit()
            else:
                trans.rollback()  # dry-run 不留痕
        except Exception:
            trans.rollback()
            raise
        finally:
            tgt_conn.close()

        return {
            "agent_count": len(agent_ids),
            "total_candidates": len(source_rows),
            "inserted": len(self.inserted),
            "skipped_existing_same_person": self.skipped_existing,
            "conflict_same_username_diff_contact": self.conflict_manual,
            "phone_or_email_taken_by_other": self.phone_email_taken,
            "dropped_bindings": self.dropped_bindings,
            "relations_candidate": self.relations_candidate,
            "relations_migrated": self.relations_migrated,
            "relations_skipped_existing": self.relations_skipped,
        }

    # ---------- aff_invite_relations 迁移 ----------
    def migrate_relations(self, tgt_conn, now):
        try:
            rel_src = Table(RELATIONS_TABLE, MetaData(), autoload_with=self.src_engine)
        except Exception:
            log(f"源库不存在 {RELATIONS_TABLE} 表，跳过分成关系迁移")
            return
        try:
            rel_tgt = Table(RELATIONS_TABLE, MetaData(), autoload_with=self.tgt_engine)
        except Exception:
            log(f"目标库不存在 {RELATIONS_TABLE} 表，跳过分成关系迁移")
            return

        rel_tgt_cols = list(rel_tgt.c.keys())
        # 仅迁移两端都已落库/将落库（新插入或判定为同一人）的关系
        mapped_ids = sorted(self.resolved_old_ids)
        if not mapped_ids:
            return
        mapped_set = set(mapped_ids)
        now_ts = int(now.timestamp())

        # 拉取源库中两端均在迁移集合内的关系
        src_rels = []
        with self.src_engine.connect() as src_conn:
            for batch in chunked(mapped_ids):
                q = select(rel_src).where(rel_src.c.inviter_id.in_(batch))
                for r in src_conn.execute(q):
                    d = dict(r._mapping)
                    if d.get("invitee_user_id") in mapped_set:
                        src_rels.append(d)

        self.relations_candidate = len(src_rels)

        for rel in src_rels:
            new_inviter = self.old_to_new.get(rel.get("inviter_id"))
            new_invitee = self.old_to_new.get(rel.get("invitee_user_id"))
            # dry-run 下新插入用户尚无真实 id，仅统计候选数，不实际写入
            if not self.execute:
                self.relations_migrated += 1
                continue
            if not new_inviter or not new_invitee:
                continue
            # 目标库已存在同一对 (inviter, invitee) 则跳过，避免重复
            exists = tgt_conn.execute(
                select(rel_tgt.c.id).where(
                    and_(
                        rel_tgt.c.inviter_id == new_inviter,
                        rel_tgt.c.invitee_user_id == new_invitee,
                    )
                )
            ).first()
            if exists:
                self.relations_skipped += 1
                continue
            values = {}
            for col in rel_tgt_cols:
                if col == "id":
                    continue
                if col == "inviter_id":
                    values[col] = new_inviter
                elif col == "invitee_user_id":
                    values[col] = new_invitee
                elif col in RELATION_ZERO_COLUMNS:
                    values[col] = 0
                elif col == "updated_at":
                    values[col] = now_ts
                elif col == "created_at":
                    values[col] = rel.get("created_at") or now_ts
                elif col in rel:
                    values[col] = rel[col]
            if self.execute:
                tgt_conn.execute(insert(rel_tgt).values(**values))
            self.relations_migrated += 1


def main():
    parser = argparse.ArgumentParser(
        description="跨环境迁移代理用户及其邀请树（不迁移金额）",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--source-dsn", default=os.getenv("SOURCE_DSN"), help="源库 DSN（默认读环境变量 SOURCE_DSN）")
    parser.add_argument("--target-dsn", default=os.getenv("TARGET_DSN"), help="目标库 DSN（默认读环境变量 TARGET_DSN）")
    parser.add_argument("--before", help="只迁移此时间以前注册的代理（YYYY-MM-DD 或 YYYY-MM-DD HH:MM:SS）")
    parser.add_argument("--include-deleted", action="store_true", help="包含已软删除用户（默认不含）")
    parser.add_argument("--execute", action="store_true", help="真正写入目标库（默认 dry-run 仅预览）")
    parser.add_argument("--report", default="migrate_report.json", help="冲突/跳过清单输出文件（JSON）")
    args = parser.parse_args()

    if not args.source_dsn or not args.target_dsn:
        parser.error("必须提供 --source-dsn 和 --target-dsn（或环境变量 SOURCE_DSN / TARGET_DSN）")

    try:
        cutoff = parse_cutoff(args.before)
    except ValueError as e:
        parser.error(str(e))

    src_url = convert_dsn(args.source_dsn)
    tgt_url = convert_dsn(args.target_dsn)

    log("=" * 60)
    log(f"源库   : {re.sub(r':[^:@/]+@', ':***@', src_url)}")
    log(f"目标库 : {re.sub(r':[^:@/]+@', ':***@', tgt_url)}")
    log(f"截止时间 : {cutoff if cutoff else '（不限）'}")
    log(f"模式   : {'执行写入(EXECUTE)' if args.execute else 'DRY-RUN 仅预览'}")
    log("=" * 60)

    src_engine = create_engine(src_url)
    tgt_engine = create_engine(tgt_url)

    migrator = Migrator(
        src_engine=src_engine,
        tgt_engine=tgt_engine,
        cutoff=cutoff,
        include_deleted=args.include_deleted,
        execute=args.execute,
    )
    result = migrator.run()

    log("")
    log("------ 迁移结果汇总 ------")
    log(f"匹配代理用户            : {result['agent_count']}")
    log(f"待迁移用户(含邀请树)    : {result['total_candidates']}")
    log(f"{'已写入' if args.execute else '可写入(预览)'}              : {result['inserted']}")
    log(f"跳过-已存在(同一人)     : {len(result['skipped_existing_same_person'])}")
    log(f"跳过-同名待人工处理     : {len(result['conflict_same_username_diff_contact'])}")
    log(f"跳过-手机/邮箱被他人占用: {len(result['phone_or_email_taken_by_other'])}")
    log(f"绑定置空(目标库已存在)  : {len(result['dropped_bindings'])}")
    log(
        f"分成关系(候选/{'已迁移' if args.execute else '可迁移'}/已存在跳过): "
        f"{result['relations_candidate']}/{result['relations_migrated']}/{result['relations_skipped_existing']}"
    )

    def dump_list(title, items):
        if not items:
            return
        log(f"\n[{title}] 共 {len(items)} 条：")
        for it in items:
            log(
                f"  - username={it.get('username')} email={it.get('email')} "
                f"phone={it.get('phone')} 原因={it.get('reason')}"
            )

    dump_list("已存在(同一人，未迁移)", result["skipped_existing_same_person"])
    dump_list("同名待人工处理", result["conflict_same_username_diff_contact"])
    dump_list("手机/邮箱被他人占用", result["phone_or_email_taken_by_other"])

    if result["dropped_bindings"]:
        log(f"\n[绑定置空(目标库已存在相同绑定)] 共 {len(result['dropped_bindings'])} 条：")
        for it in result["dropped_bindings"]:
            log(f"  - username={it['username']} {it['column']}={it['value']} -> 已置空")

    with open(args.report, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2, default=str)
    log(f"\n详细清单已写入: {os.path.abspath(args.report)}")

    if not args.execute:
        log("\n当前为 DRY-RUN，未对目标库做任何修改。确认无误后加 --execute 正式执行。")


if __name__ == "__main__":
    main()
