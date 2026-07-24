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


def seed_units(cursor) -> None:
    units = [("EA", "piece"), ("KG", "kilogram"), ("M", "meter")]
    cursor.executemany(
        "INSERT IGNORE INTO m_unit(unit_code, unit_name) VALUES (%s, %s)",
        units,
    )


def generate_parts(part_count: int, prefix: str) -> tuple[list[tuple], list[str], list[str], list[str]]:
    products, assemblies, components = split_counts(part_count)

    comp_nos: list[str] = []
    asm_nos: list[str] = []
    prod_nos: list[str] = []
    rows: list[tuple] = []

    idx = 1
    for _ in range(components):
        part_no = f"{prefix}-C-{idx:05d}"
        idx += 1
        comp_nos.append(part_no)
        rows.append((part_no, f"Component-{part_no[-5:]}", "COMPONENT", "EA", Decimal("10.0000"), 1))

    idx = 1
    for _ in range(assemblies):
        part_no = f"{prefix}-A-{idx:05d}"
        idx += 1
        asm_nos.append(part_no)
        rows.append((part_no, f"Assembly-{part_no[-5:]}", "ASSEMBLY", "EA", Decimal("100.0000"), 1))

    idx = 1
    for _ in range(products):
        part_no = f"{prefix}-P-{idx:05d}"
        idx += 1
        prod_nos.append(part_no)
        rows.append((part_no, f"Product-{part_no[-5:]}", "PRODUCT", "EA", Decimal("500.0000"), 1))

    return rows, comp_nos, asm_nos, prod_nos


def insert_parts(cursor, part_rows: list[tuple]) -> None:
    cursor.executemany(
        """
        INSERT INTO m_part(part_no, part_name, part_type, unit_code, standard_cost, is_active)
        VALUES (%s, %s, %s, %s, %s, %s)
        """,
        part_rows,
    )


def fetch_part_map(cursor, prefix: str) -> dict[str, int]:
    cursor.execute(
        "SELECT part_no, part_id FROM m_part WHERE part_no LIKE %s",
        (f"{prefix}-%",),
    )
    return {part_no: int(part_id) for (part_no, part_id) in cursor.fetchall()}


def insert_bom(
    cursor,
    part_map: dict[str, int],
    comp_nos: list[str],
    asm_nos: list[str],
    prod_nos: list[str],
    prefix: str,
    seed: int,
) -> tuple[int, int]:
    today = dt.date.today()
    bom_rows: list[tuple] = []

    for parent_no in asm_nos + prod_nos:
        parent_id = part_map[parent_no]
        bom_rows.append((parent_id, "A", f"BOM-{parent_no}", today, None, 1))

    cursor.executemany(
        """
        INSERT INTO t_bom(parent_part_id, revision, bom_name, effective_from, effective_to, is_released)
        VALUES (%s, %s, %s, %s, %s, %s)
        """,
        bom_rows,
    )

    cursor.execute(
        """
        SELECT bom_id, parent_part_id
        FROM t_bom
        WHERE revision = 'A' AND bom_name LIKE %s
        """,
        (f"BOM-{prefix}-%",),
    )
    bom_by_parent = {parent_id: bom_id for (bom_id, parent_id) in cursor.fetchall()}

    bom_item_rows: list[tuple] = []
    rng = random.Random(seed)

    for parent_no in asm_nos:
        parent_id = part_map[parent_no]
        bom_id = bom_by_parent[parent_id]
        children = rng.sample(comp_nos, k=min(len(comp_nos), rng.randint(3, 7)))
        line = 10
        for child_no in children:
            child_id = part_map[child_no]
            qty = Decimal(str(round(rng.uniform(1.0, 8.0), 3)))
            bom_item_rows.append((bom_id, line, child_id, qty, Decimal("0.0200"), Decimal("0.9800"), 0, None))
            line += 10

    for parent_no in prod_nos:
        parent_id = part_map[parent_no]
        bom_id = bom_by_parent[parent_id]

        asm_children = rng.sample(asm_nos, k=min(len(asm_nos), rng.randint(2, 4)))
        comp_children = rng.sample(comp_nos, k=min(len(comp_nos), rng.randint(1, 3)))
        children = asm_children + comp_children

        line = 10
        for child_no in children:
            child_id = part_map[child_no]
            qty = Decimal(str(round(rng.uniform(1.0, 5.0), 3)))
            bom_item_rows.append((bom_id, line, child_id, qty, Decimal("0.0100"), Decimal("0.9900"), 0, None))
            line += 10

    cursor.executemany(
        """
        INSERT INTO t_bom_item(
          bom_id, line_no, child_part_id, qty_per_parent, scrap_rate, yield_rate, is_phantom, note
        )
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        """,
        bom_item_rows,
    )

    return len(bom_rows), len(bom_item_rows)


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

        seed_units(cursor)

        part_rows, comp_nos, asm_nos, prod_nos = generate_parts(args.part_count, args.prefix)
        insert_parts(cursor, part_rows)

        part_map = fetch_part_map(cursor, args.prefix)
        bom_count, bom_item_count = insert_bom(
            cursor,
            part_map,
            comp_nos,
            asm_nos,
            prod_nos,
            args.prefix,
            args.seed,
        )

        conn.commit()
        print("Inserted mock data successfully:")
        print(f"  database      : {args.database}")
        print(f"  prefix        : {args.prefix}")
        print(f"  parts         : {len(part_rows)}")
        print(f"  boms          : {bom_count}")
        print(f"  bom_items     : {bom_item_count}")
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
