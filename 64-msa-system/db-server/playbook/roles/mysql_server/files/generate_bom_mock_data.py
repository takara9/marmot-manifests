#!/usr/bin/env python3
"""Generate and insert mock BOM data into MySQL.

This script creates a practical dataset for the following tables:
  - m_unit
  - m_part
  - t_bom
  - t_bom_item

Default behavior inserts 1000 part records and related BOM rows.
"""

from __future__ import annotations

import argparse
import datetime as dt
import random
import sys
from decimal import Decimal

MYSQL_CONNECTOR_AVAILABLE = True
MYSQL_CONNECTOR_ERROR = ""

try:
    import mysql.connector
    from mysql.connector import Error
except Exception:  # pragma: no cover - runtime dependency check
    MYSQL_CONNECTOR_AVAILABLE = False
    MYSQL_CONNECTOR_ERROR = "mysql-connector-python is not installed"
    Error = Exception


SCHEMA_SQL = [
    """
    CREATE TABLE IF NOT EXISTS m_unit (
      unit_code VARCHAR(16) NOT NULL,
      unit_name VARCHAR(64) NOT NULL,
      PRIMARY KEY (unit_code)
    ) ENGINE=InnoDB
    """,
    """
    CREATE TABLE IF NOT EXISTS m_part (
      part_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
      part_no VARCHAR(64) NOT NULL,
      part_name VARCHAR(255) NOT NULL,
      part_type ENUM('PRODUCT','ASSEMBLY','COMPONENT','MATERIAL') NOT NULL DEFAULT 'COMPONENT',
      unit_code VARCHAR(16) NOT NULL,
      standard_cost DECIMAL(15,4) NOT NULL DEFAULT 0,
      is_active TINYINT(1) NOT NULL DEFAULT 1,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (part_id),
      UNIQUE KEY uq_part_no (part_no),
      KEY idx_part_name (part_name),
      CONSTRAINT fk_part_unit FOREIGN KEY (unit_code) REFERENCES m_unit(unit_code)
    ) ENGINE=InnoDB
    """,
    """
    CREATE TABLE IF NOT EXISTS t_bom (
      bom_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
      parent_part_id BIGINT UNSIGNED NOT NULL,
      revision VARCHAR(32) NOT NULL,
      bom_name VARCHAR(255) NULL,
      effective_from DATE NOT NULL,
      effective_to DATE NULL,
      is_released TINYINT(1) NOT NULL DEFAULT 0,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (bom_id),
      UNIQUE KEY uq_parent_revision (parent_part_id, revision),
      KEY idx_parent_effective (parent_part_id, effective_from, effective_to),
      CONSTRAINT fk_bom_parent_part FOREIGN KEY (parent_part_id) REFERENCES m_part(part_id),
      CHECK (effective_to IS NULL OR effective_to >= effective_from)
    ) ENGINE=InnoDB
    """,
    """
    CREATE TABLE IF NOT EXISTS t_bom_item (
      bom_item_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
      bom_id BIGINT UNSIGNED NOT NULL,
      line_no INT NOT NULL,
      child_part_id BIGINT UNSIGNED NOT NULL,
      qty_per_parent DECIMAL(18,6) NOT NULL,
      scrap_rate DECIMAL(7,4) NOT NULL DEFAULT 0,
      yield_rate DECIMAL(7,4) NOT NULL DEFAULT 1,
      is_phantom TINYINT(1) NOT NULL DEFAULT 0,
      note VARCHAR(255) NULL,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (bom_item_id),
      UNIQUE KEY uq_bom_line (bom_id, line_no),
      UNIQUE KEY uq_bom_child (bom_id, child_part_id, line_no),
      KEY idx_child_part (child_part_id),
      CONSTRAINT fk_bom_item_bom FOREIGN KEY (bom_id) REFERENCES t_bom(bom_id) ON DELETE CASCADE,
      CONSTRAINT fk_bom_item_child_part FOREIGN KEY (child_part_id) REFERENCES m_part(part_id),
      CHECK (qty_per_parent > 0),
      CHECK (scrap_rate >= 0 AND scrap_rate < 1),
      CHECK (yield_rate > 0 AND yield_rate <= 1)
    ) ENGINE=InnoDB
    """,
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate and insert mock BOM data")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=3306)
    parser.add_argument("--user", required=True)
    parser.add_argument("--password", required=True)
    parser.add_argument("--database", default="bomdb")
    parser.add_argument("--part-count", type=int, default=1000)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument(
        "--prefix",
        default=f"MOCK{dt.datetime.now().strftime('%Y%m%d%H%M%S')}",
        help="part_no prefix to avoid duplicate key collisions",
    )
    parser.add_argument(
        "--create-schema",
        action="store_true",
        help="create database/tables if they do not exist",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=2000,
        help="executemany batch size to reduce memory usage (default: 2000)",
    )
    parser.add_argument(
        "--io-heavy-indexes",
        action="store_true",
        help="drop selected secondary indexes to increase I/O load",
    )
    return parser.parse_args()


def split_counts(total: int) -> tuple[int, int, int]:
    products = max(20, int(total * 0.08))
    assemblies = max(120, int(total * 0.22))
    if products + assemblies >= total:
        assemblies = max(1, total // 4)
        products = max(1, total // 10)
    components = total - products - assemblies
    return products, assemblies, components


def ensure_schema(cursor, database: str) -> None:
    cursor.execute(
        f"CREATE DATABASE IF NOT EXISTS {database} "
        "DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_0900_ai_ci"
    )
    cursor.execute(f"USE {database}")
    for stmt in SCHEMA_SQL:
        cursor.execute(stmt)


def drop_secondary_indexes_for_io_load(cursor, database: str) -> int:
    # Keep PK/UNIQUE/foreign key related indexes and drop only selected non-unique ones.
    candidates = [
        ("m_part", "idx_part_name"),
        ("t_bom", "idx_parent_effective"),
    ]
    dropped = 0

    for table_name, index_name in candidates:
        cursor.execute(
            """
            SELECT 1
            FROM information_schema.statistics
            WHERE table_schema = %s AND table_name = %s AND index_name = %s
            LIMIT 1
            """,
            (database, table_name, index_name),
        )
        if cursor.fetchone() is None:
            continue
        try:
            cursor.execute(f"ALTER TABLE {table_name} DROP INDEX {index_name}")
            dropped += 1
        except Error as err:
            # e.g. 1553: index is required for foreign key constraint
            if getattr(err, "errno", None) == 1553:
                continue
            raise

    return dropped


def seed_units(cursor) -> None:
    units = [("EA", "piece"), ("KG", "kilogram"), ("M", "meter")]
    cursor.executemany(
        "INSERT IGNORE INTO m_unit(unit_code, unit_name) VALUES (%s, %s)",
        units,
    )


def flush_batch(cursor, sql: str, rows: list[tuple]) -> int:
    if not rows:
        return 0
    cursor.executemany(sql, rows)
    flushed = len(rows)
    rows.clear()
    return flushed


def insert_parts(cursor, part_count: int, prefix: str, batch_size: int) -> tuple[int, int, int, int]:
    products, assemblies, components = split_counts(part_count)
    sql = """
    INSERT INTO m_part(part_no, part_name, part_type, unit_code, standard_cost, is_active)
    VALUES (%s, %s, %s, %s, %s, %s)
    """
    rows: list[tuple] = []
    inserted = 0

    for idx in range(1, components + 1):
        part_no = f"{prefix}-C-{idx:05d}"
        rows.append((part_no, f"Component-{idx:05d}", "COMPONENT", "EA", Decimal("10.0000"), 1))
        if len(rows) >= batch_size:
            inserted += flush_batch(cursor, sql, rows)

    for idx in range(1, assemblies + 1):
        part_no = f"{prefix}-A-{idx:05d}"
        rows.append((part_no, f"Assembly-{idx:05d}", "ASSEMBLY", "EA", Decimal("100.0000"), 1))
        if len(rows) >= batch_size:
            inserted += flush_batch(cursor, sql, rows)

    for idx in range(1, products + 1):
        part_no = f"{prefix}-P-{idx:05d}"
        rows.append((part_no, f"Product-{idx:05d}", "PRODUCT", "EA", Decimal("500.0000"), 1))
        if len(rows) >= batch_size:
            inserted += flush_batch(cursor, sql, rows)

    inserted += flush_batch(cursor, sql, rows)
    return inserted, components, assemblies, products


def fetch_part_id_arrays(
    cursor,
    prefix: str,
    components: int,
    assemblies: int,
    products: int,
) -> tuple[list[int], list[int], list[int]]:
    comp_ids: list[int] = [0] * components
    asm_ids: list[int] = [0] * assemblies
    prod_ids: list[int] = [0] * products

    cursor.execute(
        "SELECT part_no, part_id FROM m_part WHERE part_no LIKE %s",
        (f"{prefix}-%",),
    )

    for part_no, part_id in cursor.fetchall():
        base, kind, index_str = str(part_no).rsplit("-", 2)
        if base != prefix:
            continue
        index = int(index_str) - 1
        id_value = int(part_id)
        if kind == "C" and 0 <= index < components:
            comp_ids[index] = id_value
        elif kind == "A" and 0 <= index < assemblies:
            asm_ids[index] = id_value
        elif kind == "P" and 0 <= index < products:
            prod_ids[index] = id_value

    if any(v == 0 for v in comp_ids) or any(v == 0 for v in asm_ids) or any(v == 0 for v in prod_ids):
        raise RuntimeError("Failed to resolve all inserted part IDs. Use a unique --prefix and retry.")

    return comp_ids, asm_ids, prod_ids


def insert_bom(
    cursor,
    comp_ids: list[int],
    asm_ids: list[int],
    prod_ids: list[int],
    prefix: str,
    seed: int,
    batch_size: int,
) -> tuple[int, int]:
    today = dt.date.today()
    bom_sql = """
    INSERT INTO t_bom(parent_part_id, revision, bom_name, effective_from, effective_to, is_released)
    VALUES (%s, %s, %s, %s, %s, %s)
    """
    bom_rows: list[tuple] = []
    bom_count = 0

    for idx, parent_id in enumerate(asm_ids, start=1):
        parent_no = f"{prefix}-A-{idx:05d}"
        bom_rows.append((parent_id, "A", f"BOM-{parent_no}", today, None, 1))
        if len(bom_rows) >= batch_size:
            bom_count += flush_batch(cursor, bom_sql, bom_rows)

    for idx, parent_id in enumerate(prod_ids, start=1):
        parent_no = f"{prefix}-P-{idx:05d}"
        bom_rows.append((parent_id, "A", f"BOM-{parent_no}", today, None, 1))
        if len(bom_rows) >= batch_size:
            bom_count += flush_batch(cursor, bom_sql, bom_rows)

    bom_count += flush_batch(cursor, bom_sql, bom_rows)

    cursor.execute(
        """
        SELECT bom_id, parent_part_id
        FROM t_bom
        WHERE revision = 'A' AND bom_name LIKE %s
        """,
        (f"BOM-{prefix}-%",),
    )
    bom_by_parent = {parent_id: bom_id for (bom_id, parent_id) in cursor.fetchall()}

    bom_item_sql = """
    INSERT INTO t_bom_item(
      bom_id, line_no, child_part_id, qty_per_parent, scrap_rate, yield_rate, is_phantom, note
    )
    VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
    """
    bom_item_rows: list[tuple] = []
    bom_item_count = 0
    rng = random.Random(seed)
    comp_range = range(len(comp_ids))
    asm_range = range(len(asm_ids))

    if len(comp_ids) < 3:
        raise RuntimeError("Not enough component parts to build BOM items.")
    if len(asm_ids) < 2:
        raise RuntimeError("Not enough assembly parts to build product BOM items.")

    for parent_id in asm_ids:
        bom_id = bom_by_parent[parent_id]
        children = rng.sample(comp_range, k=min(len(comp_ids), rng.randint(3, 7)))
        line = 10
        for child_idx in children:
            child_id = comp_ids[child_idx]
            qty = rng.randint(1, 8)
            bom_item_rows.append((bom_id, line, child_id, qty, Decimal("0.0200"), Decimal("0.9800"), 0, None))
            if len(bom_item_rows) >= batch_size:
                bom_item_count += flush_batch(cursor, bom_item_sql, bom_item_rows)
            line += 10

    for parent_id in prod_ids:
        bom_id = bom_by_parent[parent_id]

        asm_children = rng.sample(asm_range, k=min(len(asm_ids), rng.randint(2, 4)))
        comp_children = rng.sample(comp_range, k=min(len(comp_ids), rng.randint(1, 3)))
        children = [("A", idx) for idx in asm_children] + [("C", idx) for idx in comp_children]

        line = 10
        for child_kind, child_idx in children:
            child_id = asm_ids[child_idx] if child_kind == "A" else comp_ids[child_idx]
            qty = rng.randint(1, 5)
            bom_item_rows.append((bom_id, line, child_id, qty, Decimal("0.0100"), Decimal("0.9900"), 0, None))
            if len(bom_item_rows) >= batch_size:
                bom_item_count += flush_batch(cursor, bom_item_sql, bom_item_rows)
            line += 10

    bom_item_count += flush_batch(cursor, bom_item_sql, bom_item_rows)

    return bom_count, bom_item_count


def main() -> int:
    args = parse_args()
    random.seed(args.seed)

    if not MYSQL_CONNECTOR_AVAILABLE:
        print(MYSQL_CONNECTOR_ERROR, file=sys.stderr)
        print("", file=sys.stderr)
        print("Use a virtual environment and run again:", file=sys.stderr)
        print("  python3 -m venv .venv", file=sys.stderr)
        print("  . .venv/bin/activate", file=sys.stderr)
        print("  pip install mysql-connector-python", file=sys.stderr)
        print(
            "  python generate_bom_mock_data.py --host 10.1.1.10 --user mysqladmin --password <PASSWORD> --database bomdb --part-count 1000 --create-schema",
            file=sys.stderr,
        )
        return 2

    if args.part_count < 100:
        print("--part-count must be >= 100 for realistic BOM distribution", file=sys.stderr)
        return 2
    if args.batch_size < 100:
        print("--batch-size must be >= 100", file=sys.stderr)
        return 2

    conn = None
    try:
        conn = mysql.connector.connect(
            host=args.host,
            port=args.port,
            user=args.user,
            password=args.password,
            autocommit=False,
        )
        cursor = conn.cursor()

        if args.create_schema:
            ensure_schema(cursor, args.database)
        else:
            cursor.execute(f"USE {args.database}")

        dropped_indexes = 0
        if args.io_heavy_indexes:
            dropped_indexes = drop_secondary_indexes_for_io_load(cursor, args.database)

        seed_units(cursor)

        inserted_parts, components, assemblies, products = insert_parts(
            cursor,
            args.part_count,
            args.prefix,
            args.batch_size,
        )

        comp_ids, asm_ids, prod_ids = fetch_part_id_arrays(
            cursor,
            args.prefix,
            components,
            assemblies,
            products,
        )
        bom_count, bom_item_count = insert_bom(
            cursor,
            comp_ids,
            asm_ids,
            prod_ids,
            args.prefix,
            args.seed,
            args.batch_size,
        )

        conn.commit()
        print("Inserted mock data successfully:")
        print(f"  database      : {args.database}")
        print(f"  prefix        : {args.prefix}")
        print(f"  parts         : {inserted_parts}")
        print(f"  boms          : {bom_count}")
        print(f"  bom_items     : {bom_item_count}")
        print(f"  batch_size    : {args.batch_size}")
        if args.io_heavy_indexes:
            print(f"  dropped_idx   : {dropped_indexes}")
        return 0

    except Error as err:
        if conn is not None:
            conn.rollback()
        print(f"MySQL error: {err}", file=sys.stderr)
        return 1
    except Exception as exc:
        if conn is not None:
            conn.rollback()
        print(f"Unexpected error: {exc}", file=sys.stderr)
        return 1
    finally:
        if conn is not None and conn.is_connected():
            conn.close()


if __name__ == "__main__":
    sys.exit(main())
