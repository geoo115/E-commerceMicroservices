#!/bin/bash

echo "Starting Kafka setup..."

# Wait for Kafka to be ready
sleep 5

# Create topics
KAFKA_BROKER="localhost:9092"

TOPICS=("order_placed" "payment_successful" "inventory_updated" "review_added")

for topic in "${TOPICS[@]}"; do
    kafka-topics.sh --create --topic "$topic" --bootstrap-server "$KAFKA_BROKER" --partitions 3 --replication-factor 1
done

echo "Kafka setup completed!"
