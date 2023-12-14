package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	httpHelper "simplism/helpers/http"
	jsonHelper "simplism/helpers/json"
	simplismTypes "simplism/types"
)

var NotifyDiscoveryServiceOfKillingProcess func(pid int) error

var wasmFunctionHandlerList = map[string]int{}

// discoveryHandler handles the /discovery endpoint in the API.
//
// It takes a WasmArguments object as a parameter and returns an http.HandlerFunc.
// The WasmArguments object contains information about the HTTP port.
// The returned http.HandlerFunc handles incoming HTTP requests to the /discovery endpoint.
// It checks if the request is authorized and if it is a POST request.
// If authorized and a POST request, it processes the information from the request body,
// creates a SimpleProcess struct instance from the JSON body, and stores the process information in the database.
// If there is an error while saving the process information, it returns a 500 Internal Server Error response.
// If the request is not authorized, it returns a 401 Unauthorized response.
// If the request method is not allowed, it returns a 405 Method Not Allowed response.
// This function is a work in progress and handles GET, DELETE, and PUT requests.
func discoveryHandler(wasmArgs simplismTypes.WasmArguments) http.HandlerFunc {
	fmt.Println("🔎 discovery mode activated: /discovery  (", wasmArgs.HTTPPort, ")")

	db, _ := initializeDB(wasmArgs)
	// TODO: look at old records and delete old ones

	// This function is called by the spawn handler (DELETE method), see handle-spawn.go
	notifyForKill := func(pid int) error {
		simplismProcess := getSimplismProcessByPiD(db, pid)

		// test simplismProcess.StopTime
		if simplismProcess.StopTime.IsZero() {
			fmt.Println("⏳ Stop time is not set")
			simplismProcess.StopTime = time.Now()

			err := saveSimplismProcessToDB(db, simplismProcess)
			if err != nil {
				fmt.Println("😡 When updating bucket with the Stop Time", err)

			} else {
				fmt.Println("🙂 Bucket updated with the Stop Time")
			}
			return err

		} else {
			fmt.Println("⏳ Stop time:", simplismProcess.StopTime)
			fmt.Println("✋ This process is already killed")
		}

		return nil

	}
	NotifyDiscoveryServiceOfKillingProcess = notifyForKill

	return func(response http.ResponseWriter, request *http.Request) {

		authorised := httpHelper.CheckDiscoveryToken(request, wasmArgs)

		switch {
		// triggered when a simplism process contacts the discovery endpoint
		case request.Method == http.MethodPost && authorised == true:

			body := httpHelper.GetBody(request) // process information from simplism POST request

			// store the process information in the database
			simplismProcess, _ := jsonHelper.GetSimplismProcesseFromJSONBytes(body)
			err := saveSimplismProcessToDB(db, simplismProcess)

			//simplismProcess.ServiceName

			if err != nil {
				fmt.Println("😡 When updating bucket", err)
				response.WriteHeader(http.StatusInternalServerError)
			} else {
				response.WriteHeader(http.StatusOK)

				/* Call a function from the discovery service
				   ------------------------------------------
					if there is a new simplism function process contact
					- create a new handler to handle the requests (kind of reverse proxy)
					- only if the handler doesn't exist

					if the process service name is "hello" and listening on port 9090
					if the process spawaner is listening on port 8080

					when you call http://localhost:8080/function/hello
					a request will be sent to http://localhost:9090/function/hello

				*/

				if wasmFunctionHandlerList[simplismProcess.ServiceName] == 0 {
					wasmFunctionHandlerList[simplismProcess.ServiceName] = simplismProcess.PID

					//fmt.Println("🔥🔥🔥", simplismProcess.PID, simplismProcess.ServiceName)

					http.HandleFunc("/service/"+simplismProcess.ServiceName, func(response http.ResponseWriter, request *http.Request) {

						host, _, _ := net.SplitHostPort(request.Host)

						// make an HTTP request to the simplismservice
						//! https? handled by the spawner
						client := &http.Client{}
						body := httpHelper.GetBody(request)
						requestToSpawnedProcess, _ := http.NewRequest(request.Method, "http://"+host+":"+simplismProcess.HTTPPort, bytes.NewBuffer(body))
						requestToSpawnedProcess.Header = request.Header

						// Send the request
						responseFromSpawnedProcess, err := client.Do(requestToSpawnedProcess)
						if err != nil {
							fmt.Println("😡 When making the HTTP request", err)
						}
						defer responseFromSpawnedProcess.Body.Close()
						// Read the response body
						responseBodyFromSpawnedProcess, err := io.ReadAll(responseFromSpawnedProcess.Body)
						if err != nil {
							fmt.Println("😡 Error reading response body:", err)
							return
						}

						response.WriteHeader(responseFromSpawnedProcess.StatusCode)
						response.Write(responseBodyFromSpawnedProcess)

					})
				}

			}

		case request.Method == http.MethodGet && authorised == true:

			// get the list of the services that are running
			processes := getSimplismProcessesListFromDB(db)
			jsonString, err := json.Marshal(processes)

			if err != nil {
				fmt.Println("😡 When marshalling", err)
				response.WriteHeader(http.StatusInternalServerError)
			} else {
				response.WriteHeader(http.StatusOK)
				response.Write(jsonString)
			}

		case request.Method == http.MethodPut && authorised == true:
			// TODO update the Information field of the service
			// if the token is propagated, the service will be able to PUT information

		// to kill a service, see the admin handler

		case authorised == false:
			response.WriteHeader(http.StatusUnauthorized)
			//fmt.Println("😡 You're not authorized")
			//fmt.Fprintln(response, "😡 You're not authorized")
			response.Write([]byte("😡 You're not authorized"))

		default:
			response.WriteHeader(http.StatusMethodNotAllowed)
			response.Write([]byte("😡 Method not allowed"))
			//fmt.Fprintln(response, "😡 Method not allowed")
		}

	}

}
