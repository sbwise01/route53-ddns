#!/usr/bin/env python3
"""Print the semver wrapper version from a VERSION file.

Usage: version.py [path-to-version-file]   (defaults to ./VERSION)
"""
import re
import sys

path = sys.argv[1] if len(sys.argv) > 1 else "VERSION"
text = open(path, encoding="utf-8").read().strip()
if not text:
    sys.exit(f"empty VERSION file: {path}")
if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", text):
    sys.exit(f"VERSION must be semver X.Y.Z, got {text!r} in {path}")
print(text)
