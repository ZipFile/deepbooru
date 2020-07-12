#!/usr/bin/env python

import json
import re
import signal
import sys
from urllib.parse import urlparse


def error(msg):
    print(json.dumps({"error": msg}))


def result(tags):
    print(json.dumps({"tags": [
        {
            "name": tag,
            "score": score
        }
        for tag, score in tags.items()
    ]}))


def get_ext(path):
    try:
        return path.rsplit(".", 1)[1]
    except IndexError:
        return "none"


def main():
    alive = True

    def handle_sigterm(_, __):
        nonlocal alive
        alive = False

    signal.signal(signal.SIGTERM, handle_sigterm)

    while alive:
        try:
            line = input()
        except (EOFError, BrokenPipeError):
            break
        except KeyboardInterrupt:
            continue

        if line == "":
            break

        try:
            data = json.loads(line)
        except json.JSONDecodeError:
            error("invalid json")
            continue

        if "shutdown" in data:
            break

        url = data.get("url", "")

        if not url:
            error("missing url")
            continue

        if re.match("^https?://.*", url) is None:
            error("invalid url")
            continue

        scheme, netloc, path, params, query, fragment = urlparse(url)

        result({
            "scheme:" + scheme: 1,
            "host:" + netloc: 1,
            "ext:" + get_ext(path): 1,
        })


if __name__ == "__main__":
    sys.exit(main())
