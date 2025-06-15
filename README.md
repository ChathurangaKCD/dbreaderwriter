# dbreaderwriter

A simple go program written to verify db/redis connectivity.

## Redis 
Reads/writes every 1 second to the connected redis instance. Provide the following envs.

```env
DB_TYPE=redis
REDIS_HOST=
REDIS_PORT=
REDIS_USER=
REDIS_PASSWORD=
```
