#!/usr/bin/env python3

import os
import random
import time

from flask import Flask, jsonify


app = Flask(__name__)


def parse_response_delay_seconds():
    raw_value = os.environ.get("RESPONSE_DELAY_SECONDS", "0")

    try:
        delay_seconds = float(raw_value)
    except ValueError:
        raise ValueError("RESPONSE_DELAY_SECONDS must be a number between 0 and 10")

    if delay_seconds < 0 or delay_seconds > 10:
        raise ValueError("RESPONSE_DELAY_SECONDS must be between 0 and 10")

    return delay_seconds


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