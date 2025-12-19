#!/bin/bash
set -e


source ./perf_test/test_create_post.sh
source ./perf_test/test_read_post.sh
pgbadger ./pglogs/postgresql-*.log -o ./pgbadger_report.html


