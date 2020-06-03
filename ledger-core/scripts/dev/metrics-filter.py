#!/usr/bin/env python3

#  Copyright 2020 Insolar Network Ltd.
#  All rights reserved.
#  This material is licensed under the Insolar License version 1.0,
#  available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

import csv
import sys
seen = {}

with open(sys.argv[1], 'r') if len(sys.argv) > 1 else sys.stdin as f:
    csv_in = csv.DictReader(f)
    csv_out = csv.DictWriter(
        sys.stdout,
        fieldnames=csv_in.fieldnames,
        delimiter=',', lineterminator='\n',
        )
    csv_out.writeheader()
    for row in csv_in:
        if seen.get(row['name']):
            continue
        seen[row['name']] = True
        row['description'] = row['description'].rstrip()
        csv_out.writerow(row)
