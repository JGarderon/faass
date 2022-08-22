package functions

import (
  "net/http"
  "sync"
  // "fmt"
  // -----------
  "api"
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
    Code: 500, 
    MessageError: "an unexpected error has occurred", 
  }
  defer httpResponse.Respond( handlerApi.Logger, w ) 
  handlerApi.ConfMutext.RLock()
  if api.VerifyAuthorization( handlerApi.Conf, r ) {
    httpResponse.Code = 401
    httpResponse.MessageError = "you must be authentified"
    handlerApi.ConfMutext.RLock()
    return 
  }
  switch r.Method  {
    case "GET":
      defer handlerApi.ConfMutext.RLock()
      routeId := r.URL.Path[15:] // /api/functions/
      route, _ := handlerApi.Conf.GetRoute( routeId )
      if route == nil {
        httpResponse.Code = 404 
        httpResponse.MessageError = "unknow route"
        handlerApi.Logger.Infof( "Route '%v' asked (non-existent)", routeId )
        return 
      } else if route.IsService == true {
        httpResponse.Code = 412 
        httpResponse.MessageError = "this route is a service, no a function" 
        handlerApi.Logger.Infof( "Route '%v' asked (non-existent)", routeId )
      } else { 
        handlerApi.Logger.Infof( "Route '%v' asked (existent)", routeId )
        handlerApi.Get( w, r ) 
      }
  }

}

func ( handlerApi *HandlerApi ) Get( w http.ResponseWriter, r *http.Request ) {
   
}
