#!/usr/bin/env python3
"""Проверка exact-source CI без skipped jobs и без inference."""
import json
from pathlib import Path
import re
import sys

JOBS = {'Quality gates', 'PostgreSQL integration gates', 'Build api image', 'Build web image', 'Build pdf-extractor image'}


def verified(data, sha):
    jobs = data.get('jobs', [])
    return bool(re.fullmatch(r'[0-9a-f]{40}', sha) and data.get('headSha') == sha and data.get('status') == 'completed' and data.get('conclusion') == 'success' and len(jobs) == len(JOBS) and {j.get('name') for j in jobs} == JOBS and all(j.get('status') == 'completed' and j.get('conclusion') == 'success' for j in jobs))


if __name__ == '__main__':
    try:
        raw = Path(sys.argv[2]).read_bytes()
        if len(raw) > 1048576 or not verified(json.loads(raw), sys.argv[1]):
            raise ValueError("unverified")
    except Exception:
        print('screening_exact_source_ci_unverified')
        sys.exit(1)
    print('screening_exact_source_ci_all_jobs_passed')
