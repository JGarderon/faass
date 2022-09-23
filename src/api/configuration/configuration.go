package configuration

import (
  "net/http"
  "io/ioutil"
  "encoding/json"
  "sync"
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
    Code: http.StatusInternalServerError,
    MessageError: "an unexpected error has occurred",
  }
  defer httpResponse.Respond( handlerApi.Logger, w )
  handlerApi.ConfMutext.RLock()
  auth := api.VerifyAuthorization( handlerApi.Conf, r )
  handlerApi.ConfMutext.RUnlock()
  if auth != true {
    httpResponse.Code = http.StatusUnauthorized
    httpResponse.MessageError = "you must be authentified"
    return
  }
  switch r.Method  {
    case http.MethodGet:
      handlerApi.Get( &httpResponse, r )
    case http.MethodPatch:
      handlerApi.Patch( &httpResponse, r )
    default:
      httpResponse.Code = http.StatusMethodNotAllowed
      httpResponse.MessageError = "HTTP verb incorrect"
   }
}

func ( handlerApi *HandlerApi ) Patch( httpResponse *httpresponse.Response, r *http.Request ) {
  handlerApi.ConfMutext.Lock()
  defer handlerApi.ConfMutext.Unlock()
  defer handlerApi.Logger.Infof( "Patch conf asked" )
  if contentType := r.Header.Get("Content-type"); contentType != "application/json" {
    httpResponse.Code = http.StatusBadRequest
    httpResponse.MessageError = "you must have 'application/json' content-type header"
    return
  }
  body, err := ioutil.ReadAll( r.Body )
  if err != nil {
    defer handlerApi.Logger.Warningf( "Patch conf asked ; can't read body : %v", err )
    httpResponse.Code = http.StatusInternalServerError
    httpResponse.MessageError = "the request's body is an invalid"
    return
  }
  var f interface{}
  err = json.Unmarshal( body, &f )
  if err != nil {
    defer handlerApi.Logger.Warningf( "Patch conf asked ; can't parse body : %v", err )
    httpResponse.Code = http.StatusBadRequest
    httpResponse.MessageError = "the request's body is an invalid"
    return
  }
  o := f.( map[string]interface{} )
  for key, value := range o {
    switch key {
    case "DelayCleaningContainers":
      switch value.(type) {
      case float64:
        delay := int(value.(float64))
        if delay >= configuration.ConfDelayCleaningContainersMin && delay <= configuration.ConfDelayCleaningContainersMax {
          defer handlerApi.Logger.Warningf( "Patch conf executed, delay changed ; new value : %v", delay )
          handlerApi.Conf.DelayCleaningContainers = delay
          continue
        } else {
          httpResponse.Code = http.StatusBadRequest
          httpResponse.MessageError = "value of delay invalid : int between 5 and 60 (seconds)"
          return
        }
      default:
        httpResponse.Code = http.StatusBadRequest
        httpResponse.MessageError = "type's value of delay invalid"
        return
      }
    default:
      httpResponse.Code = http.StatusNotImplemented
      httpResponse.MessageError = "at least one key is invalid"
      return
    }
  }
  httpResponse.Code = http.StatusAccepted
}

func ( handlerApi *HandlerApi ) Get( httpResponse *httpresponse.Response, r *http.Request ) {
  handlerApi.ConfMutext.RLock()
  defer handlerApi.ConfMutext.RUnlock()
  defer handlerApi.Logger.Infof( "Conf asked" )
  httpResponse.Code = http.StatusOK
  httpResponse.Payload = handlerApi.Conf.GetHead()
}
