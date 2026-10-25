import os,sys,json,stat,hashlib,base64,re
ROOT='/opt/grace/shared'
SOURCE='a8752028463d19351c9eddcb0ee40b71fe398b47'
MANIFEST='1639957c4398c15733643b315b7a185c82fc9fdfad2c981517c1cf2ad012f3fc'
APPROVAL='1c4e940a90ed1b4586d0dea14d16312a4c060a6b'
REPORT='21d4d71e413bc3f8631d369db1ed344e7f9fc6f13c210cc58a80f08a2a3b7963'
class Refusal(Exception):pass
def need(v):
 if not v:raise Refusal()
def unique(pairs):
 out={};seen=set()
 for k,v in pairs:
  need(k.lower() not in seen);seen.add(k.lower());out[k]=v
 return out
def parse(raw):return json.loads(raw.decode(),object_pairs_hook=unique,parse_constant=lambda _:(_ for _ in ()).throw(Refusal()))
def canonical(v):return json.dumps(v,ensure_ascii=False,separators=(',',':')).replace('&','\\u0026').replace('<','\\u003c').replace('>','\\u003e').replace('\u2028','\\u2028').replace('\u2029','\\u2029').encode()
def snap(name,limit):
 fd=os.open(ROOT+'/'+name,os.O_RDONLY|os.O_NOFOLLOW|os.O_NOATIME|os.O_CLOEXEC|os.O_NONBLOCK)
 try:
  a=os.fstat(fd);need(stat.S_ISREG(a.st_mode) and stat.S_IMODE(a.st_mode)==0o600 and 0<a.st_size<=limit)
  raw=b''
  while len(raw)<=limit:
   p=os.read(fd,min(65536,limit+1-len(raw)))
   if not p:break
   raw+=p
  attrs=lambda s:(s.st_dev,s.st_ino,s.st_size,s.st_mode,s.st_mtime_ns,s.st_ctime_ns)
  need(attrs(a)==attrs(os.fstat(fd)) and len(raw)==a.st_size)
  return raw,attrs(a),hashlib.sha256(raw).hexdigest()
 finally:os.close(fd)
# Fixed Q4.4 identity, accepted SPEC-0056 V.
MARKER_FILE_SHA='42e4e23a7c70d18cac6dd870b52d540a6041ce3d48e7941d2ab559e449c726df'
REPORT_FILE_SHA='cb657a28762be9c9338c5f7871172d8f0a7d91470de22ebbba4d2098b070842d'
BODY_SHA='c876b2c8e82268919c4060e1384c37f0a94af976a28f6039bc8cdb67dc71c13e'
GENERATION=re.compile(r'gen-[A-Za-z0-9_-]{8,196}\Z')
def correlations(value):
 found=set()
 def walk(v,depth=0):
  need(depth<=20)
  if type(v) is dict:
   need(len(v)<=100)
   for key,item in v.items():
    if key in ('id','generation_id','generationId') and type(item) is str and GENERATION.fullmatch(item):found.add(item)
    if type(item) in (dict,list):walk(item,depth+1)
    elif key=='raw' and type(item) is str:
     try:nested=parse(item.encode())
     except (ValueError,UnicodeError,Refusal):continue
     if type(nested) in (dict,list):walk(nested,depth+1)
  elif type(v) is list:
   need(len(v)<=100)
   for item in v:walk(item,depth+1)
 walk(value)
 return found

def policy_signature(envelope):
 need(type(envelope) is dict and type(envelope.get('error')) is dict and envelope['error'].get('code')==400)
 statuses=[]
 def walk(v,depth=0):
  need(depth<=24)
  if type(v) is dict:
   for key,value in v.items():
    need(key.lower() not in ('usage','cost','choices','generation_id','generationid'))
    if key=='status':statuses.append(value)
    if type(value) in (dict,list):walk(value,depth+1)
    elif type(value) is str and value.lstrip().startswith(('{','[')):
     try:nested=parse(value.encode())
     except (ValueError,UnicodeError,Refusal):continue
     walk(nested,depth+1)
  elif type(v) is list:
   for value in v:walk(value,depth+1)
 walk(envelope)
 need('INVALID_ARGUMENT' in statuses and not correlations(envelope))

