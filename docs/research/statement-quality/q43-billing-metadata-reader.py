import os,sys,json,stat,hashlib,base64,re
ROOT='/opt/grace/shared'
SOURCE='a49542e2db7522adf30171ef2cf4b2198016fa67'
MANIFEST='8f378b10d708075d5a3badcb4cf3f7d016c967f5ff86c8854c0c8b541e1de3da'
APPROVAL='689e0952b85353dd7aee95834ab0b4a01dced6bb'
REPORT='c6f93f6246c9e37612c7861865fdab606b700184130aa8d4427e79fa9fad5d4f'
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
# SPEC-0056 T1: fixed snapshot validation before bounded metadata inspection.
MARKER_FILE_SHA='730afbd8b27061c0e31e06ebb92d925190580aee6793b70e1031d5da475c3b8c'
REPORT_FILE_SHA='7f8ed93cf6862972cccdce3486fd81bed58db2e3dbd92f8821891f52cc05dee7'
BODY_SHA='281722f9e49cac3f8d58d79d47d450f0fb11928c32b41b031633d6c8e6adc5fe'
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

def fixed_projection():
 marker=snap('document-reading-spec0056-q3.started',512)
 need(marker[0]==(SOURCE+' '+MANIFEST+' '+APPROVAL+'\n').encode() and marker[2]==MARKER_FILE_SHA)
 sealed=snap('document-reading-spec0056-q3.private.json',32<<20)
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
 need(len(body)==642 and hashlib.sha256(body).hexdigest()==BODY_SHA)
 envelope=parse(body)
 need(type(envelope) is dict and type(envelope.get('error')) is dict)
 ids=correlations(envelope)
 need(snap('document-reading-spec0056-q3.started',512)==marker and snap('document-reading-spec0056-q3.private.json',32<<20)==sealed)
 reason='generation_correlation_absent' if not ids else 'generation_correlation_ambiguous' if len(ids)!=1 else 'correlation_available_billing_not_checked'
 return {'version':'document-reading.q43.T.correlation-inspection.v1','status':'UNVERIFIED','reason':reason,'snapshotVerified':True,'snapshotUnchanged':True,'reportSHA256':REPORT,'reportFileSHA256':sealed[2],'markerFileSHA256':marker[2],'responseSHA256':BODY_SHA,'correlationPresent':bool(ids),'correlationUnique':len(ids)==1,'metadataGET':0,'sqlCalls':0,'keyReads':0,'persistentWritesRequested':0,'settlementApplied':False,'paidPOST':0,'resumeAllowed':False}
# Owner-supplied correlation after the accepted T1 absence branch. Never published.
import datetime as dt, decimal, fcntl, ssl, subprocess, urllib.request, urllib.parse, urllib.error, uuid
AUDIT='document-reading-spec0056-q3.reconciliation.json'
MODEL='google/gemini-3.1-flash-lite'
CANONICAL_MODEL='google/gemini-3.1-flash-lite-20260507'
START='2026-10-25T14:58:49Z'
END='2026-10-25T14:59:44Z'
class Stop(Exception):pass
def check(ok,reason):
 if not ok:raise Stop(reason)
def timestamp(value):
 check(type(value) is str,'timestamp_invalid')
 try:stamp=dt.datetime.fromisoformat(value.replace('Z','+00:00'))
 except ValueError:raise Stop('timestamp_invalid')
 check(stamp.tzinfo is not None,'timestamp_invalid')
 return stamp.astimezone(dt.timezone.utc)
def valid_uuid(value):
 try:return type(value) is str and str(uuid.UUID(value))==value
 except (ValueError,AttributeError):return False
def command(args,stdin=None):
 result=subprocess.run(args,input=stdin,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,timeout=12)
 check(result.returncode==0 and len(result.stdout)<=65536,'operator_command_failed')
 return result.stdout.strip()
def container(service):
 values=command(['docker','ps','-q','--filter','label=com.docker.compose.project=grace-production','--filter','label=com.docker.compose.service='+service]).splitlines()
 check(len(values)==1 and re.fullmatch('[a-f0-9]{12,64}',values[0]),'runtime_not_unique')
 return values[0]
def reservation(report):
 reserves=[e for e in report['budget'] if e['kind']=='reserve' and not e['failure']]
 check(len(reserves)==1,'reservation_not_unique');event=reserves[0]
 check(valid_uuid(event.get('id')) and event['purpose']=='statement' and event['amount']==30329,'reservation_mismatch')
 return event['id']
def validate_row(row,ident):
 check(row.get('id')==ident and valid_uuid(row.get('owner_id')),'row_identity_mismatch')
 check(row.get('purpose')=='statement' and row.get('model')==MODEL and row.get('provider')=='google-vertex/eu' and row.get('reserved_micros')==30329,'row_policy_mismatch')
 check(timestamp(START)<=timestamp(row.get('created_at'))<=timestamp(END),'row_time_mismatch')
 check(row.get('status')=='uncertain' and row.get('actual_micros') is None and row.get('generation_id') is None,'row_not_expected_uncertain')
def metadata_parse(raw):
 return json.loads(raw.decode(),object_pairs_hook=unique,parse_float=decimal.Decimal,parse_constant=lambda _:(_ for _ in ()).throw(Refusal()))
