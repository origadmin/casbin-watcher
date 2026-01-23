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

#### SQS-only

```
sqs://my-casbin-queue?region=us-east-1&wait_time_seconds=10
```

#### SNS & SQS

```
snssqs://my-casbin-queue?region=us-west-2&topic_arn=arn:aws:sns:us-west-2:123456789012:casbin-policy-updates
```
