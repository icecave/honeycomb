package main

import (
	"log"
	"net/http"

	"github.com/icecave/honeycomb/src/di"
)

func main() {
	defer di.Container.Close()

	// check := flag.Bool(
	// 	"check",
	// 	false,
	// 	"Perform a health-check instead of starting the server.",
	// )
	//
	// flag.Parse()
	//
	// if len(flag.Args()) != 0 {
	// 	flag.Usage()
	// 	os.Exit(1)
	// }

	// success := false
	// if *check {
	// 	status := container.HealthChecker().Check()
	// 	fmt.Println(status.Message)
	// 	success = status.IsHealthy
	// } else {
	// 	success = container.Server().Run() == nil
	// }
	//
	// // @todo logging
	// svr.Logger.Printf("frontend: %s", err)
	// svr.Logger.Printf("Listening on %s", svr.BindAddress)
	//
	// if !success {
	// 	os.Exit(1)
	// }

	logger := di.Container.Get("logger").(*log.Logger)
	server := di.Container.Get("frontend.server").(*http.Server)

	logger.Printf("Listening on %s", server.Addr)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		logger.Fatalln(err)
	}
}
