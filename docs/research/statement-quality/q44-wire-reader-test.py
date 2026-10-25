import base64,hashlib,importlib.util,json,unittest
from pathlib import Path
from unittest.mock import patch
path=Path(__file__).with_name('q44-wire-reader.py');spec=importlib.util.spec_from_file_location('q44',path);q=importlib.util.module_from_spec(spec);spec.loader.exec_module(q)
CANARY='PRIVATE_ERROR_LITERAL_CANARY'
class ReaderTest(unittest.TestCase):
 def test_nested_google_error_redacted(self):
  raw=json.dumps({'error':{'code':400,'status':'INVALID_ARGUMENT','message':'Request contains an invalid argument. '+CANARY,'details':[{'fieldViolations':[{'field':'generation_config.response_schema.properties.'+CANARY,'description':CANARY}]}]}})
  d=q.diagnostic({'error':{'message':'Provider returned error','metadata':{'raw':raw}}})
  self.assertEqual(d['providerStatusCodes'],['INVALID_ARGUMENT']);self.assertTrue(d['genericInvalidArgumentMessage']);self.assertTrue(d['providerReturnedErrorMessage']);self.assertNotIn(CANARY,json.dumps(d));self.assertEqual(d['fieldViolationPaths'][0][-1],'other')
 def test_generation_id_not_exported(self):
  value='gen-PRIVATE_CORRELATION_CANARY';d=q.diagnostic({'id':value});self.assertTrue(d['generationCorrelationPresent']);self.assertNotIn(value,json.dumps(d))
 def call(self,unstable=False,tampered=False):
  manifest={'source':q.SOURCE};mh=hashlib.sha256(q.canonical(manifest)).hexdigest()
  value={'error':{'code':400,'message':CANARY}}
  r={'manifest':manifest,'grant':{'approval':q.APPROVAL},'core':{'decision':'INCONCLUSIVE'},'wire':[{'method':'GET'} for _ in range(6)]+[{'method':'POST','path':'/api/v1/chat/completions','status':400,'failure':False,'body':base64.b64encode(json.dumps(value).encode()).decode()}],'budget':[{'kind':'reserve','failure':False,'amount':31389}]}
  rh=hashlib.sha256(q.canonical(r)).hexdigest()
  files={'document-reading-spec0056-q4.started':(q.SOURCE+' '+mh+' '+q.APPROVAL+'\n').encode(),'document-reading-spec0056-q4.private.json':json.dumps({'sha256':rh,'report':r}).encode()}
  if tampered:files['document-reading-spec0056-q4.started']=b'other'
  counts={}
  def snap(name,limit):
   self.assertIn(name,files);raw=files[name];counts[name]=counts.get(name,0)+1;return raw,(counts[name] if unstable else 1,),hashlib.sha256(raw).hexdigest()
  with patch.multiple(q,MANIFEST=mh,REPORT=rh),patch.object(q,'snap',snap):return q.run()
 def test_fixed_projection(self):
  result=self.call();self.assertNotIn(CANARY,json.dumps(result));self.assertEqual(result['successfulReserveMicros'],[31389]);self.assertFalse(result['usageObjectPresent']);self.assertEqual(result['providerCalls'],0);self.assertEqual(result['sqlCalls'],0)
 def test_changed_snapshot_refused(self):
  with self.assertRaises(q.Refusal):self.call(unstable=True)
 def test_marker_mismatch_refused(self):
  with self.assertRaises(q.Refusal):self.call(tampered=True)
if __name__=='__main__':unittest.main()
