package robokassa

import (
	"encoding/xml"
	"io"
	"time"
)

type SendParams struct {
	HttpCode    int
	Path        string
	HttpMethod  string
	Data        string
	Body        io.Reader
	QueryParams map[string]string
	Response    interface{}
}

type PaymentReq struct {
	InvId          int64  `json:"InvId"`
	MerchantLogin  string `json:"MerchantLogin"`
	OutSum         string `json:"OutSum"`
	Description    string `json:"Description"`
	SignatureValue string `json:"SignatureValue"`
	IncCurrLabel   string `json:"IncCurrLabel,omitempty"`
	PaymentMethods string `json:"PaymentMethods,omitempty"`
}

type Response struct {
	InvoiceID    string `json:"invoiceID"`
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type SignatureParams struct {
	InvId  int64
	OutSum float32
	Method string
}

type PaymentInfo struct {
	XMLName   xml.Name  `xml:"OperationStateResponse"`
	Result    Result    `xml:"Result"`
	State     State     `xml:"State"`
	Info      Info      `xml:"Info"`
	UserField UserField `xml:"UserField"`
}

type Result struct {
	Code        int    `xml:"Code"`
	Description string `xml:"Description"`
}

type State struct {
	Code        int       `xml:"Code"`
	RequestDate time.Time `xml:"RequestDate"`
	StateDate   time.Time `xml:"StateDate"`
}

type Info struct {
	IncCurrLabel  string        `xml:"IncCurrLabel"`
	IncSum        float64       `xml:"IncSum"`
	IncAccount    string        `xml:"IncAccount"`
	PaymentMethod PaymentMethod `xml:"PaymentMethod"`
	OutCurrLabel  string        `xml:"OutCurrLabel"`
	OutSum        float64       `xml:"OutSum"`
	OpKey         string        `xml:"OpKey"`
	BankCardRRN   string        `xml:"BankCardRRN"`
}

type PaymentMethod struct {
	Code        string `xml:"Code"`
	Description string `xml:"Description"`
}

type UserField struct {
	Field []Field `xml:"Field"`
}

type Field struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}
