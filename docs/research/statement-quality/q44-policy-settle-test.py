import copy,hashlib,importlib.util,json,os,re,subprocess,tempfile,unittest
from pathlib import Path
from unittest.mock import patch
path=Path(__file__).with_name('q44-policy-settle.py')
spec=importlib.util.spec_from_file_location('settle',path);q=importlib.util.module_from_spec(spec);spec.loader.exec_module(q)
UID='11111111-1111-4111-8111-111111111111'
OWNER='22222222-2222-4222-8222-222222222222'
OTHER='33333333-3333-4333-8333-333333333333'
def row():return {'id':UID,'owner_id':OWNER,'model':'google/gemini-3.1-flash-lite','provider':'google-vertex/eu','purpose':'statement','reserved_micros':32023,'status':'uncertain','actual_micros':None,'generation_id':None,'created_at':'2026-10-25T19:15:20+00:00','settled_at':'2026-10-25T19:15:23+00:00'}
def settled(original):
 value=dict(original);value.update(status='settled',actual_micros=0,generation_id=q.SETTLEMENT_MARKER,settled_at='2026-10-25T19:00:00+00:00');return value
class OperatorTest(unittest.TestCase):
 def simulate(self,change=None,fail_snapshot=False,failed_apply=False,replay=False,conflict=False):
  original=row();state=dict(original);applies=[]
  with tempfile.TemporaryDirectory(prefix='q44-u-test-') as tmp:
   root=Path(tmp)
   def save(name,value):
    p=root/name;p.write_bytes(value);p.chmod(0o600);return hashlib.sha256(value).hexdigest()
   marker_sha=save('document-reading-spec0056-q4.started',b'marker')
   report_sha=save('document-reading-spec0056-q4.private.json',json.dumps({'report':{'budget':[{'kind':'reserve','failure':False,'id':UID,'purpose':'statement','amount':32023}]}}).encode())
   save('statement-screening.lock',b'')
   def command(args,stdin=None):
    self.assertEqual(stdin,q.select_row(UID));self.assertNotIn('api',args);return json.dumps(state)
   def apply(args,before,stable):
    applies.append(1)
    if fail_snapshot:save('document-reading-spec0056-q4.started',b'changed')
    stable()
    if failed_apply:raise q.Stop('sql_precommit_refused')
    state.update(settled(before));return dict(state)
   with patch.multiple(q,ROOT=tmp,MARKER_FILE_SHA=marker_sha,REPORT_FILE_SHA=report_sha),patch.object(q,'fixed_projection'),patch.object(q,'container',return_value='0'*12),patch.object(q,'command',command),patch.object(q,'apply_sql',apply):
    if change:state.update(change)
    if change:
     with self.assertRaises(q.Stop):q.run()
     self.assertEqual(applies,[]);return
    result=q.run()
    if replay or conflict:
     if conflict:
      state['actual_micros']=1
      with self.assertRaises(q.Stop):q.run()
     else:
      again=q.run();self.assertTrue(again['idempotentReplay']);self.assertEqual(len(applies),1)
   audit=json.loads((root/q.U_AUDIT).read_text());self.assertEqual((root/q.U_AUDIT).stat().st_mode&0o777,0o600)
   public=json.dumps(result)
   for canary in (UID,OWNER,OTHER,q.SETTLEMENT_MARKER):self.assertNotIn(canary,public)
   self.assertEqual(result['metadataGET'],0);self.assertEqual(result['paidPOST'],0);self.assertFalse(result['directMetadataVerified'])
   self.assertEqual(audit['usageBefore'],original)
   return result,audit,state
 def test_exact_one_row_private_policy_basis(self):
  result,audit,state=self.simulate();self.assertTrue(result['settlementApplied']);self.assertTrue(result['resumeAllowed']);self.assertEqual(result['contingencyMicros'],32023);self.assertEqual(audit['billingBasis'],'owner_accepted_http400_policy');self.assertEqual(state['owner_id'],OWNER)
 def test_invalid_owner_before_apply(self):self.simulate(change={'owner_id':'invalid'})
 def test_changed_cost_before_apply(self):self.simulate(change={'actual_micros':1})
 def test_changed_snapshot_before_commit(self):
  result,audit,state=self.simulate(fail_snapshot=True);self.assertFalse(result['resumeAllowed']);self.assertEqual(state['status'],'uncertain');self.assertFalse(audit['committed'])
 def test_failed_sql_never_resumes(self):
  result,audit,state=self.simulate(failed_apply=True);self.assertFalse(result['resumeAllowed']);self.assertEqual(state['status'],'uncertain');self.assertFalse(audit['committed'])
 def test_idempotent_replay(self):self.simulate(replay=True)
 def test_conflicting_replay(self):self.simulate(conflict=True)
 def test_injection_and_no_network_capability(self):
  with self.assertRaises(q.Stop):q.select_row("' OR true--")
  source=path.read_text();self.assertNotIn('urllib',source);self.assertNotIn('requests.',source);self.assertNotIn('/run/secrets',source)

 def test_policy_signature_refuses_other_billing_shapes(self):
  good={'error':{'code':400,'metadata':{'raw':json.dumps({'error':{'status':'INVALID_ARGUMENT'}})}}}
  q.policy_signature(good)
  for field in ('usage','cost','choices','generation_id'):
   bad=copy.deepcopy(good);bad[field]=None
   with self.assertRaises(q.Refusal):q.policy_signature(bad)
  for bad in ({'error':{'code':500}}, {'error':{'code':400,'status':'INTERNAL'}},dict(good,id='gen-12345678')):
   with self.assertRaises(q.Refusal):q.policy_signature(bad)

