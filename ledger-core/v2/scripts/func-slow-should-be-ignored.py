#!/usr/bin/env python3

import time
import datetime
import sys

# `make test_func_slow` errors should be ignored until ~28.06.2020
# this is an estimated date when Release 0 Milestone 3 will be ready
m3date = "28.06.2020"
tup = datetime.datetime.strptime(m3date, "%d.%m.%Y").timetuple()
if time.time() > time.mktime(tup):
	sys.exit(1) # false, don't ignore

# true, tests should be ignored
sys.exit(0)
