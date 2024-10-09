package robokassa

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

type Service struct {
	config *Config
}

const (
	getInvoiceId   = "/Merchant/Indexjson.aspx"
	getPaymentInfo = "/Merchant/WebService/Service.asmx/OpStateExt"
	getPaymentUrl  = "/Merchant/Index/%s"

	callback = "callback"
)

func New(config *Config) *Service {
	return &Service{
		config: config,
	}
}

func buildQueryString(params map[string]string) string {
	v := url.Values{}

	for key, value := range params {
		v.Set(key, value)
	}

	return v.Encode()
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

	hash := sha512.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}

func urlEncode(input string) string {
	return url.QueryEscape(input)
}

func (s *Service) CreatePayment(request PaymentReq) (paymentUrl string, invoiceId string, err error) {
	var response Response

	// Преобразование структуры в JSON-строку
	jsonData, err := json.Marshal(request.Receipt)
	if err != nil {
		return
	}

	// Преобразование JSON-данных в строку
	receipt := urlEncode(string(jsonData))

	value := calculateSHA512(s.config.Login, fmt.Sprint(request.OutSum), fmt.Sprint(request.InvId), receipt, s.config.Pass1)

	data := map[string]string{
		"MerchantLogin":  s.config.Login,
		"OutSum":         fmt.Sprint(request.OutSum),
		"invoiceId":      fmt.Sprint(request.InvId),
		"Receipt":        receipt,
		"SignatureValue": value,
	}

	if s.config.IsTest {
		data["IsTest"] = "1"
	}

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

	return fmt.Sprintf(getPaymentUrl, response.InvoiceID), response.InvoiceID, nil
}

func (s *Service) VerifySignature(receivedSignature string, params SignatureParams) bool {
	switch params.Method {
	case callback:
		expectedSignature := calculateSHA512(fmt.Sprint(params.OutSum), fmt.Sprint(params.InvId), s.config.Pass2)

		if expectedSignature == receivedSignature {
			return true
		}
	}

	return false
}

func (s *Service) GetPaymentInfo(paymentId int64) (*PaymentInfo, error) {
	data := map[string]string{
		"MerchantLogin": s.config.Login,
		"InvoiceID":     fmt.Sprint(paymentId),
		"Signature":     calculateSHA512(s.config.Login, fmt.Sprint(paymentId), s.config.Pass2),
	}

	reqData := buildQueryString(data)

	var response PaymentInfo

	inputs := SendParams{
		Path:       getPaymentInfo,
		HttpMethod: http.MethodGet,
		Response:   &response,
		Data:       reqData,
	}

	if _, err := sendRequest(s.config, &inputs); err != nil {
		return nil, err
	}

	return &response, nil
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
