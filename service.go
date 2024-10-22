package robokassa

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Service struct {
	config *Config
}

const (
	createPayment    = "/Merchant/Indexjson.aspx"
	getPaymentInfo   = "/Merchant/WebService/Service.asmx/OpStateExt"
	getPaymentUrl    = "/Merchant/Index/%s"
	recurrentPayment = "/Merchant/Recurring"

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

func calculateHash(inputs ...string) string {
	var data string

	for ind, input := range inputs {
		if ind == 0 {
			data = input
			continue
		}

		data += fmt.Sprintf(":%v", input)
	}

	hash := md5.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}

func urlEncode(input string) string {
	return url.QueryEscape(input)
}

func (s *Service) CreatePayment(request PaymentReq) (*PaymentResp, error) {
	var response Response

	// Преобразование структуры в JSON-строку
	jsonData, err := json.Marshal(request.Receipt)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal struct -> string: %w", err)
	}

	// Преобразование JSON-данных в строку
	receipt := urlEncode(string(jsonData))

	value := calculateHash(s.config.Login, fmt.Sprint(request.OutSum), fmt.Sprint(request.InvId), receipt, s.config.Pass1)

	data := map[string]string{
		"MerchantLogin":  s.config.Login,
		"Culture":        "ru",
		"OutSum":         fmt.Sprint(request.OutSum),
		"invoiceId":      fmt.Sprint(request.InvId),
		"Receipt":        receipt,
		"SignatureValue": value,
	}

	if request.IsRecurrent {
		data["Recurring"] = "true"
	}

	if s.config.IsTest {
		data["IsTest"] = "1"
	}

	log.Println("data = ", data)

	reqData := buildQueryString(data)

	inputs := SendParams{
		Path:       createPayment,
		HttpMethod: http.MethodPost,
		Response:   &response,
		Data:       reqData,
	}

	var respBody []byte
	if respBody, err = sendRequest(s.config, &inputs); err != nil {
		return nil, fmt.Errorf("sendRequest: %w", err)
	}

	return &PaymentResp{
		InvoiceId:    response.InvoiceID,
		Link:         fmt.Sprintf(s.config.URI+getPaymentUrl, response.InvoiceID),
		ReqBody:      respBody,
		ErrorCode:    response.ErrorCode,
		ErrorMessage: s.IdentifyErrCode(response.ErrorCode),
	}, nil
}

func (s *Service) IdentifyErrCode(code int) string {
	switch code {
	case 0:
		return ""
	case 25:
		return "магазин не активирован"
	case 26:
		return "Магазин не найден"
	case 29:
		return "Неверный параметр SignatureValue"
	case 30:
		return "Неверный параметр счёта"
	case 31:
		return "Неверная сумма платежа"
	case 33:
		return "Время отведённое на оплату счёта истекло"
	case 34:
		return "Услуга рекуррентных платежей не разрешена магазину"
	case 35:
		return "Неверные параметры для инициализации рекуррентного платежа"
	case 40:
		return "Повторная оплата счета с тем же номером невозможна"
	case 41:
		return "Ошибка на старте операции"
	case 51:
		return "Срок оплаты счета истек"
	case 52:
		return "Попытка повторной оплаты счета"
	case 53:
		return "Счет не найден"
	case 64:
		return "Функционал холдирования средств запрещен для магазина"
	case 65:
		return "Некорректные параметры для холдирования"
	case 20, 28, 21, 32, 22, 36, 23, 37, 24, 43, 27, 500:
		return "Внутренние ошибки сервиса"
	default:
		return "unknown code"
	}
}

func (s *Service) VerifySignature(receivedSignature string, params SignatureParams) bool {
	switch params.Method {
	case callback:
		expectedSignature := calculateHash(params.OutSum, fmt.Sprint(params.InvId), s.config.Pass2)

		if expectedSignature == strings.ToLower(receivedSignature) {
			return true
		}
	}

	return false
}

func (s *Service) GetPaymentInfo(paymentId int64) (*PaymentInfo, []byte, error) {
	data := map[string]string{
		"MerchantLogin": s.config.Login,
		"InvoiceID":     fmt.Sprint(paymentId),
		"Signature":     calculateHash(s.config.Login, fmt.Sprint(paymentId), s.config.Pass2),
	}

	var response PaymentInfo

	inputs := SendParams{
		Path:        getPaymentInfo,
		HttpMethod:  http.MethodGet,
		IsXml:       true,
		Response:    &response,
		QueryParams: data,
	}

	var (
		respBody []byte
		err      error
	)

	if respBody, err = sendRequest(s.config, &inputs); err != nil {
		return nil, respBody, err
	}

	return &response, respBody, nil
}

func (s *Service) RecurrentPayment(request RecurrentPayment) (*PaymentInfo, error) {
	data := map[string]string{
		"MerchantLogin":     s.config.Login,
		"InvoiceID":         fmt.Sprint(request.InvId),
		"PreviousInvoiceID": fmt.Sprint(request.PreviousInvId),
		"OutSum":            fmt.Sprint(request.OutSum),
		"Signature":         calculateHash(s.config.Login, fmt.Sprint(request.OutSum), fmt.Sprint(request.InvId), s.config.Pass1),
	}

	reqData := buildQueryString(data)

	var response PaymentInfo

	inputs := SendParams{
		Path:       recurrentPayment,
		HttpMethod: http.MethodPost,
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

	if inputs.IsXml {
		if err = xml.Unmarshal(respBody, &inputs.Response); err != nil {
			return respBody, fmt.Errorf("can't unmarshall response: '%v'. Err: %w", string(respBody), err)
		}
	} else {
		if err = json.Unmarshal(respBody, &inputs.Response); err != nil {
			return respBody, fmt.Errorf("can't unmarshall response: '%v'. Err: %w", string(respBody), err)
		}
	}

	return
}
