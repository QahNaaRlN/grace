#!/usr/bin/env python3
"""Read public metadata only; no credentials, inference or production writes."""
import concurrent.futures
import datetime
from decimal import Decimal, ROUND_CEILING
import hashlib
import json
import urllib.request

BASE = 'https://openrouter.ai'
PAIRS = [
    ('google/gemini-2.5-flash', 'google-vertex/eu'),
    ('google/gemini-3.1-flash-lite', 'google-vertex/eu'),
    ('inception/mercury-3', 'inception'),
]

def fetch(path):
    with urllib.request.urlopen(BASE + path, timeout=30) as response:
        raw = response.read(8 * 1024 * 1024 + 1)
    if len(raw) > 8 * 1024 * 1024:
        raise ValueError('public_metadata_too_large')
    return json.loads(raw), hashlib.sha256(raw).hexdigest()


def main():
    catalog, catalog_hash = fetch('/api/v1/models')
    zdr, zdr_hash = fetch('/api/v1/endpoints/zdr')
    models = {m['id']: m for m in catalog['data']}
    def candidate(pair):
        model, tag = pair
        m = models[model]
        path = m['links']['details']
        if not path.startswith('/api/v1/models/') or not path.endswith('/endpoints'):
            raise ValueError('unexpected_public_metadata_path')
        data, digest = fetch(path)
        endpoints = [e for e in data['data']['endpoints'] if e['tag'] == tag and e['model_id'] == model]
        if len(endpoints) > 1:
            raise ValueError('ambiguous_endpoint')
        endpoint = endpoints[0] if endpoints else None
        reasons = []
        price = None
        profile_ok = False
        listed_zdr = any(e['model_id'] == model and e['tag'] == tag for e in zdr['data'])
        if endpoint:
            params = set(endpoint.get('supported_parameters', []))
            reasoning = m.get('reasoning', {})
            reasoning_off = not reasoning.get('mandatory', True) and ('none' in reasoning.get('supported_efforts', []) or not reasoning.get('default_enabled', False))
            profile_ok = all(p in params for p in ['structured_outputs', 'response_format', 'max_tokens', 'reasoning']) and reasoning_off and endpoint['context_length'] >= 116000 and (endpoint.get('max_completion_tokens') or 0) >= 16000
            inp = Decimal(endpoint['pricing']['prompt'])
            out = Decimal(endpoint['pricing']['completion'])
            maximum = int(((inp * 100000 + out * 16000) * 1000000).to_integral_value(rounding=ROUND_CEILING))
            price = {'inputUsdPerMillion': str(inp * 1000000), 'outputUsdPerMillion': str(out * 1000000), 'maximumProviderMicrosPerDocument': maximum, 'feesIncluded': False}
            if inp > Decimal('0.0000003') or out > Decimal('0.0000025'):
                reasons.append('price_above_existing_ceiling')
            if endpoint.get('status') != 0:
                reasons.append('endpoint_not_available')
            if not profile_ok:
                reasons.append('profile_not_verified')
        else:
            reasons.append('exact_endpoint_not_listed')
        if not listed_zdr:
            reasons.append('exact_pair_not_in_zdr_snapshot')
        if tag != 'google-vertex/eu':
            reasons.append('different_provider_region_profile_requires_review')
        if model != 'google/gemini-2.5-flash':
            reasons.append('not_in_current_production_allowlist')
        reasons.append('current_s13_quality_unverified')
        return {
            'model': model, 'endpoint': tag, 'canonicalSlug': m['canonical_slug'],
            'modelCreatedUtc': datetime.datetime.fromtimestamp(m['created'], datetime.timezone.utc).isoformat(),
            'source': BASE + path, 'sourceSha256': digest,
            'providerName': endpoint['provider_name'] if endpoint else None,
            'endpointStatus': endpoint.get('status') if endpoint else None,
            'price': price, 'modelReasoning': m.get('reasoning'),
            'profileMetadataCompatible': profile_ok, 'listedZdr': listed_zdr,
            'quality': 'UNVERIFIED', 'latency': 'UNVERIFIED', 'productionEligible': False,
            'reasons': reasons,
        }
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
        candidates = list(pool.map(candidate, PAIRS))
    print(json.dumps({
        'version': 'statement-model-shortlist.v1', 'checkedAtUtc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
        'sources': [{'url': BASE + '/api/v1/models', 'sha256': catalog_hash}, {'url': BASE + '/api/v1/endpoints/zdr', 'sha256': zdr_hash}],
        'inputBound': 100000, 'outputBound': 16000, 'providerCalls': 0,
        'note': 'Public metadata snapshot only. No quality qualification, production policy update or regional residency guarantee.',
        'candidates': candidates,
    }, ensure_ascii=False, indent=2))

if __name__ == '__main__':
    main()