def fixed_projection():
 marker=snap('document-reading-spec0056-q4.started',512)
 need(marker[0]==(SOURCE+' '+MANIFEST+' '+APPROVAL+'\n').encode() and marker[2]==MARKER_FILE_SHA)
 sealed=snap('document-reading-spec0056-q4.private.json',32<<20)
 need(sealed[2]==REPORT_FILE_SHA)
 outer=parse(sealed[0]);report=outer['report']
 need(outer['sha256']==REPORT and hashlib.sha256(canonical(report)).hexdigest()==REPORT)
 need(report['manifest']['source']==SOURCE and hashlib.sha256(canonical(report['manifest'])).hexdigest()==MANIFEST)
 need(report['grant']['approval']==APPROVAL and report['core']['decision']=='INCONCLUSIVE')
 wire=report['wire'];posts=[item for item in wire if item['method']=='POST']
 need(len(posts)==1 and sum(item['method']=='GET' for item in wire)==6)
 event=posts[0]
 need(event['path']=='/api/v1/chat/completions' and event['status']==400 and event['failure'] is False)
 body=base64.b64decode(event['body'],validate=True)
 need(len(body)==641 and hashlib.sha256(body).hexdigest()==BODY_SHA)
 envelope=parse(body)
 need(type(envelope) is dict and type(envelope.get('error')) is dict)
 policy_signature(envelope)
 need(snap('document-reading-spec0056-q4.started',512)==marker and snap('document-reading-spec0056-q4.private.json',32<<20)==sealed)
# Accepted SPEC-0056 V. No HTTP client, key reader, env dump or financial query.
import datetime as dt,fcntl,selectors,subprocess,time,uuid
ACCEPTANCE='6165d8139e8068bc6d1f5e585abbe236c860f920'
U_AUDIT='document-reading-spec0056-q4.policy-settlement.json'
BASIS='owner_accepted_http400_policy'
SETTLEMENT_MARKER='owner-policy-http400:'+REPORT
class Stop(Exception):pass
def check(ok,reason):
 if not ok:raise Stop(reason)
def valid_uuid(value):
 try:return type(value) is str and str(uuid.UUID(value))==value
 except (ValueError,AttributeError):return False
def timestamp(value):
 try:stamp=dt.datetime.fromisoformat(value.replace('Z','+00:00'))
 except (ValueError,AttributeError):raise Stop('timestamp_invalid')
 check(stamp.tzinfo is not None,'timestamp_invalid')
 return stamp.astimezone(dt.timezone.utc)
def identity(row,uid):
 check(row.get('id')==uid and valid_uuid(uid) and valid_uuid(row.get('owner_id')),'usage_identity_mismatch')
 check(row.get('purpose')=='statement' and row.get('model')=='google/gemini-3.1-flash-lite' and row.get('provider')=='google-vertex/eu' and row.get('reserved_micros')==32023,'usage_policy_mismatch')
 check(timestamp('2026-10-25T19:15:05Z')<=timestamp(row.get('created_at'))<=timestamp('2026-10-25T19:16:01Z'),'usage_time_mismatch')
def before(row,uid,expected):
 identity(row,uid)
 check(row==expected and row.get('status')=='uncertain' and row.get('actual_micros') is None and row.get('generation_id') is None,'usage_changed')
def after(row,original):
 identity(row,original['id'])
 check(row.get('status')=='settled' and row.get('actual_micros')==0 and row.get('generation_id')==SETTLEMENT_MARKER and row.get('settled_at') is not None,'settlement_not_verified')
 check({k:v for k,v in row.items() if k not in ('status','actual_micros','generation_id','settled_at')}=={k:v for k,v in original.items() if k not in ('status','actual_micros','generation_id','settled_at')},'settlement_identity_changed')
def reserve_id(report):
 items=[e for e in report['budget'] if e['kind']=='reserve' and not e['failure']]
 check(len(items)==1,'reservation_not_unique');e=items[0]
 check(valid_uuid(e.get('id')) and e.get('purpose')=='statement' and e.get('amount')==32023,'reservation_mismatch')
 return e['id']
def literal(value):
 check(type(value) is str and '\x00' not in value and len(value)<=65536,'sql_literal_refused')
 return "'"+value.replace("'","''")+"'"
def select_row(uid):
 check(valid_uuid(uid),'usage_identity_mismatch')
 return 'SELECT row_to_json(u) FROM ai_usage u WHERE id='+literal(uid)+'::uuid;'
