import importlib.util
import pathlib
import sqlite3
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('feed_privacy', pathlib.Path(__file__).parent / 'scripts/feed-privacy.py')
privacy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(privacy)

class FeedPrivacyTest(unittest.TestCase):
    def test_copy_scrubbing_preserves_live_business_data(self):
        with tempfile.TemporaryDirectory() as tmp:
            source = pathlib.Path(tmp) / 'live.db'
            copy = pathlib.Path(tmp) / 'copy.db'
            with sqlite3.connect(source) as db:
                db.execute('CREATE TABLE topics(id INTEGER PRIMARY KEY, title TEXT, rank_ready INTEGER)')
                db.execute('INSERT INTO topics VALUES(1, "public discussion", 1)')
                for table in privacy.TABLES:
                    db.execute(f'CREATE TABLE {table}(id TEXT, user_id INTEGER)')
                    db.execute(f'INSERT INTO {table} VALUES("identifiable", 9)')
                db.commit()
                with sqlite3.connect(copy) as target:
                    db.backup(target)
            privacy.scrub_copy(copy)
            with sqlite3.connect(copy) as db:
                for table in privacy.TABLES:
                    self.assertEqual(db.execute(f'SELECT count(*) FROM {table}').fetchone()[0], 0)
                self.assertEqual(db.execute('SELECT title,rank_ready FROM topics').fetchone(), ('public discussion',0))
            with sqlite3.connect(source) as db:
                self.assertEqual(db.execute('SELECT count(*) FROM feed_serve_log').fetchone()[0], 1)

if __name__ == '__main__': unittest.main()
