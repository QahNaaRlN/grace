import hashlib,os,subprocess,tempfile,unittest
from pathlib import Path
SCRIPT=Path(__file__).resolve().parents[2]/'deployments/production/real-document-check.sh'
class WrapperGuards(unittest.TestCase):
 def run_case(self,scenario):
  with tempfile.TemporaryDirectory(prefix='grace-0057-wrapper-') as tmp:
   root=Path(tmp);shared=root/'shared';shared.mkdir();sha='a'*40;stage=root/('grace-real-document-check-'+sha);stage.mkdir();log=root/'called'
   script=SCRIPT.read_text().replace('shared=/opt/grace/shared','shared='+str(shared)).replace('/tmp/grace-real-document-check-$source_sha',str(root)+'/grace-real-document-check-$source_sha')
   wrapper=root/'wrapper.sh';wrapper.write_text(script)
   values={'real-document-check':b'#!/bin/sh\nexit 99\n','input.json':b'{}','run.sh':b'synthetic','SOURCE_SHA':sha.encode()}
   for name,raw in values.items():(stage/name).write_bytes(raw)
   binary=hashlib.sha256(values['real-document-check']).hexdigest();bundle=hashlib.sha256(values['input.json']).hexdigest()
   (stage/'SHA256SUMS').write_text(''.join(hashlib.sha256(raw).hexdigest()+'  '+name+'\n' for name,raw in values.items()))
   if scenario=='spent':(shared/'real-document-check-spec0057.started').write_text('consumed')
   if scenario=='report':(shared/'real-document-check-spec0057.private.json').write_text('prior')
   if scenario=='lock_symlink':(shared/'statement-screening.lock').symlink_to(log)
   if scenario=='bundle_hash':bundle='0'*64
   if scenario=='wrong_source':sha='b'*40
   before={p.name:p.read_bytes() for p in shared.iterdir() if p.is_file()}
   result=subprocess.run(['sh',str(wrapper),str(stage),sha,bundle,binary,'c'*40],capture_output=True,timeout=10)
   self.assertNotEqual(result.returncode,0);self.assertFalse(log.exists());self.assertNotIn(b'PRIVATE',result.stdout+result.stderr)
   for name,raw in before.items():self.assertEqual((shared/name).read_bytes(),raw)
   self.assertFalse((shared/'real-document-check-spec0057.started').exists() and scenario!='spent')
 def test_spent_marker(self):self.run_case('spent')
 def test_prior_report(self):self.run_case('report')
 def test_lock_symlink(self):self.run_case('lock_symlink')
 def test_wrong_bundle_hash(self):self.run_case('bundle_hash')
 def test_wrong_source(self):self.run_case('wrong_source')
if __name__=='__main__':unittest.main()
