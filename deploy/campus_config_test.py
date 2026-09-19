"""Campus secret configuration regression checks; contains only fake values."""
import unittest
import render_config as rc
class CampusConfigTests(unittest.TestCase):
    def test_disabled(self):
        rc.validate_campus({})
    def test_all_or_nothing_and_origin(self):
        valid = {'CAMPUS_CLIENT_ID': 'fake-client', 'CAMPUS_REDIRECT_URI': 'https://forum.test/api/campus/tongji/callback', 'CAMPUS_ENCRYPTION_KEY': 'e'*32, 'CAMPUS_IDENTITY_KEY': 'i'*32, 'SERVER_URL':'https://forum.test'}
        rc.validate_campus(valid)
        for key,value in [('CAMPUS_ENCRYPTION_KEY','short'), ('CAMPUS_IDENTITY_KEY',''), ('CAMPUS_REDIRECT_URI','https://another.test/api/campus/tongji/callback'), ('CAMPUS_REDIRECT_URI','https://forum.test/api/campus/tongji/callback?next=evil')]:
            with self.subTest(key=key, value=value), self.assertRaises(SystemExit):
                rc.validate_campus(dict(valid, **{key:value}))
if __name__ == '__main__': unittest.main()
