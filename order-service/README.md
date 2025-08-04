# 📋 Order Service

Order processing and management microservice for the E-commerce platform.

## 📋 Overview

The Order Service handles the complete order lifecycle from creation to fulfillment. It integrates with inventory, payment, and messaging services to provide a comprehensive order management system.

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────┐
│   API Gateway   │───▶│Order Service │───▶│ PostgreSQL  │
└─────────────────┘    └──────────────┘    └─────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Kafka Events     │
                    │ - order_placed   │
                    │ - order_updated  │
                    └──────────────────┘
```

## 🚀 Features

- **Order Management** - Create, read, update order status
- **Order Items** - Multiple items per order
- **Status Tracking** - Complete order lifecycle tracking
- **Event-Driven** - Kafka integration for order events
- **Payment Integration** - Payment status handling
- **Inventory Integration** - Stock validation and updates
- **User Orders** - User-specific order history

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Framework**: gRPC
- **Database**: PostgreSQL
- **Messaging**: Apache Kafka
- **ORM**: GORM

## 📁 Project Structure

```
order-service/
├── main.go                    # Service entry point
├── go.mod                     # Go module definition
├── Dockerfile                # Container configuration
├── proto/                    # Protocol buffer definitions
│   ├── order.proto           # gRPC service definition
│   ├── order.pb.go           # Generated protobuf code
│   └── order_grpc.pb.go      # Generated gRPC code
├── models/                   # Data models
│   └── order.go             # Order and OrderItem models
├── services/                 # Business logic
│   └── order_service.go     # Order service implementation
├── db/                       # Database configuration
│   └── database.go          # PostgreSQL connection setup
├── cache/                    # Caching layer
│   └── redis.go             # Redis client configuration
└── tests/                    # Test files
    └── order_service_test.go # Unit tests
```

## 🔌 gRPC API

### Service Definition

```protobuf
service OrderService {
    rpc CreateOrder(CreateOrderRequest) returns (OrderResponse) {}
    rpc GetOrder(GetOrderRequest) returns (OrderResponse) {}
    rpc UpdateOrderStatus(UpdateOrderStatusRequest) returns (OrderResponse) {}
    rpc GetUserOrders(GetUserOrdersRequest) returns (GetUserOrdersResponse) {}
    rpc CancelOrder(CancelOrderRequest) returns (OrderResponse) {}
}
```

### Endpoints

| Method | Description | Request | Response |
|--------|-------------|---------|----------|
| `CreateOrder` | Create new order | `CreateOrderRequest` | `OrderResponse` |
| `GetOrder` | Get order by ID | `GetOrderRequest` | `OrderResponse` |
| `UpdateOrderStatus` | Update order status | `UpdateOrderStatusRequest` | `OrderResponse` |
| `GetUserOrders` | Get user's orders | `GetUserOrdersRequest` | `GetUserOrdersResponse` |
| `CancelOrder` | Cancel order | `CancelOrderRequest` | `OrderResponse` |

## 🗄️ Database Schema

### Orders Table
```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    shipping_address_line1 VARCHAR(255),
    shipping_address_line2 VARCHAR(255),
    shipping_city VARCHAR(255),
    shipping_state VARCHAR(255),
    shipping_postal_code VARCHAR(255),
    shipping_country VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Order Items Table
```sql
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 📊 Order Status Flow

```
┌─────────┐    ┌─────────┐    ┌───────────┐    ┌───────────┐
│ pending │───▶│ paid    │───▶│ confirmed │───▶│ shipped   │
└─────────┘    └─────────┘    └───────────┘    └───────────┘
     │              │               │               │
     ▼              ▼               ▼               ▼
