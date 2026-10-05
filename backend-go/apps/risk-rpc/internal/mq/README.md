# mq

RabbitMQ boundary for risk-rpc.

Published event:

- `risk.warning.created`

The producer is created through `common/xmq`; RabbitMQ downtime degrades to structured logs and does not block risk analysis.
