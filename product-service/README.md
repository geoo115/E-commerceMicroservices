# 📦 Product Service

Product catalog management microservice for the E-commerce platform.

## 📋 Overview

The Product Service manages the product catalog, categories, and product information. It provides CRUD operations for products and categories with efficient search and filtering capabilities.

## 🏗️ Architecture

```
┌─────────────────┐    ┌────────────────┐    ┌─────────────┐
│   API Gateway   │───▶│ Product Service│───▶│ PostgreSQL  │
└─────────────────┘    └────────────────┘    └─────────────┘
                              │
                              ▼
                       ┌─────────────┐
                       │    Redis    │
                       │  (Caching)  │
                       └─────────────┘
```

## 🚀 Features

- **Product Management** - Create, read, update, delete products
- **Category Management** - Hierarchical product categories
- **Product Search** - Search and filter products
- **Inventory Integration** - Product stock information
- **Image Management** - Product image handling
- **Price Management** - Product pricing and discounts
- **Review Integration** - Product review aggregation

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Framework**: gRPC
- **Database**: PostgreSQL
- **Cache**: Redis
- **ORM**: GORM

## 📁 Project Structure

```
product-service/
├── main.go                    # Service entry point
├── go.mod                     # Go module definition
├── Dockerfile                # Container configuration
├── proto/                    # Protocol buffer definitions
│   ├── product.proto         # gRPC service definition
│   ├── product.pb.go         # Generated protobuf code
│   └── product_grpc.pb.go    # Generated gRPC code
├── models/                   # Data models
│   └── product.go           # Product, Category, and Review models
├── services/                 # Business logic
│   └── product_service.go   # Product service implementation
├── db/                       # Database configuration
│   └── database.go          # PostgreSQL connection setup
├── cache/                    # Caching layer
│   └── redis.go             # Redis client configuration
└── tests/                    # Test files
    └── product_service_test.go # Unit tests
```

## 🔌 gRPC API

### Service Definition

```protobuf
service ProductService {
    rpc CreateProduct(CreateProductRequest) returns (ProductResponse) {}
    rpc GetProduct(GetProductRequest) returns (ProductResponse) {}
    rpc UpdateProduct(UpdateProductRequest) returns (ProductResponse) {}
    rpc DeleteProduct(DeleteProductRequest) returns (GenericResponse) {}
    rpc ListProducts(ListProductsRequest) returns (ListProductsResponse) {}
    rpc CreateCategory(CreateCategoryRequest) returns (CategoryResponse) {}
    rpc ListCategories(ListCategoriesRequest) returns (ListCategoriesResponse) {}
}
```

### Endpoints

| Method | Description | Request | Response |
|--------|-------------|---------|----------|
| `CreateProduct` | Create new product | `CreateProductRequest` | `ProductResponse` |
| `GetProduct` | Get product by ID | `GetProductRequest` | `ProductResponse` |
| `UpdateProduct` | Update product | `UpdateProductRequest` | `ProductResponse` |
| `DeleteProduct` | Delete product | `DeleteProductRequest` | `GenericResponse` |
| `ListProducts` | List/search products | `ListProductsRequest` | `ListProductsResponse` |
| `CreateCategory` | Create category | `CreateCategoryRequest` | `CategoryResponse` |
| `ListCategories` | List categories | `ListCategoriesRequest` | `ListCategoriesResponse` |

## 🗄️ Database Schema

### Categories Table
```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    parent_id INTEGER REFERENCES categories(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Products Table
```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    category_id INTEGER REFERENCES categories(id),
    sku VARCHAR(100) UNIQUE,
    weight DECIMAL(8,2),
    dimensions VARCHAR(255),
    brand VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Product Images Table
```sql
CREATE TABLE product_images (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    image_url VARCHAR(500) NOT NULL,
    alt_text VARCHAR(255),
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
| `DATABASE_NAME` | Database name | `ecommerce_products` | ✅ |
| `DATABASE_SSLMODE` | SSL mode | `disable` | ✅ |
| `REDIS_HOST` | Redis host | `localhost` | ❌ |
| `REDIS_PORT` | Redis port | `6379` | ❌ |
| `PRODUCT_SERVICE_PORT` | Service port | `50052` | ❌ |

## 🚀 Getting Started

### Prerequisites
- Go 1.24+
- PostgreSQL 15+
- Redis 6+ (optional)
- Protocol Buffers compiler

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/geoo115/E-commerceMicroservices.git
cd E-commerceMicroservices/product-service
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
protoc --go_out=. --go-grpc_out=. proto/product.proto
```

5. **Run the service**
```bash
go run main.go
```

### Docker

```bash
# Build image
docker build -t product-service .

# Run container
docker run -p 50052:50052 \
  -e DATABASE_HOST=postgres \
  -e DATABASE_PASSWORD=your_password \
  product-service
```

