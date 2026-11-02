"""Selected artifacts determine required jobs, never a hard-coded pair of apps."""
import json
import os
import sys
from model import require


def verify(stage, artifacts, results):
    mobile = bool(set(artifacts) & {'android', 'ios'})
    expected = {'begin': 'success'}
    if stage == 'distribution':
        expected['identity'] = 'success' if mobile else 'skipped'
        expected.update({job: 'success' if artifact in artifacts else 'skipped'
                         for artifact, job in [('web', 'dev'), ('android', 'android'), ('ios', 'ios')]})
    for name, result in expected.items():
        require(results.get(name, {}).get('result') == result, f'Preview {name} needs {result}')


if __name__ == '__main__':
    verify(sys.argv[1], json.loads(os.environ['ARTIFACTS']), json.loads(os.environ['RESULTS']))
