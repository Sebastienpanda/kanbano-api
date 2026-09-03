package storage

import (
	"context"
	"log"
	"os"
	"strconv"
)

func ConnectStorage() *Client {
	useSSL, _ := strconv.ParseBool(os.Getenv("GARAGE_USE_SSL"))

	store, err := New(context.Background(), Config{
		Endpoint:      os.Getenv("GARAGE_ENDPOINT"),
		Region:        os.Getenv("GARAGE_REGION"),
		AccessKey:     os.Getenv("GARAGE_ACCESS_KEY"),
		SecretKey:     os.Getenv("GARAGE_SECRET_KEY"),
		Bucket:        os.Getenv("GARAGE_BUCKET"),
		PublicBaseURL: os.Getenv("GARAGE_PUBLIC_BASE_URL"),
		UseSSL:        useSSL,
	})
	if err != nil {
		log.Printf("object storage unavailable, avatar endpoints disabled: %v", err)
		return nil
	}
	log.Println("object storage connected")
	return store
}
