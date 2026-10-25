import base64,hashlib,importlib.util,json,unittest,copy
from pathlib import Path
from unittest.mock import patch
path=Path(__file__).with_name('q45-quality-reader.py');spec=importlib.util.spec_from_file_location('q45',path);q=importlib.util.module_from_spec(spec);spec.loader.exec_module(q)
CANARY='PRIVATE_MODEL_VALUE_CANARY'
def answer():return {'rows':[{'fields':{k:{'value':v,'evidence':[{'quote':v}]} for k,v in row.items()},'issues':[]} for row in q.EXPECTED]}
class ReaderTest(unittest.TestCase):
 def test_date_precision_and_privacy(self):
  value=answer();value['rows'][0]['fields']['operationDate']['value']='19.10.2026';value['rows'][1]['fields']['description']['value']=CANARY;value['rows'][1]['issues']=[CANARY+' prompt injection']
  d=q.diagnostic(value);self.assertFalse(d[0]['dates'][0]['hasClock']);self.assertTrue(d[0]['dates'][0]['sameCalendarDate']);self.assertFalse(d[0]['dates'][0]['sameHourMinute']);self.assertTrue(d[0]['dates'][0]['quotesContainExpectedPrintedValue']);self.assertTrue(d[1]['issueVocabulary']['inject']);self.assertNotIn(CANARY,json.dumps(d))
 def test_added_seconds_separate_from_wrong_clock(self):
  value=answer();value['rows'][0]['fields']['operationDate']['value']+=':00';d=q.diagnostic(value)[0]['dates'][0];self.assertTrue(d['hasSeconds']);self.assertTrue(d['sameHourMinute']);self.assertFalse(d['exactNormalizedMatch'])
 def call(self,unstable=False,tampered=False):
  manifest={'source':q.SOURCE};mh=hashlib.sha256(q.canonical(manifest)).hexdigest()
  value={'choices':[{'message':{'content':json.dumps(answer())}}]}
  r={'manifest':manifest,'grant':{'approval':q.APPROVAL},'core':{'decision':'REJECT','results':[{'measurement':{'protocol':True,'recognition':False}}]},'wire':[{'method':'GET'} for _ in range(6)]+[{'method':'POST','path':'/api/v1/chat/completions','status':200,'failure':False,'body':base64.b64encode(json.dumps(value).encode()).decode()}],'budget':[{'kind':'reserve','failure':False,'amount':31389}]}
  rh=hashlib.sha256(q.canonical(r)).hexdigest()
  files={'document-reading-spec0056-q5.started':(q.SOURCE+' '+mh+' '+q.APPROVAL+'\n').encode(),'document-reading-spec0056-q5.private.json':json.dumps({'sha256':rh,'report':r}).encode()}
  if tampered:files['document-reading-spec0056-q5.started']=b'other'
  counts={}
  def snap(name,limit):
   self.assertIn(name,files);raw=files[name];counts[name]=counts.get(name,0)+1;return raw,(counts[name] if unstable else 1,),hashlib.sha256(raw).hexdigest()
  with patch.multiple(q,MANIFEST=mh,REPORT=rh),patch.object(q,'snap',snap):return q.run()
 def test_fixed_projection(self):
  result=self.call();self.assertNotIn(CANARY,json.dumps(result));self.assertEqual(result['providerCalls'],0);self.assertEqual(result['sqlCalls'],0)
 def test_changed_snapshot_refused(self):
  with self.assertRaises(q.Refusal):self.call(unstable=True)
 def test_marker_mismatch_refused(self):
  with self.assertRaises(q.Refusal):self.call(tampered=True)
if __name__=='__main__':unittest.main()
