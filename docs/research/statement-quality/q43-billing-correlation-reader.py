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
# SPEC-0056 T1: first bounded SSH session. No SQL, key access or network.
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

def run():
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
if __name__=='__main__':
 try:
  result=run();data=json.dumps(result,separators=(',',':')).encode();need(len(data)<=2048);sys.stdout.buffer.write(data+b'\n')
 except BaseException:
  print('{"status":"UNVERIFIED","reason":"snapshot_or_correlation_refused","resumeAllowed":false}')
  sys.exit(1)
