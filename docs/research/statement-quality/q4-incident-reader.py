import os,sys,json,stat,hashlib,base64,re,decimal
ROOT='/opt/grace/shared'
SOURCE='9200b731a0dea8f171e31aaf988d038073ad577f'
MANIFEST='10200604c2a134577b3d22eed84fd106d7d951b1aececa0ec47f978dbf85653a'
APPROVAL='8716e30c025fffb8718035ba548d2165127d83f2'
REPORT='a3220ca4f7a82a69e39c62aff70afde6f2a9cf9a718ca88794bf0a67fe7a84be'
MODEL='google/gemini-3.1-flash-lite'
CANONICAL='google/gemini-3.1-flash-lite-20260507'
TAG='google-vertex/eu'
ENDPOINT='/api/v1/models/'+CANONICAL+'/endpoints'
class Refusal(Exception): pass
def need(value):
 if not value: raise Refusal()
def unique(pairs):
 out={};seen=set()
 for k,v in pairs:
  need(k.lower() not in seen);seen.add(k.lower());out[k]=v
 return out
def parse(raw):
 return json.loads(raw.decode('utf-8'),object_pairs_hook=unique,parse_constant=lambda _:(_ for _ in ()).throw(Refusal()))
def canonical(v):
 return json.dumps(v,ensure_ascii=False,separators=(',',':')).replace('&','\\u0026').replace('<','\\u003c').replace('>','\\u003e').replace('\u2028','\\u2028').replace('\u2029','\\u2029').encode()
def snap(name,limit):
 fd=os.open(ROOT+'/'+name,os.O_RDONLY|os.O_NOFOLLOW|os.O_CLOEXEC|os.O_NONBLOCK)
 try:
  a=os.fstat(fd);need(stat.S_ISREG(a.st_mode) and stat.S_IMODE(a.st_mode)==0o600 and 0<a.st_size<=limit)
  data=b''
  while len(data)<=limit:
   part=os.read(fd,min(65536,limit+1-len(data)))
   if not part: break
   data+=part
  b=os.fstat(fd)
  attrs=lambda x:(x.st_dev,x.st_ino,x.st_mode,x.st_uid,x.st_gid,x.st_size,x.st_mtime_ns,x.st_ctime_ns)
  need(attrs(a)==attrs(b) and len(data)==a.st_size)
  return data,attrs(a),hashlib.sha256(data).hexdigest()
 finally: os.close(fd)
def within(raw,cap,per_token):
 if not isinstance(raw,str) or len(raw)>64 or not re.fullmatch(r'(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]{1,2})?',raw): return False
 value=decimal.Decimal(raw)
 return value>=0 and value<=decimal.Decimal(cap)/decimal.Decimal(10**(12 if per_token else 6))
def params(v):
 return isinstance(v,list) and all(x in v for x in ['structured_outputs','response_format','max_tokens'])
