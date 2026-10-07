#!/usr/bin/env python3
"""Rotate deployment-owned Agent stream identity before reopening a restored DB."""
import argparse
import json
import os
from pathlib import Path
import secrets

parser = argparse.ArgumentParser()
parser.add_argument('state_directory', type=Path)
parser.add_argument('--isolate', action='store_true', help='Disable Agent ingress, production and egress after a copied snapshot')
args = parser.parse_args()
args.state_directory.mkdir(parents=True, exist_ok=True)
path = args.state_directory / 'state.json'
state = json.loads(path.read_text()) if path.exists() else {
    'instanceId': 'inst_' + secrets.token_hex(16), 'apiEnabled': True,
    'producerEnabled': False, 'webhookEnabled': False,
}
state['streamEpoch'] = secrets.token_hex(16)
if args.isolate:
    state.update(apiEnabled=False, producerEnabled=False, webhookEnabled=False)
temporary = path.with_suffix('.json.tmp')
temporary.write_text(json.dumps(state) + '\n')
os.chmod(temporary, 0o644)
temporary.replace(path)
