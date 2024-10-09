package robokassa

type Config struct {
	IdleConnTimeoutSec int
	RequestTimeoutSec  int
	IsTest             bool
	Login              string
	Pass1              string
	Pass2              string
	URI                string
	CallbackURI        string
}
