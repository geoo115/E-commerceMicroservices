# ⭐ Review Service

Product review and rating management microservice for the E-commerce platform.

## 📋 Overview

The Review Service manages product reviews, ratings, and user feedback. It provides functionality for creating, reading, updating, and deleting reviews while maintaining review aggregation and moderation features.

## 🚀 Features

- **Review Management** - CRUD operations for product reviews
- **Rating System** - 5-star rating system with aggregation
- **Review Moderation** - Content filtering and approval workflow
- **User Reviews** - User-specific review history
- **Review Analytics** - Review statistics and insights
- **Event Publishing** - Review events via Kafka

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Framework**: gRPC
- **Database**: PostgreSQL
- **Messaging**: Apache Kafka
- **Cache**: Redis
- **ORM**: GORM

## 📁 Project Structure

```
review-service/
├── main.go                       # Service entry point
├── go.mod                        # Go module definition
├── Dockerfile                   # Container configuration
├── proto/                       # Protocol buffer definitions
│   ├── review.proto             # gRPC service definition
│   ├── review.pb.go             # Generated protobuf code
│   └── review_grpc.pb.go        # Generated gRPC code
├── models/                      # Data models
│   └── review.go               # Review and Rating models
├── services/                    # Business logic
│   └── review_service.go       # Review service implementation
├── db/                          # Database configuration
│   └── database.go             # PostgreSQL connection setup
├── cache/                       # Caching layer
│   └── redis.go                # Redis client configuration
└── tests/                       # Test files
    └── review_service_test.go   # Unit tests
```

## 🔌 gRPC API

### Endpoints

| Method | Description | Request | Response |
|--------|-------------|---------|----------|
| `CreateReview` | Create product review | `CreateReviewRequest` | `ReviewResponse` |
| `GetReview` | Get review by ID | `GetReviewRequest` | `ReviewResponse` |
| `UpdateReview` | Update existing review | `UpdateReviewRequest` | `ReviewResponse` |
| `DeleteReview` | Delete review | `DeleteReviewRequest` | `DeleteResponse` |
| `ListProductReviews` | Get reviews for product | `ListProductReviewsRequest` | `ListReviewsResponse` |
| `GetUserReviews` | Get user's reviews | `GetUserReviewsRequest` | `ListReviewsResponse` |
| `GetProductRating` | Get aggregated rating | `GetProductRatingRequest` | `ProductRatingResponse` |

## 🗄️ Database Schema

### Reviews Table
```sql
CREATE TABLE reviews (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title VARCHAR(255),
    content TEXT,
    status VARCHAR(50) DEFAULT 'published',
    helpful_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, user_id)
);
```

### Product Ratings Table (Aggregated)
```sql
CREATE TABLE product_ratings (
    id SERIAL PRIMARY KEY,
    product_id INTEGER UNIQUE NOT NULL,
    average_rating DECIMAL(3,2) DEFAULT 0.0,
    total_reviews INTEGER DEFAULT 0,
    rating_1_count INTEGER DEFAULT 0,
    rating_2_count INTEGER DEFAULT 0,
    rating_3_count INTEGER DEFAULT 0,
    rating_4_count INTEGER DEFAULT 0,
    rating_5_count INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🚦 API Examples

### Create Review
```bash
grpcurl -plaintext -d '{
  "product_id": 123,
  "user_id": 456,
  "rating": 5,
  "title": "Excellent product!",
  "content": "This product exceeded my expectations. Highly recommended!"
}' localhost:50056 review.ReviewService/CreateReview
```

### Get Product Reviews
```bash
grpcurl -plaintext -d '{
  "product_id": 123,
  "limit": 10,
  "offset": 0,
  "sort_by": "created_at",
  "sort_order": "desc"
}' localhost:50056 review.ReviewService/ListProductReviews
```

## 🔔 Event Publishing

### Review Events
Publishes events to Kafka topics:

1. **review_added** - When new review is created
```json
{
  "event_type": "review_added",
  "review_id": 123,
  "product_id": 456,
  "user_id": 789,
  "rating": 5,
  "timestamp": "2025-08-04T10:00:00Z"
}
```

2. **review_deleted** - When review is removed
```json
{
  "event_type": "review_deleted",
  "review_id": 123,
  "product_id": 456,
  "timestamp": "2025-08-04T10:05:00Z"
}
```

## 📊 Rating Aggregation

### Automatic Rating Calculation
When reviews are added/updated/deleted, the service automatically:
1. Recalculates average rating
2. Updates rating distribution (1-5 stars)
3. Updates total review count
4. Publishes rating update events

### Rating Breakdown
```go
type ProductRating struct {
    ProductID     uint    `json:"product_id"`
    AverageRating float64 `json:"average_rating"`
    TotalReviews  int     `json:"total_reviews"`
    Rating1Count  int     `json:"rating_1_count"`
    Rating2Count  int     `json:"rating_2_count"`
    Rating3Count  int     `json:"rating_3_count"`
    Rating4Count  int     `json:"rating_4_count"`
    Rating5Count  int     `json:"rating_5_count"`
}
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