def bounded_int(v,low,high): return type(v) is int and low<=v<=high
def diagnosis():
 marker=snap('document-reading-spec0056-q.started',512)
 need(marker[0]==(SOURCE+' '+MANIFEST+' '+APPROVAL+'\n').encode())
 raw=snap('document-reading-spec0056-q.private.json',32<<20)
 outer=parse(raw[0]);need(set(outer)=={'sha256','report'} and outer['sha256']==REPORT)
 report=outer['report'];need(hashlib.sha256(canonical(report)).hexdigest()==REPORT)
 need(report['manifest']['source']==SOURCE and hashlib.sha256(canonical(report['manifest'])).hexdigest()==MANIFEST)
 need(report['grant']['source']==SOURCE and report['grant']['approval']==APPROVAL and report['grant']['manifestSHA256']==MANIFEST)
 need(report['core']['decision']=='INCONCLUSIVE' and report['core']['proofs']=={})
 need(report['budget'] in (None,[]))
 cases=report['core']['results'];need(len(cases)==8 and all(c['state']=='not_sent' and c['reason']=='preflight' and c['settlement']=='not_reserved' for c in cases))
 wire=report['wire'];need(len(wire)==3)
 documents=[];statuses=[]
 for event,path in zip(wire,['/api/v1/models','/api/v1/endpoints/zdr',ENDPOINT]):
  need(event['method']=='GET' and event['path']==path and event['requestSHA256']==hashlib.sha256(b'').hexdigest())
  need(bounded_int(event['status'],0,599) and type(event['failure']) is bool)
  statuses.append({'httpStatus':event['status'],'transportFailure':event['failure'],'jsonContentType':event['contentType']=='application/json'})
  need(not event['failure'] and event['status']==200)
  body=base64.b64decode(event['body'],validate=True);need(len(body)<=8<<20)
  documents.append(parse(body))
 catalog,zdr,endpoints=documents[0]['data'],documents[1]['data'],documents[2]['data']['endpoints']
 need(isinstance(catalog,list) and isinstance(zdr,list) and isinstance(endpoints,list) and len(catalog)<=20000 and len(zdr)<=20000 and len(endpoints)<=100)
 catalog_matches=[m for m in catalog if m.get('id')==MODEL and m.get('canonical_slug')==CANONICAL and 'text' in m.get('architecture',{}).get('input_modalities',[]) and 'image' in m.get('architecture',{}).get('input_modalities',[]) and 'text' in m.get('architecture',{}).get('output_modalities',[]) and params(m.get('supported_parameters'))]
 zdr_matches=[e for e in zdr if e.get('model_id') in (MODEL,CANONICAL) and e.get('tag')==TAG and type(e.get('status')) is int and e['status']==0]
 rows=[]
 for e in endpoints:
  if e.get('tag')!=TAG: continue
  need(len(rows)<4)
  price=e.get('pricing',{});need(isinstance(price,dict))
  flags={
   'modelIdPresent':'model_id' in e,'modelIdMatches':e.get('model_id') in (MODEL,CANONICAL),
   'modelNamePresent':'model_name' in e,'modelNameMatches':e.get('model_name') in (MODEL,CANONICAL),
   'providerNameMatches':e.get('provider_name')=='Google','statusZero':type(e.get('status')) is int and e['status']==0,
   'contextFits':bounded_int(e.get('context_length'),46000,100000000),
   'outputFits':bounded_int(e.get('max_completion_tokens'),16000,100000000),
   'inputFits':e.get('max_prompt_tokens') is None or bounded_int(e.get('max_prompt_tokens'),30000,100000000),
   'parametersFit':params(e.get('supported_parameters')),
   'promptPresent':isinstance(price.get('prompt'),str) and price['prompt']!='',
   'completionPresent':isinstance(price.get('completion'),str) and price['completion']!='',
   'promptWithinCap':within(price.get('prompt'),275000,True),
   'completionWithinCap':within(price.get('completion'),1650000,True),
   'requestFeeKeyPresent':'request' in price,
   'requestFeeNull':'request' in price and price['request'] is None,
   'requestFeeString':isinstance(price.get('request'),str),
   'requestFeeEmptyString':isinstance(price.get('request'),str) and price['request']=='',
   'requestFeePresent':isinstance(price.get('request'),str) and price['request']!='',
   'imageFeePresent':isinstance(price.get('image'),str) and price['image']!='',
   'imagePriceNumeric':isinstance(price.get('image'),str) and len(price['image'])<=64 and re.fullmatch(r'(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]{1,2})?',price['image']) is not None,
   'imagePricePositive':isinstance(price.get('image'),str) and len(price['image'])<=64 and re.fullmatch(r'(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]{1,2})?',price['image']) is not None and decimal.Decimal(price['image'])>0,
   'requestFeeWithinZero':within(price.get('request'),0,False),
   'imageFeeWithinZero':within(price.get('image'),0,False)}
  rows.append(flags)
 need(snap('document-reading-spec0056-q.started',512)==marker)
 need(snap('document-reading-spec0056-q.private.json',32<<20)==raw)
 return {'version':'document-reading.q4.readonly-diagnosis.v1','status':'VERIFIED','source':SOURCE,'manifestSHA256':MANIFEST,'reportSHA256':REPORT,'reportFileSHA256':raw[2],'markerFileSHA256':marker[2],'snapshotUnchanged':True,'networkCalls':0,'sqlCalls':0,'persistentWritesRequested':0,'storedPOST':0,'storedMetadataGET':3,'storedBudgetEvents':0,'metadata':statuses,'catalogCandidateMatches':len(catalog_matches),'zdrCandidateMatches':len(zdr_matches),'endpointCandidateRows':rows}
try:
 output=diagnosis();encoded=json.dumps(output,separators=(',',':')).encode();need(len(encoded)<16384);sys.stdout.buffer.write(encoded+b'\n')
except BaseException:
 sys.stdout.write('{"version":"document-reading.q4.readonly-diagnosis.v1","status":"UNVERIFIED","reason":"snapshot_or_projection_refused","networkCalls":0,"sqlCalls":0,"persistentWritesRequested":0}\n');sys.exit(1)
