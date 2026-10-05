# task-rpc

`task-rpc` is the unified task center for long-running and asynchronous work.

## Responsibilities

- Create, list and read system tasks.
- Update task status and progress.
- Cancel tasks.
- Write and query task logs.
- Consume RabbitMQ events from `quant.task.events` and synchronize business events into `system_tasks`.

## Run

```bash
cd backend-go
go run ./apps/task-rpc/task.go -f ./apps/task-rpc/etc/task-rpc.yaml
```

RabbitMQ downtime degrades to logs and does not block service startup.
