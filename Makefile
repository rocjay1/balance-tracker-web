.PHONY: install-frontend dev-frontend build-frontend run-backend build-backend clean

# Frontend Tasks
install-frontend:
	cd frontend && npm install

dev-frontend:
	cd frontend && npm run dev

build-frontend:
	cd frontend && npm run build

# Backend Tasks
run-backend:
	cd backend && go run ./cmd/server

build-backend:
	cd backend && go build -o bin/server ./cmd/server

clean:
	rm -rf frontend/dist backend/bin
