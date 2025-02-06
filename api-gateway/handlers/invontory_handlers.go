package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	pbInventory "github.com/geoo115/E-commerceMicroservices/inventory-service/proto"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// getInventoryServiceClient establishes a gRPC connection to the inventory service.
func getInventoryServiceClient() (pbInventory.InventoryServiceClient, *grpc.ClientConn, error) {
	addr := viper.GetString("inventory_service.address")
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return pbInventory.NewInventoryServiceClient(conn), conn, nil
}

// GetInventory retrieves inventory details for a given product.
func GetInventory(c *gin.Context) {
	productId := c.Param("productId")
	pid, err := strconv.ParseUint(productId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	client, conn, err := getInventoryServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to inventory service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.GetInventory(ctx, &pbInventory.GetInventoryRequest{ProductId: uint32(pid)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateInventory updates inventory details for a product.
func UpdateInventory(c *gin.Context) {
	var req pbInventory.UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, conn, err := getInventoryServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to inventory service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.UpdateStock(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
