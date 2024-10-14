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
	InvId       int64   `json:"InvId"`
	OutSum      int64   `json:"OutSum"`
	Description string  `json:"Description"`
	Receipt     Receipt `json:"receipt,omitempty"`
}

type PaymentResp struct {
	InvoiceId    string `json:"invoice_id"`
	Link         string `json:"link"`
	ReqBody      []byte `json:"req_body"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
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

type Receipt struct {
	Items []Item `json:"items"`
}

type Item struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Sum      int64  `json:"sum"`
	Tax      string `json:"tax"`
}

type RecurrentPayment struct {
	InvId         int64
	PreviousInvId int64
	OutSum        int64
	Description   string
}
