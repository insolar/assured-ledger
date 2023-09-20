package configuration

type TestWalletAPI struct {
	Address string
}

func NewTestWalletAPI() TestWalletAPI {
	return TestWalletAPI{Address: "localhost:5050"}
}