def validate_metadata(data,ident):
 check(type(data) is dict and data.get('id')==ident,'generation_identity_mismatch')
 check(data.get('model') in (MODEL,CANONICAL_MODEL) and data.get('is_byok') is False,'generation_route_mismatch')
 check(timestamp(START)<=timestamp(data.get('created_at'))<=timestamp(END),'generation_time_mismatch')
 check(data.get('finish_reason')=='error','generation_finish_mismatch')
 cost=data.get('total_cost')
 check(type(cost) in (int,decimal.Decimal) and not isinstance(cost,bool),'generation_cost_unknown')
 amount=decimal.Decimal(cost)
 check(amount.is_finite() and 0<=amount<=decimal.Decimal('0.030329'),'generation_cost_out_of_bounds')
 check(amount==0 or amount.adjusted()>=-1000,'generation_cost_precision_refused')
 with decimal.localcontext() as context:
  context.prec=max(28,len(amount.as_tuple().digits)+8)
  actual=int((amount*1000000).to_integral_value(rounding=decimal.ROUND_CEILING))
 provider=data.get('provider_name')
 if provider!='Google':
  zero_tokens=all(type(data.get(k)) is int and data[k]==0 for k in ('tokens_prompt','tokens_completion'))
  zero_native=all(data.get(k) in (None,0) and not isinstance(data.get(k),bool) for k in ('native_tokens_prompt','native_tokens_completion','native_tokens_reasoning'))
  check(provider in (None,'') and actual==0 and zero_tokens and zero_native,'generation_provider_mismatch')
 return actual
class NoRedirect(urllib.request.HTTPRedirectHandler):
 def redirect_request(self,*args,**kwargs):return None
def get_metadata(ident,key):
 opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),NoRedirect(),urllib.request.HTTPSHandler(context=ssl.create_default_context()))
 request=urllib.request.Request('https://openrouter.ai/api/v1/generation?'+urllib.parse.urlencode({'id':ident}),headers={'Authorization':'Bearer '+key},method='GET')
 try:
  with opener.open(request,timeout=15) as response:
   check(response.status==200,'metadata_http_refused');raw=response.read((1<<20)+1)
 except urllib.error.HTTPError:raise Stop('metadata_http_refused')
 check(0<len(raw)<=1<<20,'metadata_size_refused')
 return raw

def inspect(ident):
 check(type(ident) is str and GENERATION.fullmatch(ident),'owner_correlation_invalid')
 os.umask(0o077)
 fixed_projection()
 sealed=snap('document-reading-spec0056-q3.private.json',32<<20)
 check(sealed[2]==REPORT_FILE_SHA,'snapshot_changed')
 report=parse(sealed[0])['report'];uid=reservation(report)
 lock=os.open(ROOT+'/statement-screening.lock',os.O_WRONLY|os.O_NOFOLLOW|os.O_CLOEXEC)
 try:
  check(stat.S_ISREG(os.fstat(lock).st_mode),'lock_not_regular');fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
  postgres=container('postgres')
  query="SELECT row_to_json(u) FROM ai_usage u WHERE id='"+uid+"'::uuid;"
  row=parse(command(['docker','exec','-i',postgres,'psql','-X','-v','ON_ERROR_STOP=1','-U','grace','-d','grace','-At'],query).encode())
  sql_reads=1
  validate_row(row,uid)
  api=container('api');key=command(['docker','exec',api,'cat','/run/secrets/openrouter_api_key'])
  check(0<len(key)<=512 and not any(c.isspace() for c in key),'provider_key_unavailable')
  audit={'version':'document-reading.q43.T.reconciliation.v1','reportSHA256':REPORT,'reportFileSHA256':REPORT_FILE_SHA,'markerFileSHA256':MARKER_FILE_SHA,'ownerSuppliedCorrelation':ident,'correlationSource':'explicit_owner_activity_reply','usageBefore':row,'metadataGETConsumed':True,'status':'PENDING','committed':False}
  fd=os.open(ROOT+'/'+AUDIT,os.O_RDWR|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600)
  def save():
   encoded=json.dumps(audit,default=str,separators=(',',':')).encode();check(len(encoded)<=2<<20,'audit_size_refused')
   os.lseek(fd,0,os.SEEK_SET);os.ftruncate(fd,0)
   with os.fdopen(os.dup(fd),'wb') as output:output.write(encoded);output.flush();os.fsync(output.fileno())
  try:
   save()
   try:
    raw=get_metadata(ident,key);audit['metadataRawBase64']=base64.b64encode(raw).decode();audit['metadataSHA256']=hashlib.sha256(raw).hexdigest()
    data=metadata_parse(raw).get('data');actual=validate_metadata(data,ident)
    check(snap('document-reading-spec0056-q3.private.json',32<<20)==sealed,'snapshot_changed')
    sql_reads+=1
    current=parse(command(['docker','exec','-i',postgres,'psql','-X','-v','ON_ERROR_STOP=1','-U','grace','-d','grace','-At'],query).encode())
    check(current==row,'usage_changed')
    audit.update(status='VERIFIED',actualMicros=actual,metadata=data)
   except Stop as refusal:audit.update(status='UNVERIFIED',reason=str(refusal))
   except BaseException:audit.update(status='UNVERIFIED',reason='inspection_failed_private_details_retained')
   save()
   return {'version':audit['version'],'status':audit['status'],'reason':audit.get('reason'),'reportSHA256':REPORT,'snapshotVerified':True,'metadataGET':1,'targetUsageRowRead':True,'sqlReads':sql_reads,'settlementApplied':False,'actualMicros':audit.get('actualMicros'),'auditFileSHA256':hashlib.sha256(json.dumps(audit,default=str,separators=(',',':')).encode()).hexdigest(),'resumeAllowed':False}
  finally:os.close(fd)
 finally:os.close(lock)
if __name__=='__main__':
 try:print(json.dumps(inspect(OWNER_GENERATION),separators=(',',':')))
 except Stop as failure:print(json.dumps({'status':'UNVERIFIED','reason':str(failure),'settlementApplied':False,'resumeAllowed':False}));sys.exit(1)
 except BaseException:print('{"status":"UNVERIFIED","reason":"inspection_refused","settlementApplied":false,"resumeAllowed":false}');sys.exit(1)
