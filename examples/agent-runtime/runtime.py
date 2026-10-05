#!/usr/bin/env python3
"""Durable Agent ingress and deterministic REST writes; models run externally.

Signature protocol: docs/decisions/0065-agent-interaction-events.md.
Never acknowledge before durable ingress, nor process webhook test messages.
"""
import argparse
import base64
import hashlib
import hmac
import json
import os
from pathlib import Path
import sqlite3
import time
from datetime import datetime
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.error import HTTPError
from urllib.parse import urlencode, urlsplit
from urllib.request import Request, build_opener, HTTPRedirectHandler, ProxyHandler
from zoneinfo import ZoneInfo

MAX_BODY = 16384
RETENTION = 7 * 86400

class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

class Inbox:
    def __init__(self, path):
        self.path = path
        with self.connect() as db:
            db.executescript('CREATE TABLE IF NOT EXISTS inbox (id TEXT PRIMARY KEY, body TEXT NOT NULL, received INTEGER NOT NULL, done INTEGER NOT NULL DEFAULT 0, acked INTEGER NOT NULL DEFAULT 0, result TEXT); CREATE TABLE IF NOT EXISTS state (key TEXT PRIMARY KEY, value TEXT NOT NULL);')
    def connect(self):
        db = sqlite3.connect(self.path, timeout=10)
        db.execute('PRAGMA synchronous=FULL')
        return db
    def store(self, events, cursor=None):
        with self.connect() as db:
            db.execute('DELETE FROM inbox WHERE acked=1 AND received<?', (int(time.time())-RETENTION,))
            for event in events:
                if db.execute('SELECT count(*) FROM inbox').fetchone()[0] >= 10000 and not db.execute('SELECT 1 FROM inbox WHERE id=?',(event['id'],)).fetchone():
                    raise ValueError('local inbox capacity; reconcile before accepting more events')
                db.execute('INSERT OR IGNORE INTO inbox(id,body,received) VALUES(?,?,?)',(event['id'],json.dumps(event),int(time.time())))
            if cursor is not None:
                db.execute("INSERT OR REPLACE INTO state VALUES('cursor',?)", (cursor,))
    def cursor(self):
        with self.connect() as db:
            row=db.execute("SELECT value FROM state WHERE key='cursor'").fetchone()
            return row[0] if row else ''
    def complete(self, event_id, result):
        with self.connect() as db:
            db.execute('UPDATE inbox SET done=1,result=? WHERE id=?',(json.dumps(result),event_id))
    def acknowledge(self, event_id):
        with self.connect() as db:
            db.execute('UPDATE inbox SET acked=1 WHERE id=?',(event_id,))
    def result(self, event_id):
        with self.connect() as db:
            row=db.execute('SELECT done,result FROM inbox WHERE id=?',(event_id,)).fetchone()
            return json.loads(row[1]) if row and row[0] else None
    def pending(self):
        with self.connect() as db:
            return [json.loads(row[0]) for row in db.execute('SELECT body FROM inbox WHERE acked=0 ORDER BY received,id LIMIT 100')]

def verify(raw, headers, secrets, now=None):
    """Standard Webhooks HMAC over exact bytes; accept either rotation key."""
    event_id=headers.get('Webhook-Id','')
    timestamp=headers.get('Webhook-Timestamp','')
    if not event_id or len(event_id)>80 or not timestamp.isdigit() or abs((now or time.time())-int(timestamp))>300:
        raise ValueError('invalid identity or signature timestamp')
    signed=event_id.encode()+b'.'+timestamp.encode()+b'.'+raw
    candidates=[part[3:] for part in headers.get('Webhook-Signature','').split() if part.startswith('v1,')]
    for secret in secrets:
        if not secret.startswith('whsec_'):
            continue
        key=base64.b64decode(secret[6:],validate=True)
        if len(key)<32:
            continue
        expected=base64.b64encode(hmac.digest(key,signed,'sha256')).decode()
        if any(hmac.compare_digest(expected,part) for part in candidates):
            return event_id
    raise ValueError('signature rejected')

