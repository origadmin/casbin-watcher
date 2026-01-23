# AWS Drivers for Casbin Watcher

This directory contains drivers for using AWS services as a Pub/Sub backend for `casbin-watcher`.

## Supported Drivers

1. **SQS-only (`sqs`)**: A simple driver that uses a single SQS queue for both publishing and subscribing.
2. **SNS & SQS (`snssqs`)**: A more robust, fan-out pattern where messages are published to an SNS topic and consumed
   from an SQS queue subscribed to that topic.

---

## Authentication

All AWS drivers use the default AWS SDK credential chain. This means you can configure credentials via:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`).
2. Shared credentials file (`~/.aws/credentials`).
3. IAM role for an EC2 instance or EKS service account.

---

## 1. SQS-only Driver (`sqs`)

This driver is suitable for simple use cases where you don't need a fan-out architecture. All watcher instances will
publish and subscribe to the same SQS queue.

### Configuration

The driver is configured using a URL. The format is:

```
sqs://<queue_name>?region=<aws_region>&<other_parameters>
```

- `<queue_name>`: The name of your SQS queue.
- `region`: **(Required)** The AWS region where the queue exists.

### Configuration Parameters

| Parameter            | Type       | Default   | Description                                                                                                                                                            | Example                 |
|----------------------|------------|-----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------|
| `region`             | `string`   | `(none)`  | **Required.** The AWS region of the SQS queue.                                                                                                                         | `region=us-east-1`      |
| `wait_time_seconds`  | `integer`  | `20`      | The duration (in seconds) for which the `ReceiveMessage` call waits for a message to arrive in the queue. Enables SQS long polling to reduce costs and empty receives. | `wait_time_seconds=10`  |
| `visibility_timeout` | `integer`  | `30`      | The duration (in seconds) that a received message is hidden from subsequent retrieve requests. Should be longer than your message processing time.                     | `visibility_timeout=60` |
| `close_timeout`      | `duration` | `30s`     | The time to wait for the subscriber to gracefully close, allowing in-flight messages to be processed.                                                                  | `close_timeout=1m`      |
| `marshaler`          | `string`   | `default` | The serialization format for messages. Can be `default` (raw `[]byte`) or `json`.                                                                                      | `marshaler=json`        |

### Usage Example

```go
// watcher.NewWatcher("sqs://my-casbin-queue?region=us-east-1&wait_time_seconds=10")
```

---

## 2. SNS & SQS Driver (`snssqs`)

This is a more advanced and recommended pattern for production. It allows multiple services to consume policy updates by
subscribing their own SQS queues to a central SNS topic (fan-out).

- **Publisher**: Sends messages to a specified SNS Topic ARN.
- **Subscriber**: Receives messages from a specified SQS Queue URL.

You are responsible for creating the SNS topic, the SQS queue, and **subscribing the queue to the topic** in your AWS
environment.

### Configuration

The driver is configured using a URL that specifies the **consumer queue** and provides the **publisher topic** as a
parameter.

```
snssqs://<queue_name>?region=<aws_region>&topic_arn=<sns_topic_arn>&<other_parameters>
```

- `<queue_name>`: The name of your SQS queue that is subscribed to the SNS topic.
- `region`: **(Required)** The AWS region where the SNS topic and SQS queue exist.
- `topic_arn`: **(Required)** The full ARN of the SNS topic to publish messages to.

### Configuration Parameters

This driver supports the **same parameters** as the `sqs` driver for configuring the consumer part of the pattern (the
SQS subscriber).

| Parameter            | Type       | Default   | Description                                            | Example                     |
|----------------------|------------|-----------|--------------------------------------------------------|-----------------------------|
| `region`             | `string`   | `(none)`  | **Required.** The AWS region.                          | `region=us-east-1`          |
| `topic_arn`          | `string`   | `(none)`  | **Required.** The ARN of the SNS topic for publishing. | `topic_arn=arn:aws:sns:...` |
| `wait_time_seconds`  | `integer`  | `20`      | SQS long polling duration.                             | `wait_time_seconds=10`      |
| `visibility_timeout` | `integer`  | `30`      | SQS message visibility timeout.                        | `visibility_timeout=60`     |
| `close_timeout`      | `duration` | `30s`     | Graceful shutdown timeout for the SQS subscriber.      | `close_timeout=1m`          |
| `marshaler`          | `string`   | `default` | Serialization format. Can be `default` or `json`.      | `marshaler=json`            |

### Usage Example

```go
// watcher.NewWatcher("snssqs://my-casbin-queue?region=us-west-2&topic_arn=arn:aws:sns:us-west-2:123456789012:casbin-policy-updates")
```
