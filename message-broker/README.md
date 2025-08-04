# 🔔 Message Broker Service

Event-driven messaging service for the E-commerce microservices platform using Apache Kafka.

## 📋 Overview

The Message Broker Service handles asynchronous communication between microservices using event-driven architecture. It manages Kafka topics, producers, and consumers for reliable message delivery across the platform.

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Microservices  │───▶│ Message Broker   │───▶│ Apache Kafka    │
│                 │    │                  │    │                 │
│ - Order Service │    │ - Topic Mgmt     │    │ - Topics        │
│ - Payment Svc   │    │ - Producers      │    │ - Partitions    │
│ - Inventory Svc │    │ - Consumers      │    │ - Replication   │
│ - Review Svc    │    │ - Schema Mgmt    │    │ - Persistence   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 🚀 Features

- **Event Publishing** - Reliable message publishing to Kafka topics
- **Event Consumption** - Scalable consumer groups for event processing
- **Topic Management** - Automated topic creation and configuration
- **Schema Management** - Event schema definitions and validation
- **Dead Letter Queues** - Failed message handling
- **Consumer Groups** - Service-specific consumer group management
- **Event Replay** - Historical event processing capability

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Message Broker**: Apache Kafka
- **Client Library**: Confluent Kafka Go
- **Serialization**: JSON

## 📁 Project Structure

```
message-broker/
├── main.go                   # Service entry point
├── go.mod                    # Go module definition
├── Dockerfile               # Container configuration
├── kafka-setup.sh          # Kafka initialization script
├── topics/                  # Topic definitions
│   └── topics.go           # Topic constants and configuration
├── producers/               # Message producers
│   └── producer.go         # Producer implementation
├── consumers/               # Message consumers
│   ├── consumer.go         # Consumer implementation
│   └── 1.md               # Consumer documentation
└── docker-compose.yml      # Kafka infrastructure setup
```

## 📨 Event Topics

### Core Topics

| Topic | Description | Producers | Consumers |
|-------|-------------|-----------|-----------|
| `order_placed` | New order created | Order Service | Inventory, Payment |
| `payment_successful` | Payment completed | Payment Service | Order Service |
| `inventory_updated` | Stock level changed | Inventory Service | Product Service |
| `review_added` | Product review added | Review Service | Product Service |
| `review_deleted` | Product review removed | Review Service | Product Service |
| `wishlist_updated` | User wishlist changed | Review Service | Notification Service |

### Event Schema Examples

#### Order Placed Event
```json
{
  "event_type": "order_placed",
  "timestamp": "2025-08-04T10:00:00Z",
  "order_id": 123,
  "user_id": 456,
  "total_amount": 129.97,
  "items": [
    {
      "product_id": 789,
      "quantity": 2,
      "price": 29.99
    }
  ],
  "shipping_address": {
    "street": "123 Main St",
    "city": "Springfield",
    "country": "US"
  }
}
```

#### Payment Successful Event
```json
{
  "event_type": "payment_successful",
  "timestamp": "2025-08-04T10:05:00Z",
  "order_id": 123,
  "payment_id": "pay_abc123",
  "amount": 129.97,
  "currency": "USD",
  "payment_method": "credit_card"
}
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `KAFKA_BROKER` | Kafka broker address | `kafka:9092` | ✅ |
| `KAFKA_GROUP_ID` | Default consumer group | `ecommerce-group` | ❌ |
| `KAFKA_AUTO_OFFSET_RESET` | Consumer offset behavior | `earliest` | ❌ |
| `KAFKA_ENABLE_AUTO_COMMIT` | Auto commit offsets | `true` | ❌ |

### Kafka Configuration

```go
// Producer configuration
producerConfig := &kafka.ConfigMap{
    "bootstrap.servers": kafkaBroker,
    "acks":             "all",
    "retries":          3,
    "batch.size":       16384,
    "linger.ms":        10,
}

// Consumer configuration
consumerConfig := &kafka.ConfigMap{
    "bootstrap.servers":  kafkaBroker,
    "group.id":          groupID,
    "auto.offset.reset": "earliest",
    "enable.auto.commit": true,
}
```

## 🚀 Getting Started

### Prerequisites
- Go 1.24+
- Apache Kafka
- Zookeeper (for Kafka)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/geoo115/E-commerceMicroservices.git
cd E-commerceMicroservices/message-broker
```

2. **Install dependencies**
```bash
go mod download
```

3. **Start Kafka infrastructure**
```bash
docker-compose up -d zookeeper kafka
```

4. **Setup Kafka topics**
```bash
./kafka-setup.sh
```

5. **Run the service**
```bash
go run main.go
```

### Docker

```bash
# Build image
docker build -t message-broker .

# Run container
docker run --network kafka-network \
  -e KAFKA_BROKER=kafka:9092 \
  message-broker
```

## 📤 Producer API

### Publishing Events

```go
package main

import (
    "github.com/geoo115/E-commerceMicroservices/message-broker/producers"
    "github.com/geoo115/E-commerceMicroservices/message-broker/topics"
)

func publishOrderEvent() {
    event := OrderPlacedEvent{
        OrderID:     123,
        UserID:      456,
        TotalAmount: 129.97,
        Timestamp:   time.Now(),
    }
    
    message, _ := json.Marshal(event)
    err := producers.PublishEvent(topics.OrderPlaced, message)
    if err != nil {
        log.Printf("Failed to publish event: %v", err)
    }
}
```

