package fakeserver

import (
  "log"
  "net/http"
  "time"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "strings"
)

type fakeserver struct {
  server   *http.Server
  objects  map[string]map[string]interface{}
  debug    bool
  running  bool
}

func NewFakeServer(i_port int, i_objects map[string]map[string]interface{}, i_start bool, i_debug bool) *fakeserver {
  serverMux := http.NewServeMux()

  svr := &fakeserver{
    debug: i_debug,
    objects: i_objects,
    running: false,
  }

  serverMux.HandleFunc("/", svr.handle_api_object)

  api_object_server := &http.Server{
    Addr: fmt.Sprintf("127.0.0.1:%d", i_port),
    Handler: serverMux,
  }

  svr.server = api_object_server

  if i_start { svr.StartInBackground() }
  if svr.debug { log.Printf("fakeserver.go: Set up fakeserver: port=%d, debug=%t\n", i_port, svr.debug) }

  return svr
}

func(svr *fakeserver)StartInBackground() {
  go svr.server.ListenAndServe()

  /* Let the server start */
  time.Sleep(1 * time.Second)
  svr.running = true
}

func(svr *fakeserver)Shutdown() {
  svr.server.Close()
  svr.running = false
}

func(svr *fakeserver)Running() bool {
  return svr.running
}

func(svr *fakeserver)GetServer() *http.Server {
  return svr.server
}

func (svr *fakeserver)handle_api_object (w http.ResponseWriter, r *http.Request) {
  var obj map[string]interface{}
  var id string
  var ok bool

  /* Assume this will never fail */
  b, _ := ioutil.ReadAll(r.Body)

  if svr.debug {
    log.Printf("fakeserver.go: Recieved request: %+v\n", r)
    log.Printf("fakeserver.go: Headers:\n")
    for name, headers := range r.Header {
      name = strings.ToLower(name)
      for _, h := range headers {
      log.Printf("fakeserver.go:  %v: %v", name, h)
      }
    }
    log.Printf("fakeserver.go: BODY: %s\n", string(b))
    log.Printf("fakeserver.go: IDs and objects:\n")
    for id, obj := range svr.objects {
      log.Printf("  %s: %+v\n", id, obj)
    }
  }

  path := r.URL.EscapedPath()
  parts := strings.Split(path, "/")
  if svr.debug {
    log.Printf("fakeserver.go: Request received: %s %s\n", r.Method, path)
    log.Printf("fakeserver.go: Split request up into %d parts: %v\n", len(parts), parts)
    if r.URL.RawQuery != "" {
      log.Printf("fakeserver.go: Query string: %s\n", r.URL.RawQuery)
    }
  }
  /* If it was a valid request, there will be three parts
     and the ID will exist */
  if len(parts) == 4 {
    id = parts[3]
    obj, ok = svr.objects[id];
    if svr.debug { log.Printf("fakeserver.go: Detected ID %s (exists: %t, method: %s)", id, ok, r.Method) }

    /* Make sure the object requested exists unless it's being created */
    if r.Method != "POST" && !ok {
      http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
      return
    }
  } else if path != "/api/objects" {
    /* How did something get to this handler with the wrong number of args??? */
    if svr.debug { log.Printf("fakeserver.go: Bad request - got to /api/objects without the right number of args") }
    http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
    return
  } else if path == "/api/objects" && r.Method == "GET" {
    result := make([]map[string]interface{}, 0)
    for _, hash := range svr.objects {
      result = append(result, hash)
    }
    b, _ := json.Marshal(result)
    w.Write(b)
    return
  }

  if r.Method == "DELETE" {
    /* Get rid of this one */
    delete(svr.objects, id)
    if svr.debug { log.Printf("fakeserver.go: Object deleted.\n") }
    return
  }
  /* if data was sent, parse the data */
  if string(b) != "" {
    if svr.debug { log.Printf("fakeserver.go: data sent - unmarshalling from JSON: %s\n", string(b)) }

    err := json.Unmarshal(b, &obj)
    if err != nil {
      /* Failure goes back to the user as a 500. Log data here for
         debugging (which shouldn't ever fail!) */
      log.Fatalf("fakeserver.go: Unmarshal of request failed: %s\n", err);
      log.Fatalf("\nBEGIN passed data:\n%s\nEND passed data.", string(b));
      http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
      return
    } else {
      /* In the case of POST above, id is not yet known - set it here */
      if id == "" {
	if val, ok := obj["id"]; ok {
          id = fmt.Sprintf("%v", val)
	} else if val, ok := obj["Id"]; ok {
          id = fmt.Sprintf("%v", val)
	} else if val, ok := obj["ID"]; ok {
          id = fmt.Sprintf("%v", val)
	} else {
          if svr.debug { log.Printf("fakeserver.go: Bad request - POST to /api/objects without id field") }
          http.Error(w, "POST sent with no id field in the data. Cannot persist this!", http.StatusBadRequest)
	  return
	}
      }

      /* Overwrite our stored test object */
      if svr.debug { log.Printf("fakeserver.go: Overwriting %s with new data:%+v\n", id, obj) }
      svr.objects[id] = obj

      /* Coax the data we were sent back to JSON and send it to the user */
      b, _ := json.Marshal(obj)
      w.Write(b)
      return
    }
  } else {
    /* No data was sent... must be just a retrieval */
    if svr.debug { log.Printf("fakeserver.go: Returning object.\n") }
    b, _ := json.Marshal(obj)
    w.Write(b)
    return
  }

  /* All cases by now should have already returned... something wasn't handled */
  http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
  return
}
