package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type SendParams struct {
	HttpCode    int
	Path        string
	HttpMethod  string
	Data        string
	Token       string
	AuthNeed    bool
	Body        io.Reader
	QueryParams map[string]string
	Response    interface{}
}

type Service struct {
	config *Config
}

const (
	getInvoiceId  = "/Merchant/Indexjson.aspx"
	getPaymentUrl = "/Merchant/Index/%s"
)

func New(config *Config) *Service {
	return &Service{
		config: config,
	}
}

type Config struct {
	IdleConnTimeoutSec int
	RequestTimeoutSec  int
	Login              string
	Pass1              string
	Pass2              string
	URI                string
}

func calculateSHA512(inputs ...string) string {
	var data string

	for ind, input := range inputs {
		if ind == 0 {
			data = input
			continue
		}

		data += fmt.Sprintf(":%v", input)
	}

	log.Println("data = ", data)

	hash := sha512.New()
	hash.Write([]byte(data))

	/*hash := md5.New()
	hash.Write([]byte(data))*/

	return hex.EncodeToString(hash.Sum(nil))
}

func buildQueryString(params map[string]string) string {
	v := url.Values{}

	for key, value := range params {
		v.Set(key, value)
	}

	return v.Encode()
}

type Response struct {
	InvoiceID    string `json:"invoiceID"`
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func main() {
	conf := Config{
		IdleConnTimeoutSec: 60,
		RequestTimeoutSec:  21,
		Login:              "Lunaro",
		Pass1:              "VBxxag7d8a0OVJ4Qc7Ez",
		URI:                "https://auth.robokassa.ru",
	}

	service := New(&conf)

	sum := 1

	receipt := "%7B%22items%22%3A%5B%7B%22name%22%3A%22%D0%A2%D0%B5%D1%85%D0%BD%D0%B8%D1%87%D0%B5%D1%81%D0%BA%D0%B0%D1%8F+%D0%B4%D0%BE%D0%BA%D1%83%D0%BC%D0%B5%D0%BD%D1%82%D0%B0%D1%86%D0%B8%D1%8F+%D0%BF%D0%BE+Robokassa%22%2C%22quantity%22%3A1%2C%22sum%22%3A1%2C%22tax%22%3A%22none%22%7D%5D%7D"

	value := calculateSHA512(conf.Login, fmt.Sprint(sum), "", conf.Pass1, receipt)

	log.Println("value = ", value)

	data := map[string]string{
		"MerchantLogin":  conf.Login,
		"OutSum":         fmt.Sprint(sum),
		"Receipt":        receipt,
		"SignatureValue": value,
	}

	purl, invoiceId, err := service.GetPaymentUrl(data)
	if err != nil {
		log.Println("err = ", err)
		return
	}

	log.Println("purl = ", purl)
	log.Println("invoiceId = ", invoiceId)

}

func (s *Service) GetPaymentUrl(data map[string]string) (paymentUrl string, invoiceId string, err error) {
	var response Response

	reqData := buildQueryString(data)

	inputs := SendParams{
		Path:       getInvoiceId,
		HttpMethod: http.MethodPost,
		Response:   &response,
		Data:       reqData,
	}

	if _, err = sendRequest(s.config, &inputs); err != nil {
		return
	}

	log.Printf("response = %+v", response)

	return
}

func sendRequest(config *Config, inputs *SendParams) (respBody []byte, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("robokassa! SendRequest: %v", err)
		}
	}()

	baseURL, err := url.Parse(config.URI)
	if err != nil {
		return respBody, fmt.Errorf("can't parse URI from config: %w", err)
	}

	// Добавляем путь из inputs.Path к базовому URL
	baseURL.Path += inputs.Path

	// Устанавливаем параметры запроса из queryParams
	query := baseURL.Query()
	for key, value := range inputs.QueryParams {
		query.Set(key, value)
	}

	baseURL.RawQuery = query.Encode()

	finalUrl := baseURL.String()

	log.Println("finalUrl = ", finalUrl)

	req, err := http.NewRequest(inputs.HttpMethod, finalUrl, bytes.NewBuffer([]byte(inputs.Data)))
	if err != nil {
		return respBody, fmt.Errorf("can't create request! Err: %s", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := http.Client{
		Transport: &http.Transport{
			IdleConnTimeout: time.Second * time.Duration(config.IdleConnTimeoutSec),
		},
		Timeout: time.Second * time.Duration(config.RequestTimeoutSec),
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return respBody, fmt.Errorf("can't do request! Err: %s", err)
	}
	defer resp.Body.Close()

	inputs.HttpCode = resp.StatusCode

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return respBody, fmt.Errorf("can't read response body! Err: %w", err)
	}

	if resp.StatusCode == http.StatusInternalServerError {
		return respBody, fmt.Errorf("error: %v", string(respBody))
	}

	log.Println("resp = ", string(respBody))

	if err = json.Unmarshal(respBody, &inputs.Response); err != nil {
		return respBody, fmt.Errorf("can't unmarshall response: '%v'. Err: %w", string(respBody), err)
	}

	return
}
