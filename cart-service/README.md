# 🛒 Cart Service

Shopping cart management microservice for the E-commerce platform.

## 📋 Overview

The Cart Service manages user shopping carts with Redis-backed storage for fast, scalable cart operations. It handles adding, removing, and retrieving cart items with real-time updates.

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────┐
│   API Gateway   │───▶│ Cart Service │───▶│ PostgreSQL  │
└─────────────────┘    └──────────────┘    └─────────────┘
                              │
                              ▼
                       ┌─────────────┐
                       │    Redis    │
                       └─────────────┘
```

## 🚀 Features

- **Cart Management** - Add, remove, update cart items
- **Redis Caching** - Fast in-memory cart storage
- **User Isolation** - User-specific cart management
- **Persistence** - PostgreSQL backup for cart data
- **Real-time Updates** - Instant cart modifications
- **Session Handling** - Cart persistence across sessions

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Framework**: gRPC
- **Database**: PostgreSQL
- **Cache**: Redis (primary storage)
- **ORM**: GORM

## 📁 Project Structure

```
cart-service/
├── main.go                 # Service entry point
├── go.mod                  # Go module definition
├── Dockerfile             # Container configuration
├── proto/                 # Protocol buffer definitions
│   ├── cart.proto         # gRPC service definition
│   ├── cart.pb.go         # Generated protobuf code
│   └── cart_grpc.pb.go    # Generated gRPC code
├── models/                # Data models
│   └── cart.go           # Cart and CartItem models
├── services/              # Business logic
│   └── cart_service.go   # Cart service implementation
├── db/                    # Database configuration
│   └── database.go       # PostgreSQL connection setup
├── cache/                 # Caching layer
│   └── redis.go          # Redis client configuration
└── tests/                 # Test files
    └── cart_service_test.go # Unit tests
```

## 🔌 gRPC API

### Service Definition

```protobuf
service CartService {
    rpc GetCart(GetCartRequest) returns (CartResponse) {}
    rpc AddItemToCart(AddItemRequest) returns (CartResponse) {}
    rpc RemoveCartItem(RemoveItemRequest) returns (CartResponse) {}
    rpc ClearCart(ClearCartRequest) returns (CartResponse) {}
}
```

### Endpoints

| Method | Description | Request | Response |
|--------|-------------|---------|----------|
| `GetCart` | Retrieve user cart | `GetCartRequest` | `CartResponse` |
| `AddItemToCart` | Add item to cart | `AddItemRequest` | `CartResponse` |
| `RemoveCartItem` | Remove item from cart | `RemoveItemRequest` | `CartResponse` |
| `ClearCart` | Clear all cart items | `ClearCartRequest` | `CartResponse` |

## 🗄️ Database Schema

### Carts Table
```sql
CREATE TABLE carts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);
```

### Cart Items Table
```sql
CREATE TABLE cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INTEGER REFERENCES carts(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cart_id, product_id)
);
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_HOST` | PostgreSQL host | `localhost` | ✅ |
| `DATABASE_PORT` | PostgreSQL port | `5432` | ✅ |
| `DATABASE_USER` | Database username | `usr` | ✅ |
| `DATABASE_PASSWORD` | Database password | - | ✅ |
| `DATABASE_NAME` | Database name | `ecommerce_cart` | ✅ |
| `DATABASE_SSLMODE` | SSL mode | `disable` | ✅ |
| `REDIS_HOST` | Redis host | `localhost` | ✅ |
| `REDIS_PORT` | Redis port | `6379` | ✅ |
| `REDIS_PASSWORD` | Redis password | - | ❌ |
| `REDIS_DB` | Redis database | `0` | ❌ |
| `CART_SERVICE_PORT` | Service port | `50054` | ❌ |

## 🚀 Getting Started

### Prerequisites
- Go 1.24+
- PostgreSQL 15+
- Redis 6+
- Protocol Buffers compiler

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/geoo115/E-commerceMicroservices.git
cd E-commerceMicroservices/cart-service
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
protoc --go_out=. --go-grpc_out=. proto/cart.proto
```

