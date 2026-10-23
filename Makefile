.PHONY: dev down logs test test-go test-web test-jev-eval test-integration test-production build generate-api

dev:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

generate-api:
	cd web && npm run generate:api

test: test-go test-web test-jev-eval

test-go:
	go test ./cmd/... ./internal/...

test-web:
	cd web && npm test -- --run && npm run lint && npm run lint:fsd && npm run format:check && npm run build

test-jev-eval:
	./scripts/test-jev-evaluation.sh

test-integration:
	./scripts/test-pdf-extractor.sh
	./scripts/test-bank-statements.sh
	./scripts/test-installments.sh
	./scripts/test-commitments.sh
	./scripts/test-ux-refinement.sh
	./scripts/test-password-authentication.sh
	./scripts/test-ai-experiment.sh
	./scripts/test-imports.sh
	./scripts/test-authentication.sh
	./scripts/test-financial-events.sh
	./scripts/test-receipts.sh
	./scripts/test-obligations.sh
	./scripts/test-analytics.sh
	./scripts/test-assistant-tools.sh
	./scripts/test-income-relative-periods.sh
	./scripts/test-assistant-router.sh

test-production:
	node scripts/research/check-workflow-syntax.mjs
	go test ./scripts/research/s118-diagnose
	python3 scripts/research/test-statement-s118-diagnosis.py
	./scripts/research/prepare-statement-s118.sh --self-test
	go test ./scripts/research/s117-replay
	./scripts/test-statement-comparison-s117.sh
	python3 scripts/research/test-comparison-s117.py
	go test ./scripts/research/s115-diagnose
	python3 scripts/research/test-statement-s115-diagnosis.py
	./scripts/test-statement-comparison-s114.sh
	python3 scripts/research/test-comparison-s114.py
	python3 scripts/research/test-comparison-observed.py
	go test ./scripts/research/s113-replay
	go test ./scripts/research/s112-diagnose
	python3 scripts/research/test-statement-s112-diagnosis.py
	./scripts/test-statement-comparison-s111.sh
	python3 scripts/research/test-statement-comparison-s111.py
	./scripts/test-statement-screening.sh
	./scripts/test-statement-screening-s16.sh
	./scripts/test-statement-screening-s18.sh
	python3 scripts/research/test-screening-s18-inspection.py
	python3 scripts/research/test-screening-source-ci.py
	python3 scripts/research/test-screening-workflow-modes.py
	python3 scripts/research/test-statement-s19-diagnosis.py
	go test ./scripts/research/s19-replay
	python3 scripts/research/test-screening-s16-inspection.py
	python3 scripts/research/test-screening-s16-reconciliation.py
	./scripts/test-statement-activation.sh
	./scripts/test-restore-readiness.sh
	./scripts/test-commitments-rollout.sh
	./scripts/test-production-deployment.sh
	./scripts/test-offsite-backup.sh

build:
	go build -o bin/grace-api ./cmd/api
	cd web && npm run build
