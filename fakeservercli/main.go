package main

import (
  "os"
  "flag"
  "fmt"
  fakeserver "github.com/Mastercard/terraform-provider-restapi/fakeserver"
)


func main() {
  api_server_objects := make(map[string]map[string]interface{})

  port := flag.Int("port", 8080, "The port fakeserver will listen on")
  debug := flag.Bool("debug", false, "Enable debug output of the server")

  flag.Parse()

  svr := fakeserver.NewFakeServer(*port, api_server_objects, false, *debug)

  fmt.Printf("Starting server on port %d...\n", *port)
  fmt.Println("Objects are at /api/objects/{id}")

  internal_server := svr.GetServer()
  err := internal_server.ListenAndServe()
  if nil != err {
    fmt.Printf("Error with the internal TCP server: %s", err)
    os.Exit(1)
  }
}
