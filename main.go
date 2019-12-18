package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/julienschmidt/httprouter"
)

type CalculateRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

type CalculateResponse struct {
	Result int64 `json:"result"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func RednderJSONWithStatusCode(w http.ResponseWriter, data interface{}, statusCode int) {
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	bytes, _ := json.Marshal(&data)
	_, _ = w.Write(bytes)
}

func RenderStatusBadRequest(w http.ResponseWriter) {
	RednderJSONWithStatusCode(w, getErrorResponse(), http.StatusBadRequest)
}

func getErrorResponse() ErrorResponse {
	return ErrorResponse{
		Error: "Incorrect input",
	}
}

func (r CalculateRequest) Validate() bool {
	return r.A > 0 && r.B > 0
}

func factorial(n int) int64 {
	var result int64 = 1

	for i := int64(1); i <= int64(n); i++ {
		result *= i
	}

	return result
}

func calculate(request CalculateRequest) int64 {
	factorialChan := make(chan int64, 2)

	log.Println(request)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(n int, wg *sync.WaitGroup) {
		defer wg.Done()
		factorialChan <- factorial(n)
	}(request.A, wg)
	wg.Add(1)
	go func(n int, wg *sync.WaitGroup) {
		defer wg.Done()
		factorialChan <- factorial(n)
	}(request.B, wg)

	wg.Wait()
	close(factorialChan)

	var result int64 = 1
	for factorial := range factorialChan {
		result *= factorial
	}

	return result
}

func ValidateCalculateRequestMiddleware(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var request CalculateRequest
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			RenderStatusBadRequest(w)
			return
		}

		r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		if err := json.Unmarshal(body, &request); err != nil {
			RenderStatusBadRequest(w)
			return
		}

		if !request.Validate() {
			RenderStatusBadRequest(w)
			return
		}

		h(w, r, p)
	}
}

func CalculateEndpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request CalculateRequest
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	_ = json.Unmarshal(body, &request)
	log.Println("body ", string(body))

	result := calculate(request)
	RednderJSONWithStatusCode(w, CalculateResponse{result}, http.StatusOK)
}

func main() {
	router := httprouter.New()
	router.POST("/calculate", ValidateCalculateRequestMiddleware(CalculateEndpoint))

	log.Println("starting server on 8989 port")
	log.Fatal(http.ListenAndServe(":8989", router))
}
