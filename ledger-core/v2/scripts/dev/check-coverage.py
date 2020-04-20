#!/usr/bin/env python3

import sys
import argparse
import subprocess
import re

parser = argparse.ArgumentParser(description='Check code coverage')
parser.add_argument(
    '-m', '--minimum', metavar='N', type=float, required=True,
    help='minimum code coverage, percents')
parser.add_argument(
    '-f', '--fname', metavar='F', type=str, required=True,
    help='path to golang coverage.out file')
args = parser.parse_args()
fname = args.fname
required = args.minimum

cmd = "go tool cover -func={}".format(fname)
out = subprocess.check_output(cmd, shell=True).decode('utf-8')
m = re.search("""(?si)total:.*?([\d\.)]+)\%$""", out)
coverage = float(m.group(1))

print("Current coverage: {:.2f}, required: {:.2f}".format(coverage, required))

if coverage >= required:
	print("Check passed.")
	sys.exit(0)
else:
	print("CHECK FAILED!")
	sys.exit(1)
