up-w:
	docker compose up --build --watch

up-d:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

logs-tail:
	docker compose logs -f