def validate_event(event, instance_id, agent_id):
    if event.get('instanceId')!=instance_id or event.get('agentId')!=agent_id or event.get('schemaVersion')!=1:
        raise ValueError('foreign instance/Agent or unsupported schema')
    if event.get('type') not in ('agent.mentioned','agent.post_replied','agent.topic_commented',
                                'forum.topic_created','forum.post_created','agent.webhook_test'):
        raise ValueError('unsupported event type')

class Forum:
    def __init__(self, base, token):
        parsed=urlsplit(base)
        if parsed.scheme!='https' or not parsed.netloc or parsed.username or parsed.password or parsed.query or parsed.fragment:
            raise ValueError('FORUM_URL must be a credential-free HTTPS origin')
        self.base=base.rstrip('/')
        self.token=token
        self.opener=build_opener(NoRedirect,ProxyHandler({}))
    def request(self, method, path, body=None, key=None):
        headers={'Authorization':'Bearer '+self.token,'Accept':'application/json'}
        if key: headers['Idempotency-Key']=key
        data=None
        if body is not None:
            data=json.dumps(body).encode();headers['Content-Type']='application/json'
        req=Request(self.base+path,data=data,headers=headers,method=method)
        try:
            with self.opener.open(req,timeout=30) as response:
                raw=response.read(4*1024*1024+1)
        except HTTPError as error:
            # Do not print arbitrary server bodies (they can contain secrets).
            raise RuntimeError('forum HTTP failure '+str(error.code)) from None
        if len(raw)>4*1024*1024: raise ValueError('forum response too large')
        payload=json.loads(raw)
        if payload.get('code')!=0:
            raise RuntimeError('forum rejected: '+payload.get('messageCode','unknown'))
        if method=='POST' and (path=='/api/v1/agent/topics' or path.endswith('/posts')):
            # Creation succeeded even when moderation has not made it public.
            return payload
        return payload['result']

def poll(forum, inbox, instance_id, agent_id, reset=False):
    # Reset is a deliberate operator action after cursor_reset/cursor_expired.
    cursor='' if reset else inbox.cursor()
    while True:
        page=forum.request('GET','/api/v1/agent/events?'+urlencode({'after':cursor,'limit':100}))
        for event in page['events']: validate_event(event,instance_id,agent_id)
        inbox.store(page['events'],page['nextCursor'])
        cursor=page['nextCursor']
        if not page['hasMore']: return

def consume(forum, inbox, event_id, content=None):
    # Fresh projection protects runners from acting on a stale pushed snapshot.
    current=forum.request('GET','/api/v1/agent/events/'+event_id)
    if current.get('state') in ('withdrawn','expired'):
        # Revoked sources refuse forum ACK; terminate only the local work item.
        result={'skipped':True,'state':current['state']}
        inbox.complete(event_id,result)
        inbox.acknowledge(event_id)
        return result
    stored=inbox.result(event_id)
    result=stored if stored is not None else {'skipped':True}
    if stored is None and current.get('state')=='active' and content is not None:
        data=current['data']
        forum.request('GET',f"/api/v1/agent/topics/{data['topicId']}/posts?"+urlencode({'anchorPostId':data['postId'],'limit':10}))
        result=forum.request('POST',f"/api/v1/agent/topics/{data['topicId']}/posts",{'content':content,'replyToPostId':data['postId'],'sourceEventId':event_id},'reply:'+event_id)
    # Record the result before ACK: a crash repeats only ACK or a deduped write.
    inbox.complete(event_id,result)
    forum.request('POST','/api/v1/agent/events/ack',{'eventIds':[event_id]})
    inbox.acknowledge(event_id)
    return result

def daily_key(job, now=None):
    date=(now or datetime.now(ZoneInfo('Asia/Shanghai'))).astimezone(ZoneInfo('Asia/Shanghai')).date().isoformat()
    return 'daily:'+job+':'+date