┌─────────┐    ┌─────────┐    ┌───────────┐    ┌───────────┐
│cancelled│    │cancelled│    │ cancelled │    │delivered  │
└─────────┘    └─────────┘    └───────────┘    └───────────┘
```

### Status Definitions
- **pending**: Order created, awaiting payment
- **paid**: Payment successful, awaiting confirmation
- **confirmed**: Order confirmed, preparing for shipment
- **shipped**: Order shipped to customer
- **delivered**: Order delivered successfully
- **cancelled**: Order cancelled (any stage)

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_HOST` | PostgreSQL host | `localhost` | ✅ |
| `DATABASE_PORT` | PostgreSQL port | `5432` | ✅ |
| `DATABASE_USER` | Database username | `usr` | ✅ |
| `DATABASE_PASSWORD` | Database password | - | ✅ |
| `DATABASE_NAME` | Database name | `ecommerce_order` | ✅ |
| `DATABASE_SSLMODE` | SSL mode | `disable` | ✅ |
| `KAFKA_BROKER` | Kafka broker address | `kafka:9092` | ✅ |
| `ORDER_SERVICE_PORT` | Service port | `50053` | ❌ |

## 🚀 Getting Started

### Prerequisites
- Go 1.24+
- PostgreSQL 15+
- Apache Kafka
- Protocol Buffers compiler

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/geoo115/E-commerceMicroservices.git
cd E-commerceMicroservices/order-service
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. **Generate protobuf files** (if needed)
```bash
protoc --go_out=. --go-grpc_out=. proto/order.proto
```

5. **Run the service**
```bash
go run main.go
```

## 🚦 API Examples

### Create Order
```bash
grpcurl -plaintext -d '{
  "user_id": 123,
  "items": [
    {
      "product_id": 456,
      "quantity": 2,
      "price": 29.99
    },
    {
      "product_id": 789,
      "quantity": 1,
      "price": 49.99
    }
  ],
  "shipping_address": {
    "address_line1": "123 Main St",
    "city": "Springfield",
    "state": "IL",
    "postal_code": "62701",
    "country": "US"
  }
}' localhost:50053 order.OrderService/CreateOrder
```

### Get Order
```bash
grpcurl -plaintext -d '{
  "order_id": 1
}' localhost:50053 order.OrderService/GetOrder
```

### Update Order Status
```bash
grpcurl -plaintext -d '{
  "order_id": 1,
  "status": "shipped"
}' localhost:50053 order.OrderService/UpdateOrderStatus
```

### Get User Orders
```bash
grpcurl -plaintext -d '{
  "user_id": 123,
  "limit": 10,
  "offset": 0
}' localhost:50053 order.OrderService/GetUserOrders
```

## 🔔 Event Publishing

### Order Events

The service publishes events to Kafka topics:

1. **order_placed** - When order is created
```json
{
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
  "timestamp": "2025-08-04T10:00:00Z"
}
```

2. **order_updated** - When order status changes
```json
{
  "order_id": 123,
  "old_status": "pending",
  "new_status": "paid",
  "timestamp": "2025-08-04T10:05:00Z"
}
```

## 🔄 Integration

### Event Consumption
The service consumes events from:
- **payment_successful** - Updates order status to paid
- **inventory_updated** - Handles stock changes

### With Other Services
- **Payment Service**: Payment status updates
- **Inventory Service**: Stock validation and updates
- **Cart Service**: Order creation from cart
- **User Service**: User information validation

## 🐛 Troubleshooting

### Common Issues

1. **Order Creation Failed**
   ```
   Solution: Check inventory availability
   Verify user exists
   Validate shipping address
   ```

2. **Payment Event Not Received**
   ```
   Solution: Check Kafka connectivity
   Verify payment service is running
   Check consumer group configuration
   ```

3. **Status Update Failed**
   ```
   Solution: Verify order exists
   Check valid status transitions
   Ensure proper permissions
   ```

## 📈 Performance

### Benchmarks
- Create order: ~100ms (includes validation)
- Get order: ~20ms
- Status update: ~30ms (includes event publishing)
- User orders: ~50ms

### Optimization Tips
- Use database indexing on user_id and status
- Implement order caching for frequently accessed orders
- Batch process status updates
- Use connection pooling

## 🛡️ Security

- **Input Validation**: Comprehensive order data validation
- **User Authorization**: Orders accessible only to owner/admin
- **Price Validation**: Prevent price manipulation
- **Status Transition**: Validate legal status changes

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/order-enhancement`)
3. Commit changes (`git commit -m 'Add order feature'`)
4. Push to branch (`git push origin feature/order-enhancement`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
