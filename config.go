package robokassa

type Config struct {
	IdleConnTimeoutSec int
	RequestTimeoutSec  int
	URI                string
	CallbackURI        string
	Shops              Shop
}

type Shop struct {
	Main Credentials
	SBP  Credentials
}

type Credentials struct {
	IsTest bool
	Login  string
	Pass1  string
	Pass2  string
}
