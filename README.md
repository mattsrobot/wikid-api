# Setup MySQL

Firstly install your MySQL server locally (default port 3306) and configure it with a root user with a password (I have used root:Password123).

Create a database chat_users_db in your MySQL server.

Run your migrations

Install the go migrate CLI tool:

https://github.com/golang-migrate/migrate/tree/master/cmd/migrate

Verify you have a ```migrate``` tool available in your ```$PATH```

Create a database called chat_users_db, then run the migration:

```
migrate -database "mysql://root:password@tcp(127.0.0.1:3306)/chat_users_db" -path ./db/migrations/ up
```

This should create all the tables in the migrations folder.

If the migrations were successful, the file (e.g 000026_create_messages.up.sql) in the migrations folder with the largest number should appear in the schema_migrations table.

# Setup .env file

```
AES_IV="my16digitIvKey12"
AES_KEY="umzwBkl86iYOmoIIuWs5frDe8MyCyh6O"
JWT_SECRET="kaospdkapsodkapdkapsd"
SALT="asdioasjdojasiodjasoidaijosdaoisaj"
DATABASE_URL="root:password@tcp(127.0.0.1:3306)/chat_users_db?parseTime=true"
READ_REDIS_URL="redis://localhost:6379/0"
WRITE_REDIS_URL="redis://localhost:6379/0"
EXOTIC_FQN="wikid.app"
PORT=3003
WEB_ENV="http://localhost:3000"
PRIVATE_WS_INTERNAL_API="http://localhost:3006/v1/internal"
```

# Run the api server

Navigate to api_hot folder and run

```go run api_hot.go```

# Run the ws server

Navigate to api_ws folder and run

```go run api_ws.go```

# Run the worker

Navigate to scheduler folder and run

```go run scheduler.go```
