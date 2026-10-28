"""Read only the two fixed Q4.2 files; emit closed structural diagnostics."""
import os, sys, json, stat, hashlib, base64, math
ROOT='/opt/grace/shared'
SOURCE='b1d310ed8746d4c1159d0b345c4acc3a7d8d3fdf'
MANIFEST='28f1e68838743a597203a938e0a36b80bf60c459d24e26614e86c08f307aded5'
APPROVAL='75ebae2390d1fef818e5da8da9b3257e0316b909'
REPORT='444feb9ddf89d4b8a13c532a6638a65fdfab271e6932747dd4fa8693ab2b7329'
HEAD=['openingBookDate','closingBookDate','openingBookBalance','closingBookBalance','availableBalanceObserved','debitTurnover','creditTurnover','currency']
ROW=['operationDate','bookingDate','accountAmount','accountCurrency','direction','transactionAmount','transactionCurrency','description']
class Refusal(Exception): pass
def need(v):
 if not v: raise Refusal()
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
 fd=os.open(ROOT+'/'+name,os.O_RDONLY|os.O_NOFOLLOW|os.O_CLOEXEC|os.O_NONBLOCK|os.O_NOATIME)
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
def inspect(content):
 out={'contentBytes':len(content.encode()),'contentSHA256':hashlib.sha256(content.encode()).hexdigest(),'sizeWithinDirectLimit':0<len(content.encode())<=2<<20,'hasMarkdownFence':content.strip().startswith('```')}
 try: v=parse(content.encode())
 except BaseException:
  out['jsonValidUnique']=False;return out
 out['jsonValidUnique']=True
 shape=[];invalid=[];shapes=0;constraints=0;quotes=0
 def problem(target,path,code):
  nonlocal shapes,constraints
  if target is shape: shapes+=1
  else: constraints+=1
  if len(target)<24: target.append({'path':path,'code':code})
 def obj(v,keys,path):
  if type(v) is not dict: problem(shape,path,'object_required');return False
  for k in keys:
   if k not in v: problem(shape,path+'.'+k,'key_required')
  if set(v)-set(keys): problem(shape,path,'unknown_keys')
  return True
 def string(v,path):
  if type(v) is not str: problem(shape,path,'string_required');return False
  return True
 def array(v,path):
  if v is None:return []
  if type(v) is not list: problem(shape,path,'array_required');return []
  return v
 def qs(v,path):
  nonlocal quotes
  items=array(v,path)
  if not 1<=len(items)<=8: problem(invalid,path,'quote_count')
  for i,q in enumerate(items):
   quotes+=1;p=path+'['+str(i+1)+']'
   if not obj(q,['page','quote','box'],p): continue
   n=q.get('page')
   if type(n) not in (int,float) or not math.isfinite(n) or n!=int(n): problem(shape,p+'.page','integer_required')
   elif not 1<=n<=1: problem(invalid,p+'.page','page_out_of_bounds')
   s=q.get('quote')
   if string(s,p+'.quote') and (not s.strip() or len(s.encode())>8000):problem(invalid,p+'.quote','quote_blank_or_oversize')
   box=q.get('box');nums=array(box,p+'.box')
   if box is not None and len(nums)!=4:problem(shape,p+'.box','array_length')
   good=True
   for n in nums:
    if type(n) not in (int,float) or not math.isfinite(n):problem(shape,p+'.box','number_required');good=False
   if good:
    nums=(nums+[0,0,0,0])[:4]
    if any(not 0<=n<=1 for n in nums):problem(invalid,p+'.box','box_out_of_range')
    if nums[0]>=nums[2] or nums[1]>=nums[3]:problem(invalid,p+'.box','box_nonpositive_area')
 def fields(v,keys,path):
  if type(v) is not dict:problem(shape,path,'map_required');return
  if set(v)!=set(keys):problem(invalid,path,'field_keys_incomplete')
  for k in keys:
   if k not in v or v[k] is None:continue
   f=v[k];p=path+'.'+k
   if not obj(f,['value','evidence'],p):continue
   s=f.get('value')
   if string(s,p+'.value') and (not s.strip() or len(s.encode())>16000):problem(invalid,p+'.value','field_blank_or_oversize')
   qs(f.get('evidence'),p+'.evidence')
 if obj(v,['header','rows'],'root'):
  fields(v.get('header'),HEAD,'header')
  rows=array(v.get('rows'),'rows');out['rowCount']=len(rows)
  out['headerFieldCount']=len(v['header']) if type(v.get('header')) is dict else 0
  if not 1<=len(rows)<=200:problem(invalid,'rows','row_count')
  for i,r in enumerate(rows[:201]):
   p='rows['+str(i+1)+']'
   if not obj(r,['anchor','fields','issues'],p):continue
   qs(r.get('anchor'),p+'.anchor');fields(r.get('fields'),ROW,p+'.fields')
   issues=array(r.get('issues'),p+'.issues')
   if len(issues)>32:problem(invalid,p+'.issues','issue_count')
   for issue in issues:
    if string(issue,p+'.issues') and (not issue or len(issue.encode())>200):problem(invalid,p+'.issues','issue_blank_or_oversize')
 out.update({'shapeViolations':shape,'shapeViolationCount':shapes,'valueViolations':invalid,'valueViolationCount':constraints,'quoteCount':quotes,'directJSONShapeFits':shapes==0,'visualValidationFits':shapes==0 and constraints==0})
 return out
