package functions

import (
  "net/http"
  "sync"
  "io/ioutil"
  "encoding/json"
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
  routeId := r.URL.Path[15:] // /api/functions/
  route, _ := handlerApi.Conf.GetRoute( routeId )
  if route != nil && route.IsService == true {
    defer handlerApi.Logger.Infof( "Post function '%v' failed : existent but not a function", routeId )
    httpResponse.Code = http.StatusPreconditionFailed
    httpResponse.MessageError = "this route is a service, not a function"
    return
  }
  body, err := ioutil.ReadAll( r.Body )
  if err != nil {
    defer handlerApi.Logger.Warningf( "Post function '%v' ; can't read body : %v", routeId, err )
    httpResponse.Code = 500
    httpResponse.MessageError = "the request's body is an invalid"
    return
  }
  var newRoute = itinerary.Route {}
  err = json.Unmarshal( body, &newRoute )
  if err != nil {
    defer handlerApi.Logger.Warningf( "Post function '%v' ; can't parse body : %v", routeId, err )
    httpResponse.Code = 400
    httpResponse.MessageError = "the request's body is an invalid"
    return
  }
  if route != nil {
    route.Mutex.Lock()
    defer route.Mutex.Unlock()
  } 
  handlerApi.Conf.Routes[routeId] = &newRoute
  defer handlerApi.Logger.Warningf( "Post function '%v' executed", routeId )
  httpResponse.Code = http.StatusOK
  httpResponse.Payload = route
}

func ( handlerApi *HandlerApi ) Delete( httpResponse *httpresponse.Response, r *http.Request ) {
  handlerApi.ConfMutext.Lock()
  defer handlerApi.ConfMutext.Unlock()
  routeId := r.URL.Path[15:] // /api/functions/
  route, _ := handlerApi.Conf.GetRoute( routeId )
  if route == nil {
    defer handlerApi.Logger.Infof( "Delete function '%v' failed : non-existent", routeId )
    httpResponse.Code = http.StatusNotFound
    httpResponse.MessageError = "unknow route"
  } else if route.IsService == true {
    defer handlerApi.Logger.Infof( "Delete function '%v' failed : existent but not a function", routeId )
    httpResponse.Code = http.StatusPreconditionFailed
    httpResponse.MessageError = "this route is a service, no a function"
  } else {
    defer handlerApi.Logger.Infof( "Delete function '%v' asked : existent", routeId )
    route.Mutex.Lock()
    defer route.Mutex.Unlock()
    delete( handlerApi.Conf.Routes, routeId )
    httpResponse.Code = http.StatusNoContent
    httpResponse.MessageError = "route deleted"
  }

}

func ( handlerApi *HandlerApi ) Get( httpResponse *httpresponse.Response, r *http.Request ) {
    handlerApi.ConfMutext.RLock()
    defer handlerApi.ConfMutext.RUnlock()
    routeId := r.URL.Path[15:] // /api/functions/
    route, _ := handlerApi.Conf.GetRoute( routeId )
    if route == nil {
      defer handlerApi.Logger.Infof( "Get function '%v' failed : non-existent", routeId )
      httpResponse.Code = http.StatusNotFound
      httpResponse.MessageError = "unknow route"
    } else if route.IsService == true {
      defer handlerApi.Logger.Infof( "Get function '%v' failed : non-existent but not a function", routeId )
      httpResponse.Code = http.StatusPreconditionFailed
      httpResponse.MessageError = "this route is a service, no a function"
    } else {
      defer handlerApi.Logger.Infof( "Get function '%v' asked (existent)", routeId )
      httpResponse.Code = http.StatusOK
      routeToJson := *route
      httpResponse.Payload = routeToJson
    }
}
