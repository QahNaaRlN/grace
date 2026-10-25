import datetime as dt, decimal, hashlib, importlib.util, json, tempfile, unittest
from pathlib import Path
from unittest.mock import patch

path=Path(__file__).with_name('q43-billing-metadata-reader.py')
spec=importlib.util.spec_from_file_location('metadata_reader',path)
q=importlib.util.module_from_spec(spec);spec.loader.exec_module(q)
ID='gen-PRIVATE_OWNER_ACTIVITY_CANARY'
UID='11111111-1111-4111-8111-111111111111'
OWNER='22222222-2222-4222-8222-222222222222'
def data(**changes):
 value={'id':ID,'model':q.MODEL,'is_byok':False,'created_at':q.START,'finish_reason':'error','total_cost':0,'provider_name':'Google','tokens_prompt':0,'tokens_completion':0}
 value.update(changes);return value
class MetadataTest(unittest.TestCase):
 def test_exact_identity_time_error_cost(self):
  self.assertEqual(q.validate_metadata(data(),ID),0)
  for changes in ({'id':'gen-OTHER_PRIVATE_CANARY'},{'model':'other'},{'is_byok':True},{'created_at':'2026-10-25T11:59:32Z'},{'finish_reason':'stop'},{'total_cost':None},{'total_cost':'0'},{'total_cost':False},{'total_cost':decimal.Decimal('0.03033')},{'total_cost':decimal.Decimal('-1')},{'total_cost':decimal.Decimal('NaN')}):
   with self.subTest(changes=changes),self.assertRaises(q.Stop):q.validate_metadata(data(**changes),ID)
 def test_ceil_and_canonical(self):
  self.assertEqual(q.validate_metadata(data(model=q.CANONICAL_MODEL,total_cost=decimal.Decimal('0.00000100000000000000000000000000001')),ID),2)
 def test_missing_provider_requires_error_zero_tokens_zero_cost(self):
  self.assertEqual(q.validate_metadata(data(provider_name=None),ID),0)
  for changes in ({'tokens_prompt':1},{'tokens_completion':None},{'native_tokens_reasoning':1},{'total_cost':decimal.Decimal('0.000001')},{'provider_name':'other'}):
   with self.subTest(changes=changes),self.assertRaises(q.Stop):q.validate_metadata(data(provider_name=None,**{k:v for k,v in changes.items() if k!='provider_name'}) if 'provider_name' not in changes else data(**changes),ID)
 def simulate(self,metadata=None,row_change=None,existing=False,failed_get=False,changed_snapshot=False):
  row={'id':UID,'owner_id':OWNER,'model':q.MODEL,'provider':'google-vertex/eu','purpose':'statement','reserved_micros':30329,'actual_micros':None,'generation_id':None,'status':'uncertain','created_at':q.START}
  if row_change:row.update(row_change)
  report={'budget':[{'kind':'reserve','failure':False,'id':UID,'purpose':'statement','amount':30329}]}
  raw=json.dumps({'report':report}).encode();digest=hashlib.sha256(raw).hexdigest();snapshots=0
  def snap(name,limit):
   nonlocal snapshots
   self.assertEqual(name,'document-reading-spec0056-q3.private.json');snapshots+=1
   return raw,(snapshots if changed_snapshot else 1,),digest
  queries=[];gets=[]
  def command(args,stdin=None):
   if args[-2:]==['cat','/run/secrets/openrouter_api_key']:return 'PRIVATE_KEY_CANARY'
   self.assertIn('psql',args);self.assertEqual(stdin,"SELECT row_to_json(u) FROM ai_usage u WHERE id='"+UID+"'::uuid;");queries.append(stdin);return json.dumps(row)
  def get(ident,key):
   self.assertEqual((ident,key),(ID,'PRIVATE_KEY_CANARY'));gets.append(ident)
   if failed_get:raise q.Stop('metadata_http_refused')
   return json.dumps({'data':metadata if metadata is not None else data()}).encode()
  with tempfile.TemporaryDirectory(prefix='q43-metadata-test-') as tmp:
   root=Path(tmp);(root/'statement-screening.lock').touch(mode=0o600)
   if existing:(root/q.AUDIT).write_text('consumed')
   with patch.multiple(q,ROOT=tmp,REPORT_FILE_SHA=digest),patch.object(q,'fixed_projection',return_value={}),patch.object(q,'snap',snap),patch.object(q,'container',side_effect=lambda service:service),patch.object(q,'command',command),patch.object(q,'get_metadata',get):
    if row_change or existing:
     with self.assertRaises((q.Stop,FileExistsError)):q.inspect(ID)
     self.assertEqual(len(gets),0);return None
    result=q.inspect(ID)
   audit=json.loads((root/q.AUDIT).read_text())
   self.assertEqual((root/q.AUDIT).stat().st_mode&0o777,0o600)
   self.assertEqual(len(gets),1);self.assertFalse(result['settlementApplied']);self.assertFalse(result['resumeAllowed'])
   self.assertEqual(len(queries),2 if result['status']=='VERIFIED' else 1)
   public=json.dumps(result)
   for canary in (ID,UID,OWNER,'PRIVATE_KEY_CANARY'):self.assertNotIn(canary,public)
   self.assertEqual(audit['usageBefore']['id'],UID);self.assertTrue(audit['metadataGETConsumed'])
   return result
 def test_verified_audit_is_private_no_apply(self):self.assertEqual(self.simulate()['status'],'VERIFIED')
 def test_wrong_generation_time_stops_without_apply(self):self.assertEqual(self.simulate(metadata=data(created_at='2026-10-25T11:59:32Z'))['reason'],'generation_time_mismatch')
 def test_http_failure_is_not_zero_bill(self):self.assertEqual(self.simulate(failed_get=True)['status'],'UNVERIFIED')
 def test_wrong_row_refuses_before_get(self):self.simulate(row_change={'owner_id':'bad'})
 def test_existing_audit_forbids_second_get(self):self.simulate(existing=True)
 def test_changed_report_after_get_refuses(self):self.assertEqual(self.simulate(changed_snapshot=True)['reason'],'snapshot_changed')
if __name__=='__main__':unittest.main()
