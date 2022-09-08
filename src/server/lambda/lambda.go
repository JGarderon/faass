package lambda

import(
  "encoding/json"
  "strings"
  "time"
  "fmt"
  "strconv"
  "io"
  "net/http"
  "unicode/utf8"
  "context"
  "encoding/binary" 
  "regexp"
  "sync"
  //-----------
  "httpresponse"
  "itinerary"
  "configuration"
  "logger"
)

// -----------------------------------------------

type HandlerLambda struct {
  GlobalRouteRegex *regexp.Regexp
  Logger *logger.Logger
  ConfMutext *sync.RWMutex
  Conf *configuration.Conf
}

// -----------------------------------------------

func Authorization( c *itinerary.Route, r *http.Request ) bool {
  if c.Authorization == "" { 
    if r.Header.Get( "Authorization" ) != "" {
      return false 
    }
    return true
  } else { 
    if r.Header.Get( "Authorization" ) == c.Authorization {
      return true 
    }
    return false 
  }
}

// -----------------------------------------------

type FunctionResponseHeaders struct {
  Code int `json:"code"`
  Headers map[string]string `json:"headers"`
} 

// -----------------------------------------------

func ( handlerLambda *HandlerLambda ) ServeFunction ( route *itinerary.Route, httpResponse *httpresponse.Response, w http.ResponseWriter, r *http.Request ) {
  ctx, cancel := context.WithTimeout( 
    context.Background(), 
    time.Duration( route.Timeout ) * time.Millisecond, 
  ) 
  defer cancel() 
  handlerLambda.ConfMutext.RLock() 
  tmpDir := handlerLambda.Conf.TmpDir
  handlerLambda.ConfMutext.RUnlock() 
  fileEnvPath, err := route.CreateFileEnv( tmpDir ) 
  if err != nil {
    httpResponse.MessageError = "unable to create environment file"
    return 
  }
  routeName := route.Name 
  cmd, err := handlerLambda.Conf.Containers.ExecuteRequest( 
    ctx, 
    routeName, 
    route.ScriptPath, 
    fileEnvPath, 
    route.Image, 
    route.ScriptCmd, 
  ) 
  handlerLambda.ConfMutext.RUnlock() 
  if err != nil {
    handlerLambda.Logger.Warningf( "unable to get command for '%s' : %s", routeName, err )
    httpResponse.MessageError = "unable to run request in container (internal error)" 
    return 
  }
  stdin, err := cmd.StdinPipe()
  if err != nil {
    handlerLambda.Logger.Warningf( "unable to get container's stdin '%s' : %s", routeName, err )
    httpResponse.MessageError = "unable to run request in container (internal error)" 
    return 
  }
  stderr, err := cmd.StderrPipe()
  if err != nil {
    handlerLambda.Logger.Warningf( "unable to get on container's stderr '%s' : %s", routeName, err )
    httpResponse.MessageError = "unable to run request in container (internal error)" 
    return 
  }
  go func() {
    defer stdin.Close()
    io.Copy( stdin, r.Body ) 
  }() 
  go func() {
    errMessage, _ := io.ReadAll(stderr)
    if len( errMessage ) > 0 {
      handlerLambda.Logger.Debugf( "error message from container '%s' : %s", routeName, string( errMessage ) ) 
    }
  }()
  handlerLambda.Logger.Warningf( "run container for route '%s'", routeName )
  out, err := cmd.Output() 
  step := uint32( 0 )  
  if err != nil { 
    handlerLambda.Logger.Warningf( "unable to run request in container '%s' : %s", routeName, err )
    httpResponse.MessageError = "unable to run request in container (time out or failed)" 
    return 
  }
  httpResponse.MessageError = "unable to run request in container (incorrect response)" 
  if len(out) < 4 {
    handlerLambda.Logger.Warning( "incorrect size of headers'length from container '%s'", routeName )
    return 
  }
  sizeHeaders := binary.BigEndian.Uint32( out[0:4] ) 
  if sizeHeaders < 1 {
    handlerLambda.Logger.Warning( "headers of response null from container '%s'", routeName )
    return 
  }
  step += 4
  var responseHeaders FunctionResponseHeaders
  err = json.Unmarshal( out[step:step+sizeHeaders], &responseHeaders )
  if err != nil {
    handlerLambda.Logger.Warning( "incorrect headers payload of response from container '%s'", routeName )
    return 
  }
  step += sizeHeaders 
  httpResponse.Code = responseHeaders.Code
  header := w.Header()
  contentTypeSend := false 
  for key, value := range responseHeaders.Headers {
    if strings.ToLower( key ) == "content-type" {
      header.Add( "Content-type", value )
      contentTypeSend = true 
    } else {
      header.Add( "x-faas-"+key, value ) 
    }
  } 
  if contentTypeSend == false {
    header.Add( "Content-type", "application/json" )
  } 
  w.Write( out[step+4:] ) 
  return 
}

