package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	pbPayment "github.com/geoo115/E-commerceMicroservices/payment-service/proto"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// getPaymentServiceClient establishes a gRPC connection to the payment service.
func getPaymentServiceClient() (pbPayment.PaymentServiceClient, *grpc.ClientConn, error) {
	addr := viper.GetString("payment-service.address")
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return pbPayment.NewPaymentServiceClient(conn), conn, nil
}

// ProcessPayment handles the HTTP request to process a payment.
func ProcessPayment(c *gin.Context) {
	var req pbPayment.ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, conn, err := getPaymentServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to payment service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.ProcessPayment(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetPayment retrieves payment details for a given payment ID.
func GetPayment(c *gin.Context) {
	paymentId := c.Param("paymentId")
	pid, err := strconv.ParseUint(paymentId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment id"})
		return
	}

	client, conn, err := getPaymentServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to payment service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetPayment(ctx, &pbPayment.GetPaymentRequest{PaymentId: pid})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
