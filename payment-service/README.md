# 💳 Payment Service

Payment processing microservice for the E-commerce platform.

## 📋 Overview

The Payment Service handles payment processing, transaction management, and payment method validation. It integrates with external payment gateways and manages payment lifecycle events.

## 🚀 Features

- **Payment Processing** - Secure payment transaction handling
- **Multiple Payment Methods** - Credit cards, digital wallets, bank transfers
- **Transaction Management** - Payment status tracking and updates
- **Event Publishing** - Payment event notifications via Kafka
- **Refund Processing** - Refund and chargeback handling
- **Payment History** - Transaction history and reporting

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Framework**: gRPC
- **Database**: PostgreSQL
- **Messaging**: Apache Kafka
- **ORM**: GORM

## 📁 Project Structure

```
payment-service/
├── main.go                      # Service entry point
├── go.mod                       # Go module definition
├── Dockerfile                  # Container configuration
├── proto/                      # Protocol buffer definitions
│   ├── payment.proto           # gRPC service definition
│   ├── payment.pb.go           # Generated protobuf code
│   └── payment_grpc.pb.go      # Generated gRPC code
├── models/                     # Data models
│   └── payment.go             # Payment model
├── services/                   # Business logic
│   └── payment_service.go     # Payment service implementation
├── db/                         # Database configuration
│   └── database.go            # PostgreSQL connection setup
├── cache/                      # Caching layer
│   └── redis.go               # Redis client configuration
└── tests/                      # Test files
    └── payment_service_test.go # Unit tests
```

## 🔌 gRPC API

### Endpoints

| Method | Description | Request | Response |
|--------|-------------|---------|----------|
| `ProcessPayment` | Process payment | `ProcessPaymentRequest` | `PaymentResponse` |
| `GetPayment` | Get payment details | `GetPaymentRequest` | `PaymentResponse` |
| `RefundPayment` | Process refund | `RefundPaymentRequest` | `PaymentResponse` |
| `GetPaymentHistory` | Get payment history | `GetPaymentHistoryRequest` | `PaymentHistoryResponse` |

## 🗄️ Database Schema

### Payments Table
```sql
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    payment_method VARCHAR(50) NOT NULL,
    payment_gateway VARCHAR(50),
    transaction_id VARCHAR(255) UNIQUE,
    status VARCHAR(50) DEFAULT 'pending',
    gateway_response JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_HOST` | PostgreSQL host | `localhost` | ✅ |
| `DATABASE_PASSWORD` | Database password | - | ✅ |
| `DATABASE_NAME` | Database name | `ecommerce_payment` | ✅ |
| `KAFKA_BROKER` | Kafka broker address | `kafka:9092` | ✅ |
| `STRIPE_API_KEY` | Stripe API key | - | ❌ |
| `PAYPAL_CLIENT_ID` | PayPal client ID | - | ❌ |

## 🚦 API Examples

### Process Payment
```bash
grpcurl -plaintext -d '{
  "order_id": 123,
  "user_id": 456,
  "amount": 99.99,
  "currency": "USD",
  "payment_method": "credit_card",
  "card_details": {
    "card_number": "4242424242424242",
    "exp_month": 12,
    "exp_year": 2025,
    "cvc": "123"
  }
}' localhost:50055 payment.PaymentService/ProcessPayment
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