## 🧪 Testing

### Run Tests
```bash
# Unit tests
go test ./tests/...

# With coverage
go test -cover ./tests/...

# Integration tests
go test -tags integration ./tests/...
```

## 📊 Monitoring

### Metrics
- Products created/updated per minute
- Search query performance
- Category distribution
- Top-selling products

### Health Checks
- Database connectivity
- Cache availability
- gRPC service health

## 🚦 API Examples

### Create Product
```bash
grpcurl -plaintext -d '{
  "name": "iPhone 14 Pro",
  "description": "Latest iPhone with Pro features",
  "price": 999.99,
  "category_id": 1,
  "sku": "IPH14PRO128",
  "brand": "Apple"
}' localhost:50052 product.ProductService/CreateProduct
```

### Get Product
```bash
grpcurl -plaintext -d '{
  "product_id": 1
}' localhost:50052 product.ProductService/GetProduct
```

### List Products
```bash
grpcurl -plaintext -d '{
  "category_id": 1,
  "limit": 10,
  "offset": 0,
  "search_query": "iPhone"
}' localhost:50052 product.ProductService/ListProducts
```

### Create Category
```bash
grpcurl -plaintext -d '{
  "name": "Electronics",
  "description": "Electronic devices and gadgets"
}' localhost:50052 product.ProductService/CreateCategory
```

## 🔍 Search & Filtering

### Supported Filters
- **Category**: Filter by category ID
- **Price Range**: Min/max price filtering
- **Brand**: Filter by brand name
- **Status**: Active/inactive products
- **Text Search**: Product name and description

### Search Examples
```go
// Search by text
req := &pb.ListProductsRequest{
    SearchQuery: "smartphone",
    Limit: 20,
}

// Filter by category and price
req := &pb.ListProductsRequest{
    CategoryId: 1,
    MinPrice: 100.00,
    MaxPrice: 500.00,
}

// Combined filters
req := &pb.ListProductsRequest{
    CategoryId: 1,
    Brand: "Apple",
    SearchQuery: "iPhone",
    Status: "active",
}
```

## 🔄 Integration

### With API Gateway
- RESTful endpoints for product catalog
- Product search and filtering
- Category management

### With Other Services
- **Inventory Service**: Stock level integration
- **Review Service**: Product rating aggregation
- **Order Service**: Product information for orders
- **Cart Service**: Product details for cart items

## 🗂️ Data Models

### Product Model
```go
type Product struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    Name        string    `json:"name" gorm:"not null"`
    Description string    `json:"description"`
    Price       float64   `json:"price" gorm:"not null"`
    CategoryID  uint      `json:"category_id"`
    Category    Category  `json:"category" gorm:"foreignKey:CategoryID"`
    SKU         string    `json:"sku" gorm:"unique"`
    Brand       string    `json:"brand"`
    Weight      float64   `json:"weight"`
    Dimensions  string    `json:"dimensions"`
    Status      string    `json:"status" gorm:"default:'active'"`
    Images      []ProductImage `json:"images" gorm:"foreignKey:ProductID"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Category Model
```go
type Category struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    Name        string    `json:"name" gorm:"unique;not null"`
    Description string    `json:"description"`
    ParentID    *uint     `json:"parent_id"`
    Parent      *Category `json:"parent" gorm:"foreignKey:ParentID"`
    Children    []Category `json:"children" gorm:"foreignKey:ParentID"`
    Products    []Product `json:"products" gorm:"foreignKey:CategoryID"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

## 🐛 Troubleshooting

### Common Issues

1. **Product Not Found**
   ```
   Solution: Verify product ID exists
   Check product status (active/inactive)
   Ensure proper permissions
   ```

2. **Category Constraint Error**
   ```
   Solution: Verify category exists before creating product
   Check foreign key constraints
   Ensure category is active
   ```

3. **Duplicate SKU Error**
   ```
   Solution: Generate unique SKU
   Check existing products for SKU conflicts
   Implement SKU validation
   ```

## 📈 Performance

### Benchmarks
- Create product: ~50ms
- Get product: ~10ms (cached), ~25ms (DB)
- Search products: ~100ms (complex queries)
- List categories: ~5ms (cached)

### Optimization Tips
- Use Redis for frequently accessed products
- Implement database indexing on search fields
- Use connection pooling
- Cache category hierarchies

## 🛡️ Security

- **Input Validation**: Strict product data validation
- **SQL Injection**: GORM protection
- **Authorization**: Admin-only create/update/delete
- **Data Sanitization**: Clean user inputs

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/product-enhancement`)
3. Commit changes (`git commit -m 'Add product feature'`)
4. Push to branch (`git push origin feature/product-enhancement`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
