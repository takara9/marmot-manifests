#!/usr/bin/env python3

import os
from decimal import Decimal

from flask import Flask, jsonify, request
import mysql.connector
from mysql.connector import Error


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


if __name__ == "__main__":
    app.run(
        host=os.environ.get("APP_HOST", "0.0.0.0"),
        port=int(os.environ.get("APP_PORT", "5000")),
    )