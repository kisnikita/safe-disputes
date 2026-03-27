gen_models:
	sqlc generate -f apps/backend/sqlc.yaml
	rm apps/backend/internal/models/db_mock.go 
	rm apps/backend/internal/models/sqlc_query_mock.sql.go
diff_models:
	sqlc diff -f apps/backend/sqlc.yaml || true