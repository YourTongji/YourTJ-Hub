"""A release is not successful when a selected platform publisher was skipped."""
import json
import os
from model import FILES, require


def verify(channels, requested, results):
    require(channels and all(channel in FILES for channel in channels), 'Invalid approved channels')
    selected = 'android' if requested == 'android-alias' else requested
    require(not selected or selected in channels, 'Requested channel is outside the approved request')
    publishers = {'ios' if channel.startswith('ios-') else channel for channel in
                  ([selected] if selected else channels)}
    for publisher in sorted(publishers):
        require(results.get(publisher, {}).get('result') == 'success',
                f'Publisher {publisher} did not succeed; inspect its jobs before resuming')


if __name__ == '__main__':
    verify(json.loads(os.environ['CHANNELS']), os.environ.get('REQUESTED_CHANNEL', ''),
           json.loads(os.environ['RESULTS']))
