#!/usr/bin/env python3

import os
from decimal import Decimal

from flask import Flask, jsonify, request
import mysql.connector
from mysql.connector import Error
import requests
from requests import RequestException


app = Flask(__name__)


def mysql_config():
    return {
        "host": os.environ.get("MYSQL_HOST", "127.0.0.1"),
        "port": int(os.environ.get("MYSQL_PORT", "3306")),
        "user": os.environ.get("MYSQL_USER", "appuser"),
        "password": os.environ.get("MYSQL_PASSWORD", ""),
        "database": os.environ.get("MYSQL_DATABASE", "appdb"),
        "timeout": int(os.environ.get("MYSQL_TIMEOUT", "5")),
    }


def upstream_config():
    return {
        "random_digits_api_url": os.environ.get("RANDOM_DIGITS_API_URL", "http://10.1.1.15:5000/random-5digits"),
        "bom_tree_api_url": os.environ.get("BOM_TREE_API_URL", "http://10.1.1.13:5000/bom-tree"),
        "timeout": int(os.environ.get("UPSTREAM_TIMEOUT", "5")),
        "retry_limit": int(os.environ.get("BOM_TREE_RETRY_LIMIT", "20")),
    }


def mysql_connection(config):
    return mysql.connector.connect(
        host=config["host"],
        port=config["port"],
        user=config["user"],
        password=config["password"],
        database=config["database"],
        connection_timeout=config["timeout"],
    )


def json_safe(value):
    if isinstance(value, Decimal):
        if value == value.to_integral_value():
            return int(value)
        return float(value)
    return value


def fetch_bom_tree(config, part_name):
    query = """
WITH RECURSIVE bom_tree AS (
  SELECT
    1 AS lvl,
    parent.part_id AS root_part_id,
    parent.part_name AS root_part_name,
    child.part_id AS child_part_id,
    child.part_no AS child_part_no,
    child.part_name AS child_part_name,
    bi.qty_per_parent,
    bi.qty_per_parent AS total_qty
  FROM m_part parent
  JOIN t_bom b ON b.parent_part_id = parent.part_id
  JOIN t_bom_item bi ON bi.bom_id = b.bom_id
  JOIN m_part child ON child.part_id = bi.child_part_id
  WHERE parent.part_name = %s

  UNION ALL

  SELECT
    bt.lvl + 1,
    bt.root_part_id,
    bt.root_part_name,
    c2.part_id,
    c2.part_no,
    c2.part_name,
    bi2.qty_per_parent,
    bt.total_qty * bi2.qty_per_parent
  FROM bom_tree bt
  JOIN t_bom b2 ON b2.parent_part_id = bt.child_part_id
  JOIN t_bom_item bi2 ON bi2.bom_id = b2.bom_id
  JOIN m_part c2 ON c2.part_id = bi2.child_part_id
)
SELECT
  lvl,
  child_part_no,
  child_part_name,
  qty_per_parent,
  total_qty
FROM bom_tree
ORDER BY lvl, child_part_no;
"""

    connection = mysql_connection(config)
    try:
        cursor = connection.cursor()
        cursor.execute(query, (part_name,))
        rows = cursor.fetchall()
        columns = [column[0] for column in cursor.description]
        return [
            {key: json_safe(value) for key, value in zip(columns, row)}
            for row in rows
        ]
    finally:
        connection.close()


def fetch_products(config):
    query = """
SELECT DISTINCT
  p.part_no,
  p.part_name
FROM t_bom b
JOIN m_part p
  ON p.part_id = b.parent_part_id
WHERE p.part_name LIKE 'Product-%'
ORDER BY p.part_name;
"""

    connection = mysql_connection(config)
    try:
        cursor = connection.cursor()
        cursor.execute(query)
        rows = cursor.fetchall()
        columns = [column[0] for column in cursor.description]
        return [
            {key: json_safe(value) for key, value in zip(columns, row)}
            for row in rows
        ]
    finally:
        connection.close()


