package main

import (
	"context"
	"log"
	"net"
	"os"

	"family-finance/server/db"
	"family-finance/server/generated"
	"family-finance/server/services"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()

	// Inisialisasi koneksi database
	if err := db.Init(ctx); err != nil {
		log.Fatalf("[Server] Gagal koneksi DB: %v", err)
	}
	defer db.Pool.Close()
	log.Println("[Server] Koneksi DB berhasil")

	// Inisialisasi auto backup mingguan (replace file tiap minggu untuk hemat storage)
	services.InitAutoBackupScheduler(ctx)

	// Buat gRPC server
	grpcServer := grpc.NewServer()

	// Daftarkan semua service
	finance.RegisterAuthServiceServer(grpcServer, &services.AuthServer{})
	finance.RegisterCategoryServiceServer(grpcServer, &services.CategoryServer{})
	finance.RegisterWalletServiceServer(grpcServer, &services.WalletServer{})
	finance.RegisterTransactionServiceServer(grpcServer, &services.TransactionServer{})
	finance.RegisterBudgetServiceServer(grpcServer, &services.BudgetServer{})
	finance.RegisterDebtServiceServer(grpcServer, &services.DebtServer{})
	finance.RegisterSavingServiceServer(grpcServer, &services.SavingServer{})
	finance.RegisterDashboardServiceServer(grpcServer, &services.DashboardServer{})

	// Reflection untuk debugging dengan grpcurl
	reflection.Register(grpcServer)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("[Server] Gagal listen port %s: %v", port, err)
	}

	log.Printf("[Server] gRPC berjalan di port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[Server] Serve error: %v", err)
	}
}
