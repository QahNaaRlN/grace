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
def diagnostic(envelope):
 words=['schema','invalid','argument','request','provider','unsupported','minimum','maximum','minitems','maxitems','$ref','$defs','anyof','constraint','complex','size','length','limit','required','additionalproperties','response_schema','responsejsonschema','response_format','json_schema','max_tokens','reasoning','max_price','image','nullable','not supported','too large','too long','quota','billing','payment']
 statuses={'INVALID_ARGUMENT','NOT_FOUND','RESOURCE_EXHAUSTED','PERMISSION_DENIED','INTERNAL','UNAVAILABLE','FAILED_PRECONDITION'}
 safe={'generation_config','generationConfig','response_schema','responseSchema','responseJsonSchema','response_format','json_schema','schema','properties','required','additionalProperties','type','items','anyOf','$ref','$defs','nullable','minimum','maximum','minItems','maxItems','header','rows','evidence','anchor','box','page','quote','fields','value','description','max_tokens','reasoning','provider','max_price','prompt','completion','request','image','strict','name','operationDate','bookingDate','accountAmount','accountCurrency','direction','transactionAmount','transactionCurrency'}
 strings=[];codes=set();paths=[];ids=set()
 def walk(value,depth=0):
  need(depth<=24)
  if type(value) is dict:
   need(len(value)<=100)
   for key,item in value.items():
    strings.append(key[:200])
    if key=='status' and type(item) is str and item in statuses:codes.add(item)
    if key in ('id','generation_id','generationId') and type(item) is str and re.fullmatch(r'gen-[A-Za-z0-9_-]{8,196}',item):ids.add(item)
    if key=='field' and type(item) is str and len(item)<=1024:
     tokens=re.findall(r'[A-Za-z_$][A-Za-z0-9_$]*|[0-9]+',item)
     paths.append([x if x in safe else int(x) if x.isdigit() and len(x)<=3 and int(x)<=200 else 'other' for x in tokens[:40]])
    walk(item,depth+1)
  elif type(value) is list:
   need(len(value)<=100)
   for item in value:walk(item,depth+1)
  elif type(value) is str:
   strings.append(value[:16000])
   if len(value)<=16000 and value.lstrip().startswith(('{','[')):
    try:nested=parse(value.encode())
    except (ValueError,UnicodeError,Refusal):return
    walk(nested,depth+1)
 walk(envelope)
 text='\n'.join(strings).lower()
 return {'errorVocabulary':{word:word in text for word in words},'providerStatusCodes':sorted(codes),'fieldViolationPaths':paths[:16],'genericInvalidArgumentMessage':'request contains an invalid argument' in text,'providerReturnedErrorMessage':'provider returned error' in text,'generationCorrelationPresent':bool(ids)}
def run():
 marker=snap('document-reading-spec0056-q4.started',512)
 need(marker[0]==(SOURCE+' '+MANIFEST+' '+APPROVAL+'\n').encode())
 sealed=snap('document-reading-spec0056-q4.private.json',32<<20);outer=parse(sealed[0]);r=outer['report']
 need(outer['sha256']==REPORT and hashlib.sha256(canonical(r)).hexdigest()==REPORT)
 need(r['manifest']['source']==SOURCE and hashlib.sha256(canonical(r['manifest'])).hexdigest()==MANIFEST and r['grant']['approval']==APPROVAL and r['core']['decision']=='INCONCLUSIVE')
 wire=r['wire'];posts=[e for e in wire if e['method']=='POST'];need(len(posts)==1 and sum(e['method']=='GET' for e in wire)==6)
 e=posts[0];need(e['path']=='/api/v1/chat/completions' and type(e['status']) is int and 0<=e['status']<=599)
 body=base64.b64decode(e['body'],validate=True)
 try:value=parse(body);valid=True
 except (ValueError,UnicodeError,Refusal):value={};valid=False
 reserves=[e['amount'] for e in r['budget'] if e['kind']=='reserve' and not e['failure']]
 need(len(reserves)==1 and all(type(n) is int and 0<n<=35654 for n in reserves))
 error=value.get('error') if type(value) is dict else None
 usage=value.get('usage') if type(value) is dict else None
 out={'version':'document-reading.q44.wire-diagnosis.v1','status':'VERIFIED_SNAPSHOT','source':SOURCE,'manifestSHA256':MANIFEST,'reportSHA256':REPORT,'reportFileSHA256':sealed[2],'markerFileSHA256':marker[2],'snapshotUnchanged':True,'providerCalls':0,'sqlCalls':0,'keyReads':0,'persistentWritesRequested':0,'storedPOST':1,'storedMetadataGET':6,'postHTTPStatus':e['status'],'transportFailure':e['failure'],'responseBytes':len(body),'responseSHA256':hashlib.sha256(body).hexdigest(),'responseJSONValid':valid,'errorObjectPresent':type(error) is dict,'usageObjectPresent':type(usage) is dict,'costFieldPresent':type(usage) is dict and 'cost' in usage,'successfulReserveMicros':reserves}
 out.update(diagnostic(value))
 need(snap('document-reading-spec0056-q4.started',512)==marker and snap('document-reading-spec0056-q4.private.json',32<<20)==sealed)
 return out
if __name__=='__main__':
 try:
  data=json.dumps(run(),separators=(',',':')).encode();need(len(data)<=8192);sys.stdout.buffer.write(data+b'\n')
 except BaseException:print('{"status":"UNVERIFIED","reason":"snapshot_or_projection_refused"}');sys.exit(1)
