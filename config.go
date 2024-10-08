package robokassa

type Config struct {
	IdleConnTimeoutSec int
	RequestTimeoutSec  int
	Login              string
	Pass1              string
	Pass2              string
	URI                string
	CallbackURI        string
}
