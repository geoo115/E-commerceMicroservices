# Build and run all services using Docker Compose
run:
	docker-compose up --build -d

# Stop and remove all running containers
stop:
	docker-compose down

# Restart all services
restart:
	docker-compose down && docker-compose up --build -d

# View logs for all containers
logs:
	docker-compose logs -f

# Run tests in all services
test:
	go test ./... -v

# Clean up Docker resources
clean:
	docker-compose down -v && docker system prune -f