def fetch_random_5digits_from_api(config):
    response = requests.get(config["random_digits_api_url"], timeout=config["timeout"])
    response.raise_for_status()

    payload = response.json()
    if payload.get("status") != "ok":
        raise ValueError("random-5digits API returned non-ok status")

    random_digits = str(payload.get("random_digits", ""))
    if len(random_digits) != 5 or not random_digits.isdigit():
        raise ValueError("random-5digits API returned invalid random_digits")

    return random_digits


def fetch_bom_tree_from_api(config, part_name):
    response = requests.get(
        config["bom_tree_api_url"],
        params={"part_name": part_name},
        timeout=config["timeout"],
    )

    if response.status_code == 404:
        return None

    response.raise_for_status()

    payload = response.json()
    if payload.get("status") != "ok":
        if "notfound" in str(payload.get("message", "")).lower().replace(" ", ""):
            return None
        raise ValueError("bom-tree API returned non-ok status")

    items = payload.get("items", [])
    if not isinstance(items, list):
        raise ValueError("bom-tree API response items is not a list")

    if len(items) == 0:
        return None

    return items


@app.get("/health")
def health():
    return jsonify(status="ok")


@app.get("/db")
def db():
    config = mysql_config()

    try:
        connection = mysql_connection(config)
        cursor = connection.cursor()
        cursor.execute("SELECT 1")
        result = cursor.fetchone()
        cursor.close()
        connection.close()

        return jsonify(
            status="ok",
            mysql={
                "host": config["host"],
                "database": config["database"],
                "select_one": result[0] if result else None,
            },
        )
    except Error as exc:
        return jsonify(
            status="error",
            mysql={
                "host": config["host"],
                "database": config["database"],
                "message": str(exc),
            },
        ), 500


@app.get("/products")
def products():
    config = mysql_config()

    try:
        rows = fetch_products(config)
        return jsonify(
            status="ok",
            items=rows,
        )
    except Error as exc:
        return jsonify(
            status="error",
            message=str(exc),
        ), 500


@app.get("/bom-tree")
def bom_tree():
    part_name = request.args.get("part_name", "").strip()
    if not part_name:
        return jsonify(status="error", message="part_name is required"), 400

    config = mysql_config()

    try:
        rows = fetch_bom_tree(config, part_name)
        return jsonify(
            status="ok",
            part_name=part_name,
            items=rows,
        )
    except Error as exc:
        return jsonify(
            status="error",
            part_name=part_name,
            message=str(exc),
        ), 500


@app.get("/mock-bom")
def mock_bom():
    config = upstream_config()
    retry_limit = config["retry_limit"]

    if retry_limit < 1:
        return jsonify(status="error", message="BOM_TREE_RETRY_LIMIT must be >= 1"), 400

    try:
        for attempt in range(1, retry_limit + 1):
            random_digits = fetch_random_5digits_from_api(config)
            selected_part_name = f"Product-{random_digits}"
            bom_items = fetch_bom_tree_from_api(config, selected_part_name)

            if bom_items is None:
                continue

            return jsonify(
                status="ok",
                random_digits_api_url=config["random_digits_api_url"],
                bom_tree_api_url=config["bom_tree_api_url"],
                selected_random_digits=random_digits,
                requested_bom_part_name=selected_part_name,
                attempt=attempt,
                bom_item_count=len(bom_items),
                bom_items=bom_items,
            )

        return jsonify(
            status="error",
            message="bom-tree not found within retry limit",
            random_digits_api_url=config["random_digits_api_url"],
            bom_tree_api_url=config["bom_tree_api_url"],
            retry_limit=retry_limit,
        ), 404
    except (RequestException, ValueError) as exc:
        return jsonify(status="error", message=str(exc)), 502


if __name__ == "__main__":
    app.run(
        host=os.environ.get("APP_HOST", "0.0.0.0"),
        port=int(os.environ.get("APP_PORT", "5000")),
    )