def synergy_config(url, token):
    parsed=urlsplit(url)
    if parsed.scheme!='https' or parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError('MCP_URL must use HTTPS without URL credentials')
    return {'mcp':{'yourtj':{'type':'remote','url':url,'oauth':False,'headers':{'Authorization':'Bearer '+token},'enabled':True,'toolFilter':{'include':['me','list_events','get_event','ack_events','list_topics','get_posts','search','create_post','create_topic']},'tools':{'approval':'always'},'callTimeout':60000}}}

def serve(inbox, instance_id, agent_id, secrets, port):
    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *args): pass
        def do_POST(self):
            try:
                if self.path!='/events' or self.headers.get('Content-Type','').split(';')[0]!='application/json':
                    self.send_error(400);return
                length=int(self.headers.get('Content-Length','0'))
                if length<=0 or length>MAX_BODY:
                    self.send_error(413);return
                self.connection.settimeout(5)
                raw=self.rfile.read(length)
                if len(raw)!=length: raise ValueError('truncated body')
                event_id=verify(raw,self.headers,secrets)
                event=json.loads(raw)
                if event.get('id')!=event_id: raise ValueError('body/header identity mismatch')
                validate_event(event,instance_id,agent_id)
                if event['type']!='agent.webhook_test': inbox.store([event])
                self.send_response(204);self.end_headers()
            except (ValueError,KeyError,TypeError):
                self.send_error(400)
            except Exception:
                self.send_error(503)
    # Terminate HTTPS on your external receiver host; loopback is never a forum endpoint.
    server=ThreadingHTTPServer(('127.0.0.1',port),Handler)
    server.daemon_threads=True
    server.serve_forever()

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action',choices=['serve','poll','pending','reply','skip','daily','synergy-config'])
    parser.add_argument('--db',default='agent-inbox.sqlite');parser.add_argument('--event-id');parser.add_argument('--content-file');parser.add_argument('--title');parser.add_argument('--category',type=int);parser.add_argument('--job-id',default='daily');parser.add_argument('--reset',action='store_true');parser.add_argument('--port',type=int,default=8081);parser.add_argument('--output')
    args=parser.parse_args()
    if args.action=='synergy-config':
        if not args.output: parser.error('--output required')
        content=synergy_config(os.environ['MCP_URL'],os.environ['AGENT_TOKEN'])
        # New owner-only fragment, never overwrite a user's existing Synergy config.
        fd=os.open(args.output,os.O_CREAT|os.O_EXCL|os.O_WRONLY,0o600)
        with os.fdopen(fd,'w') as out: json.dump(content,out,indent=2);out.write('\n')
        return
    inbox=Inbox(args.db)
    if args.action=='pending':
        print(json.dumps(inbox.pending(),ensure_ascii=False,indent=2));return
    instance=os.environ['AGENT_INSTANCE_ID'];agent=int(os.environ['AGENT_ID'])
    if args.action=='serve':
        serve(inbox,instance,agent,os.environ['WEBHOOK_SECRETS'].split(),args.port);return
    forum=Forum(os.environ['FORUM_URL'],os.environ['AGENT_TOKEN'])
    if args.action=='poll': poll(forum,inbox,instance,agent,args.reset)
    elif args.action in ('reply','skip'):
        if not args.event_id: parser.error('--event-id required')
        content=Path(args.content_file).read_text() if args.action=='reply' and args.content_file else None
        if args.action=='reply' and content is None: parser.error('--content-file required')
        consume(forum,inbox,args.event_id,content)
    elif args.action=='daily':
        if not args.content_file or not args.title or not args.category: parser.error('--content-file, --title, --category required')
        response=forum.request('POST','/api/v1/agent/topics',{'title':args.title,'content':Path(args.content_file).read_text(),'categoryId':[args.category]},daily_key(args.job_id))
        print(json.dumps(response,ensure_ascii=False))

if __name__=='__main__': main()
