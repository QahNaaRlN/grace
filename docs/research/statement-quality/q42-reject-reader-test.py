"""Offline controls for fixed-snapshot admission and literal-free projection."""
import base64, importlib.util, json, unittest
from pathlib import Path
from unittest.mock import patch
p=Path(__file__).with_name('q42-reject-reader.py')
spec=importlib.util.spec_from_file_location('reader',p);q=importlib.util.module_from_spec(spec);spec.loader.exec_module(q)
CANARY='PRIVATE_REPLY_CANARY_DO_NOT_EMIT'
def visual():
 def quote(): return {'page':1,'quote':CANARY,'box':[100,200,300,400]}
 def field(): return {'value':CANARY,'evidence':[quote()]}
 v={'header':{k:field() for k in q.HEAD},'rows':[]};v['header']['availableBalanceObserved']=None
 for _ in range(2):v['rows'].append({'anchor':[quote()],'fields':{k:field() for k in q.ROW},'issues':[]})
 for k in ('transactionAmount','transactionCurrency'):v['rows'][1]['fields'][k]['evidence']=[]
 return v
def snapshot(v):
 manifest={'source':q.SOURCE};mh=q.hashlib.sha256(q.canonical(manifest)).hexdigest()
 response={'choices':[{'finish_reason':'stop','message':{'content':json.dumps(v)}}]}
 wire=[{'method':'GET'} for _ in range(6)]+[{'method':'POST','path':'/api/v1/chat/completions','status':200,'failure':False,'body':base64.b64encode(json.dumps(response).encode()).decode()}]
 r={'manifest':manifest,'grant':{'source':q.SOURCE,'approval':q.APPROVAL,'manifestSHA256':mh},'core':{'decision':'REJECT','results':[{'case':'cards','state':'rejected','settlement':'settled'}]+[{'state':'not_sent'} for _ in range(7)]},'wire':wire}
 rh=q.hashlib.sha256(q.canonical(r)).hexdigest()
 data={'document-reading-spec0056-q2.started':(q.SOURCE+' '+mh+' '+q.APPROVAL+'\n').encode(),'document-reading-spec0056-q2.private.json':json.dumps({'sha256':rh,'report':r}).encode()}
 return mh,rh,data
class ReaderTest(unittest.TestCase):
 def run_snapshot(self,v,mutate=None):
  mh,rh,data=snapshot(v)
  if mutate:mutate(data)
  def read(name,limit):
   self.assertIn(name,data);raw=data[name];self.assertLessEqual(len(raw),limit)
   return raw,('unchanged',),q.hashlib.sha256(raw).hexdigest()
  with patch.object(q,'MANIFEST',mh),patch.object(q,'REPORT',rh),patch.object(q,'snap',read):return q.diagnosis()
 def test_actual_classes_and_counterfactuals(self):
  out=self.run_snapshot(visual());c=out['content']
  self.assertNotIn(CANARY,json.dumps(out));self.assertTrue(out['snapshotUnchanged'])
  self.assertEqual(c['valueViolationCount'],25);self.assertEqual(c['boxesAboveOne'],23)
  self.assertEqual(c['counterfactualScaleOneThousand']['valueViolationCount'],2)
  self.assertTrue(c['counterfactualScaleAndNullUncited']['visualValidationFits'])
 def test_unknown_model_keys_never_escape(self):
  v=visual();v['rows'][0][CANARY]=CANARY;out=self.run_snapshot(v)
  self.assertNotIn(CANARY,json.dumps(out));self.assertFalse(out['content']['directJSONShapeFits'])
 def test_duplicate_and_markdown_refused(self):
  self.assertFalse(q.inspect('{"rows":[],"ROWS":[]}')['jsonValidUnique'])
  self.assertFalse(q.inspect('```'+CANARY)['jsonValidUnique'])
 def test_wrong_marker_refused(self):
  with self.assertRaises(q.Refusal):self.run_snapshot(visual(),lambda d:d.update({'document-reading-spec0056-q2.started':b'changed'}))
 def test_wrong_seal_refused(self):
  def mutate(d):
   name='document-reading-spec0056-q2.private.json';v=json.loads(d[name]);v['sha256']='0'*64;d[name]=json.dumps(v).encode()
  with self.assertRaises(q.Refusal):self.run_snapshot(visual(),mutate)
 def test_changing_snapshot_refused(self):
  mh,rh,data=snapshot(visual());counts={}
  def read(name,limit):
   counts[name]=counts.get(name,0)+1;return data[name],(counts[name],),'0'*64
  with patch.object(q,'MANIFEST',mh),patch.object(q,'REPORT',rh),patch.object(q,'snap',read):
   with self.assertRaises(q.Refusal):q.diagnosis()
if __name__=='__main__':unittest.main()