### Batch Publishing
```go
func publishBatchEvents() {
    events := [][]byte{
        []byte(`{"event": "order_placed", "order_id": 1}`),
        []byte(`{"event": "order_placed", "order_id": 2}`),
    }
    
    err := producers.PublishBatchEvents(topics.OrderPlaced, events)
    if err != nil {
        log.Printf("Batch publish failed: %v", err)
    }
}
```

## 📥 Consumer API

### Event Consumption

```go
package main

import (
    "github.com/geoo115/E-commerceMicroservices/message-broker/consumers"
    "github.com/geoo115/E-commerceMicroservices/message-broker/topics"
)

func startOrderConsumer() {
    handler := func(message []byte) {
        var event OrderPlacedEvent
        if err := json.Unmarshal(message, &event); err != nil {
            log.Printf("Failed to unmarshal event: %v", err)
            return
        }
        
        // Process the order event
        processOrderEvent(event)
    }
    
    // Start consumer with dedicated group
    go consumers.ConsumeEvents(topics.OrderPlaced, "inventory-service-group", handler)
}
```

### Consumer Groups

Each service uses its own consumer group:
- `order-service-group` - Order service consumers
- `payment-service-group` - Payment service consumers
- `inventory-service-group` - Inventory service consumers
- `review-service-group` - Review service consumers

## 🔄 Event Flow Examples

### Order Processing Flow
```
1. Order Service → order_placed → [Inventory Service, Payment Service]
2. Payment Service → payment_successful → [Order Service]
3. Inventory Service → inventory_updated → [Product Service]
4. Order Service → order_confirmed → [Notification Service]
```

### Review Management Flow
```
1. Review Service → review_added → [Product Service]
2. Product Service updates rating
3. Review Service → review_deleted → [Product Service] 
4. Product Service recalculates rating
```

## 🛠️ Topic Management

### Kafka Setup Script
```bash
#!/bin/bash
# kafka-setup.sh

KAFKA_BROKER="localhost:9092"

# Create topics
kafka-topics.sh --create --topic order_placed --bootstrap-server $KAFKA_BROKER --partitions 3 --replication-factor 1
kafka-topics.sh --create --topic payment_successful --bootstrap-server $KAFKA_BROKER --partitions 3 --replication-factor 1
kafka-topics.sh --create --topic inventory_updated --bootstrap-server $KAFKA_BROKER --partitions 3 --replication-factor 1
kafka-topics.sh --create --topic review_added --bootstrap-server $KAFKA_BROKER --partitions 2 --replication-factor 1
kafka-topics.sh --create --topic review_deleted --bootstrap-server $KAFKA_BROKER --partitions 2 --replication-factor 1
kafka-topics.sh --create --topic wishlist_updated --bootstrap-server $KAFKA_BROKER --partitions 2 --replication-factor 1

echo "Topics created successfully!"
```

### Dynamic Topic Creation
```go
// Auto-create topics with default configuration
func createTopicIfNotExists(topic string) error {
    adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
        "bootstrap.servers": kafkaBroker,
    })
    if err != nil {
        return err
    }
    defer adminClient.Close()
    
    topicSpec := kafka.TopicSpecification{
        Topic:             topic,
        NumPartitions:     3,
        ReplicationFactor: 1,
    }
    
    _, err = adminClient.CreateTopics(ctx, []kafka.TopicSpecification{topicSpec})
    return err
}
```

## 📊 Monitoring & Observability

### Metrics
- Message throughput (messages/second)
- Consumer lag per topic
- Producer success/failure rates
- Topic partition distribution

### Health Checks
```go
func healthCheck() bool {
    // Check Kafka connectivity
    producer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": kafkaBroker,
    })
    if err != nil {
        return false
    }
    defer producer.Close()
    
    // Test message production
    testMsg := &kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &healthTopic, Partition: kafka.PartitionAny},
        Value:          []byte("health-check"),
    }
    
    return producer.Produce(testMsg, nil) == nil
}
```

### Kafka UI Integration
Access Kafka UI at `http://localhost:8081` for:
- Topic management
- Consumer group monitoring  
- Message browsing
- Partition information

## 🐛 Troubleshooting

### Common Issues

1. **Kafka Connection Failed**
   ```
   Solution: Check KAFKA_BROKER environment variable
   Verify Kafka is running: docker-compose ps
   Check network connectivity
   ```

2. **Consumer Lag High**
   ```
   Solution: Scale consumer instances
   Check message processing time
   Optimize consumer handler logic
   ```

3. **Message Delivery Failed**
   ```
   Solution: Check producer acknowledgment settings
   Verify topic exists and is accessible
   Monitor broker health
   ```

## 📈 Performance Tuning

### Producer Optimization
```go
producerConfig := &kafka.ConfigMap{
    "bootstrap.servers": kafkaBroker,
    "acks":             "1",           // Balance between speed and reliability
    "batch.size":       32768,         // Larger batches for throughput
    "linger.ms":        50,            // Wait time for batching
    "compression.type": "snappy",      // Compress messages
}
```

### Consumer Optimization
```go
consumerConfig := &kafka.ConfigMap{
    "bootstrap.servers":     kafkaBroker,
    "group.id":             groupID,
    "fetch.min.bytes":      1024,       // Minimum fetch size
    "fetch.max.wait.ms":    500,        // Max wait for fetch
    "max.partition.fetch.bytes": 1048576, // 1MB per partition
}
```

## 🛡️ Security

- **SASL Authentication**: Support for SASL/PLAIN and SASL/SCRAM
- **SSL/TLS**: Encrypted communication
- **ACLs**: Topic-level access control
- **Schema Validation**: Event schema enforcement

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/messaging-enhancement`)
3. Commit changes (`git commit -m 'Add messaging feature'`)
4. Push to branch (`git push origin feature/messaging-enhancement`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
