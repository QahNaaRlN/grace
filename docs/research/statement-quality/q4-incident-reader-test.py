import json,base64,tempfile,os,stat,hashlib
from pathlib import Path
src=Path(__file__).with_name('q4-incident-reader.py').read_text()
g={};exec(compile(src[:src.index('\ntry:\n output=diagnosis()')],'<fixed-reader>','exec'),g)
model,canonical,tag=g['MODEL'],g['CANONICAL'],g['TAG']; params=['structured_outputs','response_format','max_tokens']
manifest={'source':g['SOURCE']}
g['MANIFEST']=hashlib.sha256(g['canonical'](manifest)).hexdigest()
def body(v):return base64.b64encode(json.dumps(v).encode()).decode()
ep={'model_id':model,'tag':tag,'provider_name':'Google','status':0,'context_length':1048576,'max_completion_tokens':65536,'max_prompt_tokens':1048576,'supported_parameters':params,'pricing':{'prompt':'0.000000275','completion':'0.00000165','request':'0','image':'0'},'untrusted':'PRIVATE_PAYLOAD'}
cat={'data':[{'id':model,'canonical_slug':canonical,'architecture':{'input_modalities':['text','image'],'output_modalities':['text']},'supported_parameters':params,'untrusted':'PRIVATE_PAYLOAD'}]}
zdr={'data':[{'model_id':model,'tag':tag,'status':0}]}
report={'manifest':manifest,'grant':{'source':g['SOURCE'],'approval':g['APPROVAL'],'manifestSHA256':g['MANIFEST']},'core':{'decision':'INCONCLUSIVE','proofs':{},'results':[{'state':'not_sent','reason':'preflight','settlement':'not_reserved'} for _ in range(8)]},'budget':None,'wire':[]}
for path,data in zip(['/api/v1/models','/api/v1/endpoints/zdr',g['ENDPOINT']],[cat,zdr,{'data':{'endpoints':[ep]}}]):
 report['wire'].append({'method':'GET','path':path,'requestSHA256':hashlib.sha256(b'').hexdigest(),'status':200,'failure':False,'contentType':'application/json','body':body(data)})
with tempfile.TemporaryDirectory(prefix='q4-reader-control-') as root:
 g['ROOT']=root
 marker=Path(root)/'document-reading-spec0056-q.started'; raw=Path(root)/'document-reading-spec0056-q.private.json'
 marker.write_text(g['SOURCE']+' '+g['MANIFEST']+' '+g['APPROVAL']+'\n');marker.chmod(0o600)
 def save():
  g['REPORT']=hashlib.sha256(g['canonical'](report)).hexdigest()
  raw.write_text(json.dumps({'sha256':g['REPORT'],'report':report}));raw.chmod(0o600)
 save();out=g['diagnosis']();assert out['snapshotUnchanged'] and out['catalogCandidateMatches']==out['zdrCandidateMatches']==1
 assert all(out['endpointCandidateRows'][0][k] for k in ['modelIdMatches','providerNameMatches','statusZero','contextFits','outputFits','inputFits','parametersFit','promptWithinCap','completionWithinCap','requestFeePresent','imageFeePresent','requestFeeWithinZero','imageFeeWithinZero','requestFeeKeyPresent','requestFeeString','imagePriceNumeric'])
 assert not out['endpointCandidateRows'][0]['requestFeeNull'] and not out['endpointCandidateRows'][0]['requestFeeEmptyString'] and not out['endpointCandidateRows'][0]['imagePricePositive']
 assert 'PRIVATE_PAYLOAD' not in json.dumps(out)
 del ep['pricing']['request']; del ep['model_id'];report['wire'][2]['body']=body({'data':{'endpoints':[ep]}});save()
 out=g['diagnosis']();r=out['endpointCandidateRows'][0];assert not r['requestFeePresent'] and not r['requestFeeWithinZero'] and not r['modelIdPresent'] and not r['modelIdMatches']
 raw.chmod(0o644)
 try:g['diagnosis']();raise AssertionError('unsafe mode accepted')
 except g['Refusal']:pass
 raw.chmod(0o600);original=raw.read_bytes();raw.unlink();target=Path(root)/'target';target.write_bytes(original);target.chmod(0o600);raw.symlink_to(target)
 try:g['diagnosis']();raise AssertionError('symlink accepted')
 except OSError:pass
 raw.unlink();raw.write_bytes(original);raw.chmod(0o600)
 marker.write_text('tamper\n')
 try:g['diagnosis']();raise AssertionError('marker tamper accepted')
 except g['Refusal']:pass
print('PASS: fixed snapshot, model/fee predicates, no payload projection, wrong mode, symlink and marker tamper refusals; network=0')
