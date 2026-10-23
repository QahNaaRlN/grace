#!/usr/bin/env python3
"""One explicit fixed-host SSH after local source, CI, manifest and bounds gates."""
import base64
import hashlib
import importlib.util
import json
from pathlib import Path
import shlex
import sys
sys.dont_write_bytecode=True


def module(name, filename):
    spec=importlib.util.spec_from_file_location(name,Path(__file__).with_name(filename))
    m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m


schema=module('s120schema','s120-bootstrap.py')
ci=module('s120ci','verify-screening-source-ci.py')


def bounded(path,limit):
    with Path(path).open('rb') as f: raw=f.read(limit+1)
    schema.require(len(raw)<=limit);return raw


def main():
    source=''
    try:
        # binary, public context, exact reader, authorization, explicit true, CI JSON
        schema.require(len(sys.argv)==7)
        source=sys.argv[3] if schema.source_ok(sys.argv[3]) else ''
        schema.require(source and source==sys.argv[4] and sys.argv[5]=='true')
        schema.require(ci.verified(json.loads(bounded(sys.argv[6],1048576)),source))
        binary=bounded(sys.argv[1],16*1024*1024);context=bounded(sys.argv[2],32768)
        schema.require(binary and hashlib.sha256(context).hexdigest()==schema.CONTEXT_SHA)
        code,check=schema.capture([sys.argv[1],'-mode=check'],b'',10,env={})
        schema.require(code==0 and json.dumps(json.loads(check.decode('utf-8'),object_pairs_hook=schema.unique),sort_keys=True)==json.dumps(schema.manifest(source),sort_keys=True))
        envelope=json.dumps(dict(binary=base64.b64encode(binary).decode(),sha256=hashlib.sha256(binary).hexdigest(),context=context.decode(),readerSource=source)).encode()
        schema.require(len(envelope)<=schema.LIMIT)
        bootstrap=Path(__file__).with_name('s120-bootstrap.py').read_text()
        code,out=schema.capture(['ssh','-o','BatchMode=yes','-o','StrictHostKeyChecking=yes','-o','ConnectTimeout=10','grace-deploy@195.133.195.190','python3 -B -c '+shlex.quote(bootstrap)],envelope,75)
        schema.checked_result(code,out,source)
        sys.stdout.buffer.write(out)
        return 0 if schema.projection(out,source)['status']=='VERIFIED' else 1
    except Exception:
        sys.stdout.buffer.write(schema.failure(source));return 1


if __name__=='__main__':sys.exit(main())
