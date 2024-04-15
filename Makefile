up:
	docker-compose up --build -d
populate-data:
	./scripts/populate_data.sh
down:
	docker-compose down 
destroy:
	docker-compose down   --volumes --remove-orphans
cleanup_data:
	psql "postgres://postgres:postgres@localhost:5432/library?sslmode=disable" \
	-Atc "delete from borrow_history where id > 0; \
	delete from books where id > 0 ; \
	delete from users where id > 1;"