def transaction(row):
 identity(row,row['id']);uid=literal(row['id']);owner=literal(row['owner_id']);marker=literal(SETTLEMENT_MARKER)
 expected=literal(json.dumps(row,separators=(',',':')))
 immutable={k:v for k,v in row.items() if k not in ('status','actual_micros','generation_id','settled_at')}
 expected_immutable=literal(json.dumps(immutable,separators=(',',':')))
 return "BEGIN; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='12s'; SET LOCAL idle_in_transaction_session_timeout='15s';\n"+\
 'SELECT singleton FROM ai_experiment_budget WHERE singleton FOR UPDATE;\n'+\
 'SELECT id FROM ai_usage WHERE id='+uid+'::uuid FOR UPDATE;\n'+\
 "DO $u_guard$ BEGIN IF NOT EXISTS(SELECT 1 FROM ai_usage u WHERE id="+uid+"::uuid AND row_to_json(u)::jsonb="+expected+"::jsonb) THEN RAISE EXCEPTION 'bound_usage_changed'; END IF; END $u_guard$;\n"+\
 "SELECT set_config('grace.owner_id',"+owner+",true); SET LOCAL ROLE grace_financial;\n"+\
 'SELECT grace_ai_settle('+uid+'::uuid,0,'+marker+');\n'+\
 "DO $u_guard$ BEGIN IF NOT EXISTS(SELECT 1 FROM ai_usage u WHERE id="+uid+"::uuid AND status='settled' AND actual_micros=0 AND generation_id="+marker+" AND settled_at IS NOT NULL AND (row_to_json(u)::jsonb - ARRAY['status','actual_micros','generation_id','settled_at'])="+expected_immutable+"::jsonb) THEN RAISE EXCEPTION 'settlement_identity_changed'; END IF; END $u_guard$;\n"+\
 select_row(row['id'])+'\n\\echo U_READY_TO_COMMIT\n'
def command(args,stdin=None):
 p=subprocess.run(args,input=stdin,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,timeout=15)
 check(p.returncode==0 and len(p.stdout)<=65536,'operator_command_failed')
 return p.stdout.strip()
def container():
 ids=command(['docker','ps','-q','--filter','label=com.docker.compose.project=grace-production','--filter','label=com.docker.compose.service=postgres']).splitlines()
 check(len(ids)==1 and re.fullmatch('[a-f0-9]{12,64}',ids[0]),'runtime_not_unique')
 return ids[0]
def sql_args(postgres):return ['docker','exec','-i',postgres,'psql','-X','-qAt','-P','pager=off','-v','ON_ERROR_STOP=1','-U','grace','-d','grace']
def apply_sql(args,row,stable):
 # Keep the SQL transaction open until Python checks the sealed files and row.
 p=subprocess.Popen(args,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,bufsize=0)
 try:
  request=transaction(row).encode();check(p.stdin.write(request)==len(request),'sql_input_refused');p.stdin.flush()
  pending=b'';deadline=time.monotonic()+25
  with selectors.DefaultSelector() as selector:
   selector.register(p.stdout,selectors.EVENT_READ)
   while b'U_READY_TO_COMMIT\n' not in pending:
    remaining=deadline-time.monotonic();check(remaining>0,'sql_ready_timeout')
    check(selector.select(remaining),'sql_ready_timeout')
    chunk=os.read(p.stdout.fileno(),65537-len(pending));check(bool(chunk),'sql_precommit_refused');pending+=chunk
    check(len(pending)<=65536,'sql_output_refused')
  lines=pending.decode().splitlines();check(lines.count('U_READY_TO_COMMIT')==1 and lines[-1]=='U_READY_TO_COMMIT','sql_ready_refused')
  candidates=[parse(line.encode()) for line in lines if line.startswith('{')]
  check(len(candidates)==1,'sql_row_projection_refused');after(candidates[0],row)
  stable()  # Failure here closes the connection and rolls back before COMMIT.
  commit=b'COMMIT;\n\\echo U_COMMITTED\n';check(p.stdin.write(commit)==len(commit),'sql_input_refused');p.stdin.close();p.stdin=None
  out,err=p.communicate(timeout=15)
  check(p.returncode==0 and len(out)<=65536 and len(err)<=65536 and out.decode().splitlines()==['U_COMMITTED'],'sql_commit_unverified')
  return candidates[0]
 except BaseException:
  if p.poll() is None and p.stdin is not None:
   try:
    p.stdin.write(b'ROLLBACK;\n');p.stdin.close();p.stdin=None;p.communicate(timeout=5)
   except BaseException:pass
  raise
 finally:
  if p.poll() is None:p.kill();p.wait(timeout=5)
  for pipe in (p.stdin,p.stdout,p.stderr):
   if pipe is not None:pipe.close()
