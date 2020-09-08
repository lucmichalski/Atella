#!/bin/bash
clickhouse-client -q "select distinct partition from system.parts where active=1 and database='graphite' and table='data' and max_date < today() - 6;" | while read PART; do clickhouse-client -q "OPTIMIZE TABLE graphite.data PARTITION ('"$PART"') FINAL";done
