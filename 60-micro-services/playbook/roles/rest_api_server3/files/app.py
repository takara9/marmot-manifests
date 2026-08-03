#!/usr/bin/env python3

import os
import random
import time
import json
import logging
from datetime import datetime, timezone

from flask import Flask, jsonify, request, g


app = Flask(__name__)

logging.basicConfig(
    level=getattr(logging, os.environ.get("APP_LOG_LEVEL", "INFO").upper(), logging.INFO),
    format="%(message)s",
)


def parse_response_delay_seconds():
    raw_value = os.environ.get("RESPONSE_DELAY_SECONDS", "0")

    try:
        delay_seconds = float(raw_value)
    except ValueError:
        raise ValueError("RESPONSE_DELAY_SECONDS must be a number between 0 and 10")

    if delay_seconds < 0 or delay_seconds > 10:
        raise ValueError("RESPONSE_DELAY_SECONDS must be between 0 and 10")

    return delay_seconds


@app.before_request
def start_request_timer():
    g.request_start_time = time.perf_counter()


@app.after_request
def log_access(response):
    started = getattr(g, "request_start_time", None)
    duration_ms = (time.perf_counter() - started) * 1000 if started is not None else -1

    access_log = {
        "ts": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "level": "INFO",
        "event": "http_access",
        "method": request.method,
        "path": request.path,
        "status": response.status_code,
        "duration_ms": round(duration_ms, 2),
        "remote_addr": request.headers.get("X-Forwarded-For", request.remote_addr),
        "user_agent": request.user_agent.string,
    }
    app.logger.info(json.dumps(access_log, separators=(",", ":")))
    return response


@app.get("/health")
def health():
    return jsonify(status="ok")


@app.get("/random-5digits")
def random_5digits():
    try:
        delay_seconds = parse_response_delay_seconds()
    except ValueError as exc:
        return jsonify(status="error", message=str(exc)), 400

    time.sleep(delay_seconds)
    random_digits = f"{random.randint(0, 99999):05d}"

    return jsonify(
        status="ok",
        random_digits=random_digits,
        response_delay_seconds=delay_seconds,
    )


if __name__ == "__main__":
    app.run(
        host=os.environ.get("APP_HOST", "0.0.0.0"),
        port=int(os.environ.get("APP_PORT", "5000")),
    )