5. **Run the service**
```bash
go run main.go
```

### Docker

```bash
# Build image
docker build -t cart-service .

# Run container
docker run -p 50054:50054 \
  -e DATABASE_HOST=postgres \
  -e DATABASE_PASSWORD=your_password \
  -e REDIS_HOST=redis \
  cart-service
```

## 🧪 Testing

### Run Tests
```bash
# Unit tests
go test ./tests/...

# With coverage
go test -cover ./tests/...

# Integration tests with Redis
go test -tags integration ./tests/...
```

## 📊 Monitoring

### Metrics
- Cart operations per second
- Redis cache hit/miss rates
- Average cart size
- Response times

### Health Checks
- Redis connectivity
- PostgreSQL connectivity
- gRPC service health

## 🔄 Data Flow

### Cart Operations Flow

1. **Add Item to Cart**
```
Client Request → Cart Service → Redis Cache → PostgreSQL Backup
```

2. **Get Cart**
```
Client Request → Cart Service → Redis Cache (primary) → PostgreSQL (fallback)
```

3. **Cart Persistence**
```
Redis Operations → Async PostgreSQL Sync → Long-term Storage
```

## 🚦 API Examples

### Get User Cart
```bash
grpcurl -plaintext -d '{
  "user_id": 123
}' localhost:50054 cart.CartService/GetCart
```

### Add Item to Cart
```bash
grpcurl -plaintext -d '{
  "user_id": 123,
  "product_id": 456,
  "quantity": 2,
  "price": 29.99
}' localhost:50054 cart.CartService/AddItemToCart
```

### Remove Item from Cart
```bash
grpcurl -plaintext -d '{
  "user_id": 123,
  "product_id": 456
}' localhost:50054 cart.CartService/RemoveCartItem
```

### Clear Cart
```bash
grpcurl -plaintext -d '{
  "user_id": 123
}' localhost:50054 cart.CartService/ClearCart
```

## 🔧 Redis Keys Structure

### Cart Data
```
cart:user:{user_id} → JSON cart data
cart:user:{user_id}:items → Hash of cart items
cart:user:{user_id}:total → Cart total amount
cart:user:{user_id}:updated → Last update timestamp
```

### Example Redis Commands
```bash
# Get cart for user 123
HGETALL cart:user:123:items

# Get cart total
GET cart:user:123:total

# Check cart expiry
TTL cart:user:123
```

## 🔄 Integration

### With API Gateway
- RESTful HTTP endpoints mapped to gRPC calls
- Authentication middleware integration
- Request/response transformation

### With Other Services
- **Product Service**: Product validation and pricing
- **Order Service**: Cart checkout integration
- **Inventory Service**: Stock availability checks

## 🐛 Troubleshooting

### Common Issues

1. **Redis Connection Failed**
   ```
   Solution: Check REDIS_* environment variables
   Verify Redis is running and accessible
   Test with: redis-cli ping
   ```

2. **Cart Not Found**
   ```
   Solution: Check user_id parameter
   Verify cart exists with GetCart call
   Check Redis key expiration
   ```

3. **Item Already in Cart**
   ```
   Solution: Use update quantity instead of add
   Check for duplicate product_id in cart
   ```

## 📈 Performance

### Benchmarks
- Add to cart: ~2ms (Redis)
- Get cart: ~1ms (Redis)
- Clear cart: ~5ms (Redis + DB)

### Optimization Tips
- Use Redis pipelining for bulk operations
- Implement cart expiry for inactive users
- Batch PostgreSQL sync operations

## 🛡️ Security

- **Input Validation**: Product ID and quantity validation
- **User Isolation**: Cart access restricted to owner
- **Data Sanitization**: Prevent injection attacks
- **Rate Limiting**: Prevent cart spam

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/cart-enhancement`)
3. Commit changes (`git commit -m 'Add cart feature'`)
4. Push to branch (`git push origin feature/cart-enhancement`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