def diagnosis():
 marker=snap('document-reading-spec0056-q2.started',512)
 need(marker[0]==(SOURCE+' '+MANIFEST+' '+APPROVAL+'\n').encode())
 raw=snap('document-reading-spec0056-q2.private.json',32<<20)
 outer=parse(raw[0]);need(set(outer)=={'sha256','report'} and outer['sha256']==REPORT)
 r=outer['report'];need(hashlib.sha256(canonical(r)).hexdigest()==REPORT)
 need(r['manifest']['source']==SOURCE and hashlib.sha256(canonical(r['manifest'])).hexdigest()==MANIFEST)
 need(r['grant']['source']==SOURCE and r['grant']['approval']==APPROVAL and r['grant']['manifestSHA256']==MANIFEST)
 need(r['core']['decision']=='REJECT')
 results=r['core']['results'];need(len(results)==8 and results[0]['case']=='cards' and results[0]['state']=='rejected' and results[0]['settlement']=='settled')
 need(all(x['state']=='not_sent' for x in results[1:]))
 posts=[e for e in r['wire'] if e['method']=='POST'];gets=[e for e in r['wire'] if e['method']=='GET'];need(len(posts)==1 and len(gets)==6)
 post=posts[0];need(post['path']=='/api/v1/chat/completions' and post['status']==200 and not post['failure'])
 response=parse(base64.b64decode(post['body'],validate=True));choices=response['choices'];need(type(choices) is list and len(choices)==1)
 content=choices[0]['message']['content'];need(type(content) is str and len(content.encode())<=2<<20)
 detail=inspect(content)
 visual=parse(content.encode());boxes=[];empty=[]
 def visit(qs):
  if type(qs) is list:
   for q in qs:
    if type(q) is dict and type(q.get('box')) is list: boxes.append(q['box'])
 for k in HEAD:
  f=visual['header'].get(k)
  if type(f) is dict: visit(f.get('evidence'))
 for i,row in enumerate(visual['rows']):
  visit(row.get('anchor'))
  for k in ROW:
   f=row['fields'].get(k)
   if type(f) is dict:
    visit(f.get('evidence'))
    if f.get('value','').strip() and f.get('evidence')==[]: empty.append({'row':i+1,'field':k})
 flat=[n for box in boxes for n in box]
 detail['boxCount']=len(boxes)
 detail['coordinateCount']=len(flat)
 detail['coordinatesAllIntegers']=all(type(n) in (int,float) and n==int(n) for n in flat)
 detail['coordinatesAllWithinZeroToThousand']=all(0<=n<=1000 for n in flat)
 detail['coordinatesAboveOne']=sum(n>1 for n in flat)
 detail['boxesAboveOne']=sum(any(n>1 for n in box) for box in boxes)
 detail['populatedFieldsWithoutEvidence']=empty
 for box in boxes:
  for j in range(len(box)): box[j]/=1000
 detail['counterfactualScaleOneThousand']=inspect(json.dumps(visual))
 for item in empty: visual['rows'][item['row']-1]['fields'][item['field']]=None
 detail['counterfactualScaleAndNullUncited']=inspect(json.dumps(visual))
 finish=choices[0].get('finish_reason');finish=finish if finish in ('stop','length','content_filter','tool_calls') else 'other'
 need(snap('document-reading-spec0056-q2.started',512)==marker)
 need(snap('document-reading-spec0056-q2.private.json',32<<20)==raw)
 return {'version':'document-reading.q42.reject-diagnosis.v1','status':'VERIFIED','source':SOURCE,'manifestSHA256':MANIFEST,'reportSHA256':REPORT,'reportFileSHA256':raw[2],'markerFileSHA256':marker[2],'snapshotUnchanged':True,'providerCalls':0,'sqlCalls':0,'persistentWritesRequested':0,'storedPOST':1,'storedMetadataGET':6,'postHTTPStatus':200,'finishReason':finish,'content':detail}
if __name__=='__main__':
 try:
  out=diagnosis();b=json.dumps(out,separators=(',',':')).encode();need(len(b)<=32768);sys.stdout.buffer.write(b+b'\n')
 except BaseException:
  print('{"version":"document-reading.q42.reject-diagnosis.v1","status":"UNVERIFIED","reason":"snapshot_or_projection_refused","providerCalls":0,"sqlCalls":0,"persistentWritesRequested":0}');sys.exit(1)
