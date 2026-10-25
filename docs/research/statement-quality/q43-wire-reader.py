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
def run():
 m=snap('document-reading-spec0056-q3.started',512);need(m[0]==(SOURCE+' '+MANIFEST+' '+APPROVAL+'\n').encode())
 raw=snap('document-reading-spec0056-q3.private.json',32<<20);outer=parse(raw[0]);r=outer['report'];need(outer['sha256']==REPORT and hashlib.sha256(canonical(r)).hexdigest()==REPORT)
 need(r['manifest']['source']==SOURCE and hashlib.sha256(canonical(r['manifest'])).hexdigest()==MANIFEST)
 need(r['grant']['approval']==APPROVAL and r['core']['decision']=='INCONCLUSIVE')
 wire=r['wire'];posts=[e for e in wire if e['method']=='POST'];need(len(posts)==1 and sum(e['method']=='GET' for e in wire)==6)
 e=posts[0];need(e['path']=='/api/v1/chat/completions')
 need(type(e['status']) is int and 0<=e['status']<=599)
 body=base64.b64decode(e['body'],validate=True)
 try:v=parse(body);valid=True
 except BaseException:v={};valid=False
 error=v.get('error',{}) if type(v) is dict else {};error=error if type(error) is dict else {}
 # Only booleans over a fixed vocabulary, never arbitrary message/provider text.
 strings=[]
 def walk(v,depth=0):
  if depth>16:return
  if type(v) is str:strings.append(v[:16000])
  elif type(v) is dict:
   for k,x in v.items():
    strings.append(k[:200]);walk(x,depth+1)
  elif type(v) is list:
   for x in v[:32]:walk(x,depth+1)
 walk(error)
 text='\n'.join(strings).lower()
 safe_keys={'response_format','json_schema','schema','properties','header','rows','items','anchor','fields','evidence','box','value','quote','page','required','additionalProperties','$defs','$ref','field','anyOf','type','minimum','maximum','minItems','maxItems','strict','name','max_tokens','reasoning','enabled','provider','only','max_price','prompt','completion','request','image','operationDate','bookingDate','accountAmount','accountCurrency','direction','transactionAmount','transactionCurrency','description'}
 codes={'invalid_type','unrecognized_keys','invalid_union','invalid_value','invalid_key','custom','too_small','too_big','invalid_format','invalid_element'}
 details=[]
 def structured(v,depth=0):
  if depth>20 or len(details)>=32:return
  if type(v) is dict:
   d={}
   if v.get('code') in codes:d['code']=v['code']
   if type(v.get('path')) is list:d['path']=[x if type(x) is int and 0<=x<=200 else x if type(x) is str and x in safe_keys else 'other' for x in v['path'][:20]]
   for key in ('expected','received'):
    if v.get(key) in ('object','array','string','number','integer','boolean','null','undefined'):d[key]=v[key]
   if type(v.get('keys')) is list:d['keys']=[x if type(x) is str and x in safe_keys else 'other' for x in v['keys'][:20]]
   if d:details.append(d)
   for x in v.values():structured(x,depth+1)
  elif type(v) is list:
   for x in v[:32]:structured(x,depth+1)
  elif type(v) is str and len(v)<16000:
   try:parsed=parse(v.encode())
   except BaseException:return
   structured(parsed,depth+1)
 structured(error)
 words=['schema','invalid','json','minimum','maximum','minitems','maxitems','$ref','$defs','anyof','constraint','complex','states','branch','billing','credit','payment','quota','rate limit','endpoint','unsupported','required','additionalproperties','response_format','json_schema','max_tokens','reasoning','provider','only','max_price','prompt','completion','request','image','invalid_type','unrecognized_keys','invalid_union','invalid_value','invalid_key','expected','unrecognized','strict','definitions','nullable','null','object','array','string','too_small','too_big','not supported','not found']
 usage=v.get('usage') if type(v) is dict else None
 cost=usage.get('cost') if type(usage) is dict else None
 code=error.get('code');code=code if type(code) is int and 0<=code<=99999 else None
 budget=r.get('budget') or [];reserves=[x['amount'] for x in budget if x.get('kind')=='reserve' and not x.get('failure')]
 need(all(type(n) is int and 0<=n<=250000 for n in reserves))
 need(snap('document-reading-spec0056-q3.started',512)==m and snap('document-reading-spec0056-q3.private.json',32<<20)==raw)
 return {'version':'document-reading.q43.wire-diagnosis.v1','status':'VERIFIED','source':SOURCE,'manifestSHA256':MANIFEST,'reportSHA256':REPORT,'reportFileSHA256':raw[2],'markerFileSHA256':m[2],'snapshotUnchanged':True,'providerCalls':0,'sqlCalls':0,'persistentWritesRequested':0,'storedPOST':1,'storedMetadataGET':6,'postHTTPStatus':e['status'],'transportFailure':e['failure'],'responseBytes':len(body),'responseSHA256':hashlib.sha256(body).hexdigest(),'responseJSONValid':valid,'errorObjectPresent':bool(error),'errorCode':code,'errorVocabulary':{w:w in text for w in words},'structuredValidationDetails':details,'usageObjectPresent':type(usage) is dict,'costFieldPresent':type(usage) is dict and 'cost' in usage,'explicitNumericZeroCost':type(cost) in (int,float) and cost==0,'successfulReserveMicros':reserves}
if __name__=='__main__':
 try:
  b=json.dumps(run(),separators=(',',':')).encode();need(len(b)<=8192);sys.stdout.buffer.write(b+b'\n')
 except BaseException:print('{"status":"UNVERIFIED","reason":"snapshot_or_projection_refused"}');sys.exit(1)
