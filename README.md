# Сервис-фейкер для заполнения базы данных по схеме

**build:** go build -o pg_gen cmd/main.go

**run:** ./pg_gen <filename> <db_dsn>

**Пример использования:** 
./pg_gen examples/valid/no_params postgres://postgres:postgres@localhost:5432/postgres