def run():
 os.umask(0o077);fixed_projection()
 marker=snap('document-reading-spec0056-q4.started',512);sealed=snap('document-reading-spec0056-q4.private.json',32<<20)
 check(marker[2]==MARKER_FILE_SHA and sealed[2]==REPORT_FILE_SHA,'snapshot_changed')
 report=parse(sealed[0])['report'];uid=reserve_id(report)
 def stable():
  check(snap('document-reading-spec0056-q4.started',512)==marker and snap('document-reading-spec0056-q4.private.json',32<<20)==sealed,'snapshot_changed')
 lock=os.open(ROOT+'/statement-screening.lock',os.O_WRONLY|os.O_NOFOLLOW|os.O_CLOEXEC)
 try:
  check(stat.S_ISREG(os.fstat(lock).st_mode),'lock_not_regular');fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
  args=sql_args(container());row=parse(command(args,select_row(uid)).encode())
  try:old=snap(U_AUDIT,2<<20)
  except FileNotFoundError:old=None
  if old is not None:
   audit=parse(old[0]);check(audit.get('committed') is True and audit.get('reportSHA256')==REPORT and audit.get('billingBasis')==BASIS and audit.get('directMetadataVerified') is False and audit.get('acceptanceCommit')==ACCEPTANCE and type(audit.get('usageBefore')) is dict,'audit_conflict')
   after(row,audit['usageBefore']);stable()
   return projection(old[2],True)
  before(row,uid,dict(row));stable()
  audit={'version':'document-reading.q44.V.policy-settlement.v1','acceptanceCommit':ACCEPTANCE,'reportSHA256':REPORT,'reportFileSHA256':REPORT_FILE_SHA,'markerFileSHA256':MARKER_FILE_SHA,'billingBasis':BASIS,'directMetadataVerified':False,'usageBefore':row,'actualMicros':0,'contingencyMicros':32023,'committed':False,'status':'AUTHORIZED_PENDING'}
  fd=os.open(ROOT+'/'+U_AUDIT,os.O_RDWR|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600)
  def save():
   raw=json.dumps(audit,separators=(',',':')).encode();check(len(raw)<=2<<20,'audit_size_refused');os.lseek(fd,0,os.SEEK_SET);os.ftruncate(fd,0)
   with os.fdopen(os.dup(fd),'wb') as stream:stream.write(raw);stream.flush();os.fsync(stream.fileno())
   return hashlib.sha256(raw).hexdigest()
  try:
   save()
   try:
    apply_sql(args,row,stable)
    current=parse(command(args,select_row(uid)).encode());after(current,row);stable()
    audit.update(committed=True,status='VERIFIED_OPERATION',usageAfter=current)
   except Stop as refusal:audit.update(status='UNVERIFIED',reason=str(refusal))
   except BaseException:audit.update(status='UNVERIFIED',reason='settlement_unverified')
   digest=save()
   if audit['committed']:return projection(digest,False)
   return {'version':audit['version'],'status':'UNVERIFIED','reason':audit['reason'],'resumeAllowed':False,'auditFileSHA256':digest,'metadataGET':0,'paidPOST':0,'directMetadataVerified':False}
  finally:os.close(fd)
 finally:os.close(lock)
def projection(digest,replay):return {'version':'document-reading.q44.V.policy-settlement.v1','status':'VERIFIED_OPERATION','billingBasis':BASIS,'directMetadataVerified':False,'settlementApplied':True,'actualMicros':0,'contingencyMicros':32023,'targetUsageRows':1,'ownerUnchanged':True,'snapshotUnchanged':True,'metadataGET':0,'paidPOST':0,'idempotentReplay':replay,'auditFileSHA256':digest,'resumeAllowed':True}
if __name__=='__main__':
 try:print(json.dumps(run(),separators=(',',':')))
 except Stop as failure:print(json.dumps({'status':'UNVERIFIED','reason':str(failure),'resumeAllowed':False}));sys.exit(1)
 except BaseException:print('{"status":"UNVERIFIED","reason":"settlement_refused","resumeAllowed":false}');sys.exit(1)
