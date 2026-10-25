import base64, hashlib, importlib.util, json, unittest
from pathlib import Path
from unittest.mock import patch

path=Path(__file__).with_name('q43-billing-correlation-reader.py')
spec=importlib.util.spec_from_file_location('billing_reader',path)
q=importlib.util.module_from_spec(spec);spec.loader.exec_module(q)
CANARY='gen-PRIVATE_CORRELATION_CANARY'

class CorrelationTest(unittest.TestCase):
 def test_explicit_or_nested_id(self):
  self.assertEqual(q.correlations({'id':CANARY}),{CANARY})
  self.assertEqual(q.correlations({'error':{'metadata':{'raw':json.dumps({'generation_id':CANARY})}}}),{CANARY})
 def test_no_guess_from_message_or_request(self):
  self.assertEqual(q.correlations({'error':{'message':CANARY,'request_id':CANARY,'id':'req-123456789'}}),set())
 def test_ambiguous_candidates_remain_distinct(self):
  self.assertEqual(len(q.correlations({'id':CANARY,'error':{'generation_id':'gen-OTHER_PRIVATE_ID'}})),2)
 def fixture(self,has_id=False):
  manifest={'source':q.SOURCE}
  envelope={'error':{'code':400,'message':'PRIVATE_ERROR_BODY'}}
  if has_id:envelope['id']=CANARY
  body=json.dumps(envelope).encode()
  body+=b' '*(642-len(body))
  report={'manifest':manifest,'grant':{'approval':q.APPROVAL},'core':{'decision':'INCONCLUSIVE'},'wire':[{'method':'GET'} for _ in range(6)]+[{'method':'POST','path':'/api/v1/chat/completions','status':400,'failure':False,'body':base64.b64encode(body).decode()}]}
  mh=hashlib.sha256(q.canonical(manifest)).hexdigest();rh=hashlib.sha256(q.canonical(report)).hexdigest()
  marker=(q.SOURCE+' '+mh+' '+q.APPROVAL+'\n').encode()
  sealed=json.dumps({'report':report,'sha256':rh}).encode()
  files={'document-reading-spec0056-q3.started':marker,'document-reading-spec0056-q3.private.json':sealed}
  constants={'MANIFEST':mh,'REPORT':rh,'MARKER_FILE_SHA':hashlib.sha256(marker).hexdigest(),'REPORT_FILE_SHA':hashlib.sha256(sealed).hexdigest(),'BODY_SHA':hashlib.sha256(body).hexdigest()}
  return files,constants
 def inspect(self,has_id=False,tamper=False,unstable=False):
  files,constants=self.fixture(has_id)
  if tamper:files['document-reading-spec0056-q3.private.json']+=b' '
  reads={}
  def snap(name,limit):
   self.assertIn(name,files);raw=files[name];self.assertLessEqual(len(raw),limit)
   reads[name]=reads.get(name,0)+1
   return raw,(reads[name] if unstable else 1,),hashlib.sha256(raw).hexdigest()
  with patch.multiple(q,**constants),patch.object(q,'snap',snap):return q.run()
 def test_absent_means_stop_without_side_effects(self):
  result=self.inspect()
  self.assertEqual(result['reason'],'generation_correlation_absent')
  self.assertEqual(result['status'],'UNVERIFIED')
  for key in ('metadataGET','sqlCalls','keyReads','persistentWritesRequested','paidPOST'):self.assertEqual(result[key],0)
  self.assertFalse(result['resumeAllowed']);self.assertFalse(result['settlementApplied'])
 def test_present_still_requires_billing_verification_and_redaction(self):
  result=self.inspect(has_id=True)
  self.assertTrue(result['correlationUnique']);self.assertFalse(result['resumeAllowed'])
  self.assertNotIn(CANARY,json.dumps(result));self.assertNotIn('PRIVATE_ERROR_BODY',json.dumps(result))
 def test_tampered_seal_refused(self):
  with self.assertRaises(q.Refusal):self.inspect(tamper=True)
 def test_changed_snapshot_refused(self):
  with self.assertRaises(q.Refusal):self.inspect(unstable=True)
 def test_duplicate_fields_refused(self):
  with self.assertRaises(q.Refusal):q.parse(b'{"id":"one","ID":"two"}')

if __name__=='__main__':unittest.main()
