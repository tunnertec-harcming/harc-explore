.PHONY: api web speaker test stems render demo

API_DIR=apps/api
WEB_DIR=apps/web
SPK_DIR=apps/speaker

api:
	cd $(API_DIR) && go run ./cmd/server

web:
	cd $(WEB_DIR) && npm run dev -- --host 127.0.0.1 --port 5173

speaker-build:
	cd $(SPK_DIR) && go build -o ../../bin/harc-speaker ./cmd/harc-speaker

speaker-dry: speaker-build
	./bin/harc-speaker -mode sleep -dry-run -duration 1

render: speaker-build
	./bin/harc-speaker -mode sleep -render /tmp/harc-preview.ogg -render-sec 6 -duration 1

test:
	cd $(API_DIR) && go test ./...
	cd $(SPK_DIR) && go test ./...
	cd $(WEB_DIR) && npm run build

stems:
	cd $(API_DIR) && python3 scripts/generate_stems.py

demo:
	@echo "Terminal 1: make api"
	@echo "Terminal 2: make web   -> http://127.0.0.1:5173"
	@echo "Terminal 3: make speaker-dry  (or make render)"
