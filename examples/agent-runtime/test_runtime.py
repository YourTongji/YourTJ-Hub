import base64
import hashlib
import hmac
import json
import io
from pathlib import Path
import tempfile
import threading
import time
import unittest
from http.server import ThreadingHTTPServer
from unittest.mock import patch
from urllib.error import HTTPError
from urllib.request import Request, build_opener, ProxyHandler
from datetime import datetime,timezone
from runtime import Forum,Inbox,consume,daily_key,poll,serve,synergy_config,verify

class RuntimeTests(unittest.TestCase):
    def test_broadcast_page_persists_both_types_and_cursor(self):
        events=[{'id':'evt_'+str(i),'instanceId':'i','agentId':1,'schemaVersion':1,'type':kind}
                for i,kind in enumerate(['agent.mentioned','forum.topic_created','forum.post_created'])]
        class ForumStub:
            def request(self,*args): return {'events':events,'nextCursor':'broadcast_cursor','hasMore':False}
        with tempfile.TemporaryDirectory() as folder:
            box=Inbox(str(Path(folder)/'inbox.db'))
            poll(ForumStub(),box,'i',1)
            self.assertEqual(box.pending(),events)
            self.assertEqual(box.cursor(),'broadcast_cursor')
            events[1]['type']='forum.unknown'
            with self.assertRaises(ValueError): poll(ForumStub(),box,'i',1)
            self.assertEqual(box.cursor(),'broadcast_cursor')

    def test_signed_broadcast_webhook_persists_and_unknown_type_is_rejected(self):
        secret='whsec_'+base64.b64encode(b'x'*32).decode()
        opener=build_opener(ProxyHandler({}))
        with tempfile.TemporaryDirectory() as folder:
            box=Inbox(str(Path(folder)/'inbox.db'))
            with patch('runtime.ThreadingHTTPServer') as factory:
                serve(box,'i',1,[secret],0)
            handler=factory.call_args.args[1]
            with ThreadingHTTPServer(('127.0.0.1',0),handler) as server:
                thread=threading.Thread(target=server.serve_forever,daemon=True)
                thread.start()
                try:
                    for i,kind in enumerate(['forum.topic_created','forum.post_created','forum.unknown']):
                        event={'id':'evt_'+str(i),'instanceId':'i','agentId':1,'schemaVersion':1,'type':kind}
                        raw=json.dumps(event).encode()
                        stamp=str(int(time.time()))
                        signature=base64.b64encode(hmac.digest(b'x'*32,event['id'].encode()+b'.'+stamp.encode()+b'.'+raw,'sha256')).decode()
                        request=Request(f'http://127.0.0.1:{server.server_port}/events',data=raw,headers={
                            'Content-Type':'application/json','Webhook-Id':event['id'],
                            'Webhook-Timestamp':stamp,'Webhook-Signature':'v1,'+signature})
                        if kind=='forum.unknown':
                            with self.assertRaises(HTTPError) as error: opener.open(request,timeout=3)
                            self.assertEqual(error.exception.code,400)
                        else:
                            with opener.open(request,timeout=3) as response: self.assertEqual(response.status,204)
                    self.assertEqual([e['type'] for e in box.pending()],['forum.topic_created','forum.post_created'])
                finally:
                    server.shutdown()
                    thread.join(timeout=3)

    def test_write_response_keeps_pending_review_envelope(self):
        class Opener:
            def open(self,*args,**kwargs):
                return io.BytesIO(json.dumps({'code':0,'result':{'id':4},'messageCode':'content.moderation.pendingReview'}).encode())
        forum=Forum('https://forum.example','agt_example');forum.opener=Opener()
        response=forum.request('POST','/api/v1/agent/topics/2/posts',{'content':'A reply'})
        self.assertEqual(response,{'code':0,'result':{'id':4},'messageCode':'content.moderation.pendingReview'})
    def test_raw_signature_rotation_tampering_and_timestamp(self):
        raw=b'{"id":"evt_1", "schemaVersion":1}'
        secret='whsec_'+base64.b64encode(b'x'*32).decode()
        signature=base64.b64encode(hmac.digest(b'x'*32,b'evt_1.1000.'+raw,'sha256')).decode()
        headers={'Webhook-Id':'evt_1','Webhook-Timestamp':'1000','Webhook-Signature':'v1,invalid v1,'+signature}
        self.assertEqual(verify(raw,headers,[secret],now=1000),'evt_1')
        for changed in [json.dumps(json.loads(raw)).encode(),raw+b' ']:
            with self.assertRaises(ValueError): verify(changed,headers,[secret],now=1000)
        with self.assertRaises(ValueError): verify(raw,headers,[secret],now=1400)
    def test_ingress_and_cursor_survive_restart_without_duplicate_processing(self):
        with tempfile.TemporaryDirectory() as folder:
            path=str(Path(folder)/'inbox.db');box=Inbox(path)
            e={'id':'evt_1'};box.store([e],'cursor_1');box.store([e],'cursor_1')
            box=Inbox(path)
            self.assertEqual(box.cursor(),'cursor_1');self.assertEqual(box.pending(),[e])
            box.complete('evt_1',{'id':456})
            self.assertEqual(box.pending(),[e]) # ACK not durable yet
            self.assertEqual(box.result('evt_1'),{'id':456})
            box.acknowledge('evt_1');self.assertEqual(Inbox(path).pending(),[])
    def test_cursor_does_not_advance_when_storage_fails(self):
        class ForumStub:
            def request(self,*args): return {'events':[{'id':'evt_1','instanceId':'i','agentId':1,'schemaVersion':1,'type':'agent.mentioned'}],'nextCursor':'new','hasMore':False}
        class BrokenInbox:
            def cursor(self): return 'old'
            def store(self,*args): raise RuntimeError('storage failure')
        with self.assertRaises(RuntimeError): poll(ForumStub(),BrokenInbox(),'i',1)
    def test_response_loss_uses_source_key_and_ack_retry_keeps_result(self):
        class ForumStub:
            def __init__(self): self.calls=[];self.acks=0
            def request(self,method,path,body=None,key=None):
                self.calls.append((method,path,body,key))
                if path=='/api/v1/agent/events/evt_1': return {'state':'active','data':{'topicId':2,'postId':3}}
                if path=='/api/v1/agent/events/ack':
                    self.acks+=1
                    if self.acks==1: raise RuntimeError('ACK response lost')
                return {'id':4}
        with tempfile.TemporaryDirectory() as folder:
            box=Inbox(str(Path(folder)/'inbox.db'));box.store([{'id':'evt_1'}]);forum=ForumStub()
            with self.assertRaises(RuntimeError): consume(forum,box,'evt_1','A reply')
            consume(forum,box,'evt_1','A reply')
            writes=[call for call in forum.calls if call[0]=='POST' and call[1].endswith('/posts')]
            self.assertEqual(len(writes),1)
            self.assertEqual(writes[0][2]['sourceEventId'],'evt_1');self.assertEqual(writes[0][3],'reply:evt_1')
            self.assertEqual(box.pending(),[])
    def test_withdrawn_event_finishes_locally_without_forum_ack(self):
        class ForumStub:
            def __init__(self): self.calls=[]
            def request(self,method,path,body=None,key=None):
                self.calls.append((method,path))
                if method!='GET': raise RuntimeError('withdrawn event cannot be ACKed')
                return {'state':'withdrawn'}
        with tempfile.TemporaryDirectory() as folder:
            box=Inbox(str(Path(folder)/'inbox.db'));box.store([{'id':'evt_1'}]);forum=ForumStub()
            result=consume(forum,box,'evt_1','Do not post this')
            self.assertEqual(result,{'skipped':True,'state':'withdrawn'})
            self.assertEqual(box.pending(),[])
            self.assertEqual(forum.calls,[('GET','/api/v1/agent/events/evt_1')])
    def test_shanghai_daily_key_and_synergy_wire_config(self):
        now=datetime(2026,10,3,17,tzinfo=timezone.utc)
        self.assertEqual(daily_key('morning',now),'daily:morning:2026-10-04')
        config=synergy_config('https://forum.example/mcp','agt_example')['mcp']['yourtj']
        self.assertEqual(config['type'],'remote');self.assertFalse(config['oauth'])
        self.assertIn('list_events',config['toolFilter']['include']);self.assertEqual(config['headers']['Authorization'],'Bearer agt_example')
        with self.assertRaises(ValueError): synergy_config('http://unsafe/mcp','agt_example')

if __name__=='__main__': unittest.main()
