package services

import (
  "net/http"
  "io/ioutil"
  "encoding/json"
  "time"
  "sync"
  // "fmt"
  // -----------
  "api"
  "itinerary"
  "configuration"
  "httpresponse"
  "logger"
)

type HandlerApi struct {
  Logger *logger.Logger
  ConfMutext *sync.RWMutex
  Conf *configuration.Conf
}

func ( handlerApi HandlerApi ) ServeHTTP ( w http.ResponseWriter, r *http.Request ) {
  httpResponse := httpresponse.Response {
    Code: http.StatusInternalServerError,
    MessageError: "an unexpected error has occurred",
  }
  defer httpResponse.Respond( handlerApi.Logger, w )
  handlerApi.ConfMutext.RLock()
  auth := api.VerifyAuthorization( handlerApi.Conf, r )
  handlerApi.ConfMutext.RUnlock()
  if auth {
    httpResponse.Code = http.StatusUnauthorized
    httpResponse.MessageError = "you must be authentified"
    return
  }
  switch r.Method  {
    case http.MethodGet:
      handlerApi.Get( &httpResponse, r )
    case http.MethodPost:
      handlerApi.Post( &httpResponse, r )
    case http.MethodDelete:
      handlerApi.Delete( &httpResponse, r )
    default:
      httpResponse.Code = http.StatusMethodNotAllowed
      httpResponse.MessageError = "HTTP verb incorrect"
  }
}

func ( handlerApi *HandlerApi ) Post( httpResponse *httpresponse.Response, r *http.Request ) {
  handlerApi.ConfMutext.Lock()
  defer handlerApi.ConfMutext.Unlock()
  routeId := r.URL.Path[14:] // /api/services/
  route, _ := handlerApi.Conf.GetRoute( routeId )
  if route != nil && route.IsService != true {
    defer handlerApi.Logger.Infof( "Post service '%v' failed : existent but not a service", routeId )
    httpResponse.Code = http.StatusPreconditionFailed
    httpResponse.MessageError = "this route is a function, no a service"
    return 
  }

  body, err := ioutil.ReadAll( r.Body )
  if err != nil { 
    defer handlerApi.Logger.Warningf( "Post service '%v' ; can't read body : %v", routeId, err )
    httpResponse.Code = http.StatusInternalServerError 
    httpResponse.MessageError = "the request's body is an invalid"
    return 
  } 
  var newRoute = itinerary.Route {}
  err = json.Unmarshal( body, &newRoute ) 
  if err != nil { 
    defer handlerApi.Logger.Warningf( "Post function '%v' ; can't parse body : %v", routeId, err )
    httpResponse.Code = http.StatusBadRequest 
    httpResponse.MessageError = "the request's body is an invalid"
    return 
  } 
  if route != nil {
    route.Mutex.Lock()
    defer route.Mutex.Unlock()
    cId := route.Id 
    if cId != "" {
      _, err := handlerApi.Conf.Containers.Stop( route )
      if err != nil {
        handlerApi.Logger.Errorf( "Delete service '%v' (cId %v) not stopped - maybe he is still active ?", routeId, cId )
      } 
      handlerApi.Logger.Warningf( "Delete service '%v' (cId %v) stopped", routeId, cId )
      time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
      _, err = handlerApi.Conf.Containers.Remove( route )
      if err != nil {
        handlerApi.Logger.Errorf( "Delete service '%v' (cId %v) not terminated", routeId, cId )
      } else {
        handlerApi.Logger.Warningf( "Delete service '%v' (cId %v) terminated", routeId, cId )
      } 
    }
  } 
  defer handlerApi.Logger.Warningf( "Post service '%v' executed", routeId )
  handlerApi.Conf.Routes[routeId] = &newRoute 
  httpResponse.Code = http.StatusOK 
  httpResponse.Payload = nil 
}

func ( handlerApi *HandlerApi ) Delete( httpResponse *httpresponse.Response, r *http.Request ) {
  handlerApi.ConfMutext.Lock()
  defer handlerApi.ConfMutext.Unlock()
  routeId := r.URL.Path[14:] // /api/services/
  route, _ := handlerApi.Conf.GetRoute( routeId )
  if route == nil {
    defer handlerApi.Logger.Infof( "Delete service '%v' failed : non-existent", routeId )
    httpResponse.Code = http.StatusNotFound 
    httpResponse.MessageError = "unknow route"
    return 
  } else if route.IsService != true {
    defer handlerApi.Logger.Infof( "Delete service '%v' failed : existent but not a service", routeId )
    httpResponse.Code = http.StatusPreconditionFailed
    httpResponse.MessageError = "this route is a function, no a service"
    return 
  }
  route.Mutex.Lock()
  defer route.Mutex.Unlock()
  cId := route.Id 
  if cId != "" {
    _, err := handlerApi.Conf.Containers.Stop( route )
    if err != nil {
      handlerApi.Logger.Errorf( "Delete service '%v' (cId %v) not stopped - maybe he is still active ?", routeId, cId )
    } 
    handlerApi.Logger.Warningf( "Delete service '%v' (cId %v) stopped", routeId, cId )
    time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
    _, err = handlerApi.Conf.Containers.Remove( route )
    if err != nil {
      handlerApi.Logger.Errorf( "Delete service '%v' (cId %v) not terminated", routeId, cId )
    } else {
      handlerApi.Logger.Warningf( "Delete service '%v' (cId %v) terminated", routeId, cId )
    } 
  }
  handlerApi.Logger.Warningf( "Delete service '%v' (cId %v) executed", routeId, cId )
  handlerApi.Logger.Warningf( "Delete service '%v' removed from routes", routeId )
  httpResponse.Code = http.StatusOK 
  httpResponse.Payload = nil 
}

func ( handlerApi *HandlerApi ) Get( httpResponse *httpresponse.Response, r *http.Request ) {
  handlerApi.ConfMutext.RLock()
  defer handlerApi.ConfMutext.RUnlock()
    routeId := r.URL.Path[14:] // /api/services/
    route, _ := handlerApi.Conf.GetRoute( routeId )
  if route == nil {
    defer handlerApi.Logger.Infof( "Get service '%v' failed : non-existent", routeId )
    httpResponse.Code = http.StatusNotFound 
    httpResponse.MessageError = "unknow route"
    return 
  } else if route.IsService != true {
    defer handlerApi.Logger.Infof( "Get service '%v' failed : existent but not a function", routeId )
    httpResponse.Code = http.StatusBadRequest
    httpResponse.MessageError = "this route is not a service"
    return 
  }
  defer handlerApi.Logger.Infof( "Get service '%v' asked (existent)", routeId )
  httpResponse.Code = http.StatusOK 
  routeToJson := *route
  httpResponse.Payload = routeToJson 
}

