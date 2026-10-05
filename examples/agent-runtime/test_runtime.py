import base64
import hashlib
import hmac
import json
import io
from pathlib import Path
import tempfile
import unittest
from datetime import datetime,timezone
from runtime import Forum,Inbox,consume,daily_key,poll,synergy_config,verify

class RuntimeTests(unittest.TestCase):
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
