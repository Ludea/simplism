// main package
package main

import (
    "encoding/json"
    "github.com/extism/go-pdk"
)

// RequestData structure (from the request)
type RequestData struct {
    Body   string              `json:"body"`
    Header map[string][]string `json:"header"`
    Method string              `json:"method"`
    URI    string              `json:"uri"`
}

// ResponseData structure (for the response)
type ResponseData struct {
    Body   string              `json:"body"`
    Header map[string][]string `json:"header"`
    Code   int                 `json:"code"`
}

//export handle
func handle() {
    // read function request data from the memory
    input := pdk.Input()

    var requestData RequestData
    json.Unmarshal(input, &requestData)
    
    message := "[Service 3️⃣] 🤗 Hello " + requestData.Body
    
    responseData := ResponseData{
        Body:   message,
        Header: map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
        Code:   200,
    }
    jsonResponse, _ := json.Marshal(responseData)

    mem := pdk.AllocateBytes(jsonResponse)
    pdk.OutputMemory(mem)

}

func main() {}
