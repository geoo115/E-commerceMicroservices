# 📊 Inventory Service

Inventory and stock management microservice for the E-commerce platform.

## 📋 Overview

The Inventory Service manages product stock levels, inventory tracking, and stock updates. It processes order events to maintain accurate inventory counts and publishes inventory change events.

## 🚀 Features

- **Stock Management** - Real-time inventory tracking
- **Stock Updates** - Automatic stock deduction on orders
- **Low Stock Alerts** - Notifications for low inventory
- **Event Processing** - Order event consumption
- **Inventory History** - Stock change audit trail
- **Multi-warehouse Support** - Multiple location inventory

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Framework**: gRPC
- **Database**: PostgreSQL
- **Messaging**: Apache Kafka
- **Cache**: Redis
- **ORM**: GORM

## 📁 Project Structure

```
inventory-service/
├── main.go                         # Service entry point
├── go.mod                          # Go module definition
├── Dockerfile                     # Container configuration
├── proto/                         # Protocol buffer definitions
│   ├── inventory.proto            # gRPC service definition
│   ├── inventory.pb.go            # Generated protobuf code
│   └── inventory_grpc.pb.go       # Generated gRPC code
├── models/                        # Data models
│   └── inventory.go              # Inventory model
├── services/                      # Business logic
│   └── inventory_service.go      # Inventory service implementation
├── db/                            # Database configuration
│   └── database.go               # PostgreSQL connection setup
├── cache/                         # Caching layer
│   └── redis.go                  # Redis client configuration
└── tests/                         # Test files
    └── inventory_service_test.go  # Unit tests
```

## 🔌 gRPC API

### Endpoints

| Method | Description | Request | Response |
|--------|-------------|---------|----------|
| `GetInventory` | Get product stock | `GetInventoryRequest` | `InventoryResponse` |
| `UpdateStock` | Update stock level | `UpdateStockRequest` | `InventoryResponse` |
| `CheckAvailability` | Check stock availability | `CheckAvailabilityRequest` | `AvailabilityResponse` |
| `GetLowStockItems` | Get low stock products | `GetLowStockRequest` | `LowStockResponse` |

## 🗄️ Database Schema

### Inventory Table
```sql
CREATE TABLE inventories (
    id SERIAL PRIMARY KEY,
    product_id INTEGER UNIQUE NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    reserved_stock INTEGER NOT NULL DEFAULT 0,
    reorder_level INTEGER DEFAULT 10,
    max_stock INTEGER,
    warehouse_location VARCHAR(255),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🔔 Event Processing

### Order Placed Events
Consumes `order_placed` events to deduct stock:
```go
func handleOrderPlaced(orderEvent OrderPlacedEvent) {
    for _, item := range orderEvent.Items {
        updateStock := &pb.UpdateStockRequest{
            ProductId: item.ProductID,
            Delta:     -item.Quantity, // Deduct stock
        }
        _, err := server.UpdateStock(ctx, updateStock)
        if err != nil {
            log.Printf("Failed to update stock: %v", err)
        }
    }
}
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
