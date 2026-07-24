#!/usr/bin/env python3
"""Generate transactional load on bomdb.

This script runs concurrent workers that repeatedly execute:
  - INSERT into a load table
  - UPDATE random row in the load table
  - SELECT against BOM tables
  - COMMIT or ROLLBACK based on configured ratio

It is intended for test/staging environments only.
"""

from __future__ import annotations

import argparse
import random
import sys
import threading
import time
from dataclasses import dataclass

MYSQL_CONNECTOR_AVAILABLE = True
MYSQL_CONNECTOR_ERROR = ""

try:
    import mysql.connector
    from mysql.connector import Error
except Exception:
    MYSQL_CONNECTOR_AVAILABLE = False
    MYSQL_CONNECTOR_ERROR = "mysql-connector-python is not installed"
    Error = Exception


CREATE_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS t_tx_load (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  worker_id INT NOT NULL,
  tx_tag VARCHAR(64) NOT NULL,
  part_id BIGINT UNSIGNED NULL,
  qty DECIMAL(18,6) NOT NULL,
  payload VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_worker_id (worker_id),
  KEY idx_part_id (part_id)
) ENGINE=InnoDB
"""


@dataclass
class SharedStats:
    lock: threading.Lock
    commits: int = 0
    rollbacks: int = 0
    errors: int = 0
    tx_total: int = 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Transactional stress tool for bomdb")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=3306)
    parser.add_argument("--user", required=True)
    parser.add_argument("--password", required=True)
    parser.add_argument("--database", default="bomdb")
    parser.add_argument("--workers", type=int, default=8, help="number of concurrent worker threads")
    parser.add_argument("--duration", type=int, default=120, help="test duration in seconds")
    parser.add_argument("--rollback-ratio", type=float, default=0.2, help="ratio of transactions to rollback")
    parser.add_argument("--sleep-ms", type=int, default=10, help="sleep between transactions per worker")
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--create-table", action="store_true", help="create t_tx_load table if needed")
    parser.add_argument("--verbose-errors", action="store_true")
    return parser.parse_args()


def check_dependency_or_exit() -> None:
    if MYSQL_CONNECTOR_AVAILABLE:
        return
    print(MYSQL_CONNECTOR_ERROR, file=sys.stderr)
    print("", file=sys.stderr)
    print("Use a virtual environment and install dependency:", file=sys.stderr)
    print("  python3 -m venv .venv", file=sys.stderr)
    print("  . .venv/bin/activate", file=sys.stderr)
    print("  python -m pip install mysql-connector-python", file=sys.stderr)
    sys.exit(2)


def get_connection(args: argparse.Namespace):
    return mysql.connector.connect(
        host=args.host,
        port=args.port,
        user=args.user,
        password=args.password,
        database=args.database,
        autocommit=False,
    )


def setup_schema(args: argparse.Namespace) -> None:
    conn = get_connection(args)
    try:
        cur = conn.cursor()
        cur.execute(CREATE_TABLE_SQL)
        conn.commit()
    finally:
        conn.close()


def load_part_ids(args: argparse.Namespace, limit: int = 5000) -> list[int]:
    conn = get_connection(args)
    try:
        cur = conn.cursor()
        cur.execute("SELECT part_id FROM m_part ORDER BY part_id LIMIT %s", (limit,))
        return [int(row[0]) for row in cur.fetchall()]
    finally:
        conn.close()


def worker_run(
    worker_id: int,
    args: argparse.Namespace,
    stop_at: float,
    part_ids: list[int],
    stats: SharedStats,
    global_seed: int,
) -> None:
    rng = random.Random(global_seed + worker_id)
    conn = None

    try:
        conn = get_connection(args)
        cur = conn.cursor()

        while time.time() < stop_at:
            part_id = rng.choice(part_ids) if part_ids else None
            tx_tag = f"w{worker_id}-{int(time.time() * 1000)}-{rng.randint(1000, 9999)}"
            qty = rng.randint(1, 500)
            payload = f"load:{tx_tag}"

            try:
                cur.execute(
                    """
                    INSERT INTO t_tx_load(worker_id, tx_tag, part_id, qty, payload)
                    VALUES (%s, %s, %s, %s, %s)
                    """,
                    (worker_id, tx_tag, part_id, qty, payload),
                )

                cur.execute(
                    """
                    UPDATE t_tx_load
                    SET qty = qty + %s
                    WHERE worker_id = %s
                    ORDER BY id DESC
                    LIMIT 1
                    """,
                    (rng.randint(1, 3), worker_id),
                )

                if part_id is not None:
                    cur.execute(
                        """
                        SELECT COUNT(*)
                        FROM t_bom_item bi
                        JOIN t_bom b ON b.bom_id = bi.bom_id
                        WHERE b.parent_part_id = %s
                        """,
                        (part_id,),
                    )
                    cur.fetchone()

                if rng.random() < args.rollback_ratio:
                    conn.rollback()
                    with stats.lock:
                        stats.rollbacks += 1
                else:
                    conn.commit()
                    with stats.lock:
                        stats.commits += 1

                with stats.lock:
                    stats.tx_total += 1

            except Exception as exc:
                conn.rollback()
                with stats.lock:
                    stats.errors += 1
                    stats.tx_total += 1
                if args.verbose_errors:
                    print(f"[worker-{worker_id}] error: {exc}", file=sys.stderr)

            if args.sleep_ms > 0:
                time.sleep(args.sleep_ms / 1000.0)

    except Error as exc:
        with stats.lock:
            stats.errors += 1
        print(f"[worker-{worker_id}] connection error: {exc}", file=sys.stderr)
    finally:
        if conn is not None and conn.is_connected():
            conn.close()


def main() -> int:
    args = parse_args()
    check_dependency_or_exit()

    if args.workers < 1:
        print("--workers must be >= 1", file=sys.stderr)
        return 2
    if args.duration < 1:
        print("--duration must be >= 1", file=sys.stderr)
        return 2
    if not (0.0 <= args.rollback_ratio <= 1.0):
        print("--rollback-ratio must be between 0.0 and 1.0", file=sys.stderr)
        return 2

    if args.create_table:
        setup_schema(args)

    part_ids = load_part_ids(args)
    stats = SharedStats(lock=threading.Lock())
    stop_at = time.time() + args.duration

    threads: list[threading.Thread] = []
    for i in range(args.workers):
        t = threading.Thread(
            target=worker_run,
            args=(i + 1, args, stop_at, part_ids, stats, args.seed),
            daemon=True,
        )
        threads.append(t)
        t.start()

    for t in threads:
        t.join()

    elapsed = args.duration
    tps = stats.tx_total / elapsed if elapsed > 0 else 0.0

    print("Stress test finished")
    print(f"  database   : {args.database}")
    print(f"  workers    : {args.workers}")
    print(f"  duration_s : {args.duration}")
    print(f"  tx_total   : {stats.tx_total}")
    print(f"  commits    : {stats.commits}")
    print(f"  rollbacks  : {stats.rollbacks}")
    print(f"  errors     : {stats.errors}")
    print(f"  approx_tps : {tps:.2f}")

    return 1 if stats.errors > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