func ( handlerLambda HandlerLambda ) ServeHTTP ( w http.ResponseWriter, r *http.Request ) {
  httpResponse := httpresponse.Response { 
    Code: 500, 
    MessageError: "an unexpected error found", 
  }
  defer httpResponse.Respond( handlerLambda.Logger, w ) 
  url := r.URL.Path[8:] // "/lambda/" = 8 signes
  if handlerLambda.GlobalRouteRegex.MatchString( url ) != true {
    handlerLambda.Logger.Info( "bad desired url :", url )
    httpResponse.Code = 400
    httpResponse.MessageError = "bad desired url" 
    return
  }
  handlerLambda.Logger.Info( "known real desired url :", r.URL )
  rNameSize := utf8.RuneCountInString( handlerLambda.GlobalRouteRegex.FindStringSubmatch( url )[1] )
  routeName := url[:rNameSize]
  rRest := url[rNameSize:]
  if rRest == "" {
    rRest += "/"
  }
  handlerLambda.ConfMutext.RLock()
  route, err := handlerLambda.Conf.GetRoute( routeName )
  if err != nil {
    handlerLambda.Logger.Info( "unknow desired url :", routeName, "(", err, ")" )
    httpResponse.Code = 404
    httpResponse.MessageError = "unknow desired url" 
    handlerLambda.ConfMutext.RUnlock()
    return
  } 
  handlerLambda.Logger.Info( "known desired url :", routeName )
  if Authorization( route, r ) != true { 
    httpResponse.Code = 401
    httpResponse.MessageError = "you must be authentified" 
    handlerLambda.Logger.Info( "known desired url and unauthentified request :", routeName )
    handlerLambda.ConfMutext.RUnlock()
    return 
  } 
  if route.IsService != true {
    handlerLambda.ServeFunction( route, &httpResponse, w, r )
    return 
  }
  handlerLambda.ConfMutext.RUnlock()
  handlerLambda.ConfMutext.Lock()
  route, err = handlerLambda.Conf.GetRoute( routeName )
  tmpDir := handlerLambda.Conf.TmpDir
  if err != nil || route.IsService != true {
    handlerLambda.Logger.Info( "unknow desired url :", routeName, "(", err, ")" )
    httpResponse.Code = 404
    httpResponse.MessageError = "unknow desired url" 
    handlerLambda.ConfMutext.RUnlock()
    return
  } 
  err = handlerLambda.Conf.Containers.Run( tmpDir, route )
  routeIpAdress := route.IpAdress
  routePort := route.Port
  routeId := route.Id
  handlerLambda.ConfMutext.Unlock()
  if err != nil {
    handlerLambda.Logger.Warning( "unknow state of container for route :", routeName, "(", err, ")" )
    httpResponse.Code = 503
    httpResponse.MessageError = "unknow state of container" 
    return
  }
  handlerLambda.Logger.Debug( "running container for desired route :", routeIpAdress, "(cId", route.Id, ")" )
  if r.URL.RawQuery != "" {
    rRest += "?"+r.URL.RawQuery
  }
  if r.URL.RawFragment != "" {
    rRest += "#"+r.URL.RawFragment
  }
  dURL := fmt.Sprintf(
    "http://%s%s",
    routeIpAdress+":"+strconv.Itoa( routePort ) ,
    rRest,
  )
  handlerLambda.Logger.Debug( "new url for desired route :", dURL, "(cId", routeId, ")" )
  proxyReq, err := http.NewRequest(
    r.Method,
    dURL,
    r.Body,
  )
  if err != nil {
    handlerLambda.Logger.Warning( "bad gateway for container as route :", routeName, "(", err, ")" )
    httpResponse.Code = 502
    httpResponse.MessageError = "bad gateway for container" 
    httpResponse.Respond( handlerLambda.Logger, w ) 
    return
  }
  proxyReq.Header.Set( "Host", r.Host )
  proxyReq.Header.Set( "X-Forwarded-For", r.RemoteAddr )
  for header, values := range r.Header {
    for _, value := range values {
      proxyReq.Header.Add(header, value)
    }
  }
  client := &http.Client{
    Timeout: time.Duration( route.Timeout ) * time.Millisecond,
  }
  proxyRes, err := client.Do( proxyReq )
  if err != nil {
    handlerLambda.Logger.Warning( "request failed to container as route :", routeName, "(", err, ")" )
    httpResponse.Code = 500
    httpResponse.MessageError = "request failed to container"
    return
  }
  handlerLambda.Logger.Debug( "result of desired route :", proxyRes.StatusCode, "(cId", routeId, ")" )
  wH := w.Header()
  for header, values := range proxyRes.Header {
    for _, value := range values {
      wH.Add(header, value)
    }
  }
  httpResponse.Code = proxyRes.StatusCode 
  httpResponse.IOFile = proxyRes.Body
}