CONTAINER=os.environ.get('GRACE_V_TEST_CONTAINER','')
@unittest.skipUnless(re.fullmatch(r'grace-v-test-[a-z0-9]+',CONTAINER),'isolated SQL fixture not requested')
class SQLTest(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.args=q.sql_args(CONTAINER)
  def sql(text):return q.command(cls.args,text)
  cls.sql=staticmethod(sql)
  repo=path.resolve().parents[3]
  bootstrap="CREATE ROLE grace_financial NOLOGIN; CREATE TABLE users(id uuid PRIMARY KEY); CREATE FUNCTION grace_owner_id() RETURNS uuid LANGUAGE sql STABLE AS $$ SELECT nullif(current_setting('grace.owner_id',true),'')::uuid $$;"
  sql(bootstrap+(repo/'migrations/000013_ai_experiment_usage.up.sql').read_text()+"ALTER TABLE ai_usage DROP CONSTRAINT ai_usage_purpose_check; ALTER TABLE ai_usage ADD CONSTRAINT ai_usage_purpose_check CHECK(purpose IN ('analysis','receipt','jev_category','statement'));" )
  sql("INSERT INTO users VALUES('"+OWNER+"'),('"+OTHER+"');")
 def setUp(self):
  self.sql('TRUNCATE ai_usage;')
  self.sql("INSERT INTO ai_usage(id,owner_id,reserved_micros,status,model,provider,purpose,created_at,settled_at) VALUES('"+UID+"','"+OWNER+"',32023,'uncertain','google/gemini-3.1-flash-lite','google-vertex/eu','statement','2026-10-25T19:15:20Z','2026-10-25T19:15:23Z'),('44444444-4444-4444-8444-444444444444','"+OTHER+"',32023,'uncertain','google/gemini-3.1-flash-lite','google-vertex/eu','statement','2026-10-25T19:15:20Z','2026-10-25T19:15:23Z');")
  self.original=json.loads(self.sql(q.select_row(UID)))
  self.all_before=self.sql('SELECT json_agg(u ORDER BY id) FROM ai_usage u;')
  self.budget_before=self.sql('SELECT row_to_json(b) FROM ai_experiment_budget b;')
 def test_real_transaction_and_neighbor_budget_unchanged(self):
  q.apply_sql(self.args,self.original,lambda:None)
  current=json.loads(self.sql(q.select_row(UID)));q.after(current,self.original)
  before=json.loads(self.all_before);after=json.loads(self.sql('SELECT json_agg(u ORDER BY id) FROM ai_usage u;'))
  self.assertEqual(before[1],after[1]);self.assertEqual(self.budget_before,self.sql('SELECT row_to_json(b) FROM ai_experiment_budget b;'))
  replay="BEGIN;SELECT set_config('grace.owner_id','"+OWNER+"',true);SET LOCAL ROLE grace_financial;SELECT grace_ai_settle('"+UID+"',0,'"+q.SETTLEMENT_MARKER+"');COMMIT;"
  self.sql(replay);self.assertEqual(current,json.loads(self.sql(q.select_row(UID))))
  for changed in (replay.replace("',0,'","',1,'"),replay.replace("set_config('grace.owner_id','"+OWNER,"set_config('grace.owner_id','"+OTHER)):
   with self.assertRaises(q.Stop):self.sql(changed)
   self.assertEqual(current,json.loads(self.sql(q.select_row(UID))))
 def test_stale_owner_guard_rolls_back(self):
  changed=dict(self.original,owner_id=OTHER)
  with self.assertRaises(q.Stop):q.apply_sql(self.args,changed,lambda:None)
  self.assertEqual(self.all_before,self.sql('SELECT json_agg(u ORDER BY id) FROM ai_usage u;'))
 def test_snapshot_check_rolls_back_before_commit(self):
  def fail():raise q.Stop('snapshot_changed')
  with self.assertRaises(q.Stop):q.apply_sql(self.args,self.original,fail)
  self.assertEqual(self.all_before,self.sql('SELECT json_agg(u ORDER BY id) FROM ai_usage u;'))
if __name__=='__main__':unittest.main()
