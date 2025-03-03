package handlers

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/geoo115/E-commerceMicroservices/product-service/proto"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getProductServiceClient() (pb.ProductServiceClient, *grpc.ClientConn, error) {
	addr := viper.GetString("product-service.address")
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := pb.NewProductServiceClient(conn)
	return client, conn, nil
}

func GetProduct(c *gin.Context) {
	id := c.Param("id")
	productID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	client, conn, err := getProductServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to product service"})
		return
	}
	defer conn.Close()

	resp, err := client.GetProduct(context.Background(), &pb.GetProductRequest{Id: uint32(productID)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func CreateProduct(c *gin.Context) {
	var req pb.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, conn, err := getProductServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to product service"})
		return
	}
	defer conn.Close()

	resp, err := client.CreateProduct(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func CreateCategory(c *gin.Context) {
	// CORRECT REQUEST TYPE
	var req pb.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, conn, err := getProductServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to product service"})
		return
	}
	defer conn.Close()

	// USE THE PROPER REQUEST STRUCT
	resp, err := client.CreateCategory(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateProduct updates a product
func UpdateProduct(c *gin.Context) {
	var req pb.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, conn, err := getProductServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to product service"})
		return
	}
	defer conn.Close()

	resp, err := client.UpdateProduct(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DeleteProduct deletes a product
func DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	productID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	client, conn, err := getProductServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to product service"})
		return
	}
	defer conn.Close()

	_, err = client.DeleteProduct(context.Background(), &pb.DeleteProductRequest{Id: uint32(productID)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "product deleted successfully"})
}
