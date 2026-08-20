#!/usr/bin/env python3
"""Generate and decode Tornado signed cookie values.

Used by the Go cross-stack integration tests.
"""

from __future__ import annotations

import argparse
import sys
from collections.abc import Callable

import tornado.web


def fixed_clock(timestamp: float) -> Callable[[], float]:
    """Return a Tornado clock function fixed at timestamp."""
    return lambda: timestamp


def build_parser() -> argparse.ArgumentParser:
    """Build the command-line parser used by the integration test."""
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    generate = subparsers.add_parser("generate")
    generate.add_argument("--secret", required=True)
    generate.add_argument("--name", required=True)
    generate.add_argument("--value", required=True)
    generate.add_argument("--version", required=True, type=int)
    generate.add_argument("--now", required=True, type=float)

    decode = subparsers.add_parser("decode")
    decode.add_argument("--secret", required=True)
    decode.add_argument("--name", required=True)
    decode.add_argument("--value", required=True)
    decode.add_argument("--max-age-days", required=True, type=float)
    decode.add_argument("--now", required=True, type=float)

    return parser


def run(args: argparse.Namespace) -> int:
    """Execute one signed-cookie operation."""
    clock = fixed_clock(args.now)
    if args.command == "generate":
        signed_value = tornado.web.create_signed_value(
            args.secret,
            args.name,
            args.value,
            version=args.version,
            clock=clock,
        )
        print(signed_value.decode("ascii"))
        return 0

    decoded = tornado.web.decode_signed_value(
        args.secret,
        args.name,
        args.value,
        max_age_days=args.max_age_days,
        clock=clock,
    )
    if decoded is None:
        return 1
    print(decoded.decode("utf-8"))
    return 0


def main() -> int:
    """Parse arguments and run the requested operation."""
    return run(build_parser().parse_args())


if __name__ == "__main__":
    sys.exit(main())
