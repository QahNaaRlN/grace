import base64,hashlib,importlib.util,json,unittest
from pathlib import Path
from unittest.mock import patch
p=Path(__file__).with_name('q43-wire-reader.py');spec=importlib.util.spec_from_file_location('q',p);q=importlib.util.module_from_spec(spec);spec.loader.exec_module(q)
CANARY='PRIVATE_PROVIDER_INPUT_CANARY'
def make():
 m={'source':q.SOURCE};mh=hashlib.sha256(q.canonical(m)).hexdigest()
 body={'error':{'code':400,'message':CANARY,'metadata':{'raw':json.dumps({'code':'invalid_type','path':['response_format',CANARY],'expected':'object','received':'string','message':CANARY})}}}
 r={'manifest':m,'grant':{'approval':q.APPROVAL},'core':{'decision':'INCONCLUSIVE'},'wire':[{'method':'GET'} for _ in range(6)]+[{'method':'POST','path':'/api/v1/chat/completions','status':400,'failure':False,'body':base64.b64encode(json.dumps(body).encode()).decode()}],'budget':[{'kind':'reserve','amount':30329,'failure':False}]}
 rh=hashlib.sha256(q.canonical(r)).hexdigest()
 files={'document-reading-spec0056-q3.started':(q.SOURCE+' '+mh+' '+q.APPROVAL+'\n').encode(),'document-reading-spec0056-q3.private.json':json.dumps({'sha256':rh,'report':r}).encode()}
 return mh,rh,files
class ReaderTest(unittest.TestCase):
 def call(self,change=None,unstable=False):
  mh,rh,files=make()
  if change:change(files)
  counts={}
  def read(name,limit):
   self.assertIn(name,files);b=files[name];self.assertLessEqual(len(b),limit);counts[name]=counts.get(name,0)+1
   return b,(counts[name] if unstable else 1,),hashlib.sha256(b).hexdigest()
  with patch.object(q,'MANIFEST',mh),patch.object(q,'REPORT',rh),patch.object(q,'snap',read):return q.run()
 def test_fixed_projection(self):
  d=self.call();self.assertNotIn(CANARY,json.dumps(d));self.assertEqual(d['postHTTPStatus'],400);self.assertEqual(d['successfulReserveMicros'],[30329]);self.assertFalse(d['costFieldPresent']);self.assertFalse(d['explicitNumericZeroCost']);self.assertEqual(d['structuredValidationDetails'][0]['path'],['response_format','other'])
 def test_marker_mismatch(self):
  with self.assertRaises(q.Refusal):self.call(lambda f:f.update({'document-reading-spec0056-q3.started':b'wrong'}))
 def test_report_mismatch(self):
  def change(f):
   name='document-reading-spec0056-q3.private.json';v=json.loads(f[name]);v['sha256']='0'*64;f[name]=json.dumps(v).encode()
  with self.assertRaises(q.Refusal):self.call(change)
 def test_changing_snapshot(self):
  with self.assertRaises(q.Refusal):self.call(unstable=True)
if __name__=='__main__':unittest.main()
