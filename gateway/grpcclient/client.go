package grpcclient

import (
	"fmt"
	"os"

	"family-finance/gateway/generated"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	Conn        *grpc.ClientConn
	Auth        finance.AuthServiceClient
	Category    finance.CategoryServiceClient
	Wallet      finance.WalletServiceClient
	Transaction finance.TransactionServiceClient
	Budget      finance.BudgetServiceClient
	Debt        finance.DebtServiceClient
	Saving      finance.SavingServiceClient
	Dashboard   finance.DashboardServiceClient
)

// Init membuka koneksi gRPC ke server dan menginisialisasi semua stub.
func Init() error {
	host := getenv("GRPC_HOST", "localhost")
	port := getenv("GRPC_PORT", "50051")
	addr := fmt.Sprintf("%s:%s", host, port)

	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("grpcclient: gagal dial %s: %w", addr, err)
	}

	Conn = conn
	Auth = finance.NewAuthServiceClient(conn)
	Category = finance.NewCategoryServiceClient(conn)
	Wallet = finance.NewWalletServiceClient(conn)
	Transaction = finance.NewTransactionServiceClient(conn)
	Budget = finance.NewBudgetServiceClient(conn)
	Debt = finance.NewDebtServiceClient(conn)
	Saving = finance.NewSavingServiceClient(conn)
	Dashboard = finance.NewDashboardServiceClient(conn)

	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
