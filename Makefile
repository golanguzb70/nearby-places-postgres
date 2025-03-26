start-db-not-optimized: 
	docker compose -f devops/docker-compse-psql-1.yml up -d

stop-db-not-optimized: 
	docker compose -f devops/docker-compse-psql-1.yml down

insert-not-optimized: 
	cd insert-to-database && go run main.go 

run-not-optimized-server: 
	cd not-optimized && go run main.go

load-test-not-optimized: 
	k6 run loadtest/not-optimized.js

db-optimized: 
	