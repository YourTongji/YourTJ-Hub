#!/usr/bin/env python3
"""Scrub a database COPY before backup/clone publication; never pass a live DB."""
import argparse
import json
import pathlib
import sqlite3

TABLES = json.loads((pathlib.Path(__file__).with_name("feed-raw-tables.json")).read_text())

def scrub_copy(path):
    with sqlite3.connect(path) as db:
        existing = {r[0] for r in db.execute("SELECT name FROM sqlite_master WHERE type='table'")}
        for table in TABLES:
            if table in existing:
                db.execute(f'DELETE FROM "{table}"')
        if 'feed_state' in existing:
            db.execute('DELETE FROM feed_state')
        if 'topic_rank_schedule' in existing:
            db.execute('DELETE FROM topic_rank_schedule')
        if 'topics' in existing:
            columns = {r[1] for r in db.execute('PRAGMA table_info(topics)')}
            if 'rank_ready' in columns:
                db.execute('UPDATE topics SET rank_ready = 0')
    # Remove recoverable raw pages from the COPY before it leaves this process.
    with sqlite3.connect(path) as db:
        db.execute('VACUUM')

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--pg-flags', action='store_true')
    parser.add_argument('--sqlite-copy')
    args = parser.parse_args()
    if args.pg_flags:
        for table in TABLES + ['feed_state', 'topic_rank_schedule']:
            print(f'--exclude-table-data=public.{table}')
    elif args.sqlite_copy:
        scrub_copy(args.sqlite_copy)
    else:
        parser.error('choose --pg-flags or --sqlite-copy')
