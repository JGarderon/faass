package main
/*
  FaasS = Function as a (Simple) Service
  ---
  Created by Julien Garderon (Nothus)
  from August 01 to 07, 2022
  MIT Licence
  ---
  This is a POC - Proof of Concept -, based on the idea of the OpenFaas project
  /!\ NOT INTENDED FOR PRODUCTION, only dev /!\
*/

import (
  "encoding/json"
  "io/ioutil"
  "os"
  "os/exec"
  "reflect"
  "errors"
  "flag"
  "net/http"
  "strings"
  "log"
  "time"
  "fmt"
  "sync"
  "regexp"
  "strconv"
  "io"
  "unicode/utf8"
  "path/filepath"
  "os/signal"
  "syscall"
  "context"
)

// -----------------------------------------------

var GLOBAL_CONF_PATH string
var GLOBAL_CONF Conf
var GLOBAL_CONF_MUTEXT sync.RWMutex 

var GLOBAL_REGEX_ROUTE_NAME *regexp.Regexp

var GLOBAL_WAIT_GROUP sync.WaitGroup

// -----------------------------------------------

const (
  ExitOk = iota           // toujours '0'
  ExitUndefined           // toujours '1'
  ExitConfCreateKo
  ExitConfLoadKo
  ExitConfRegexUrlKo
  ExitConfShuttingServerFailed
)

// -----------------------------------------------

var (
  DebugLogger     *log.Logger
  InfoLogger      *log.Logger
  WarningLogger   *log.Logger
  ErrorLogger     *log.Logger
  PanicLogger     *log.Logger
)

// -----------------------------------------------

type Conf struct {
  Containers Containers
  Domain string `json:"domain"`
  Authorization string `json:"authorization"`
  IncomingPort int `json:"listen"`
  DelayCleaningContainers int `json:"delay"`
  UI string `json:"ui"`
  TmpDir string `json:"tmp"`
  Prefix string `json:"prefix"`
  Routes map[string]*Route `json:"routes"`
}

func ConfImport( pathOption ...string ) bool {
  path := GLOBAL_CONF_PATH
  if len( pathOption ) < 0 {
    path = pathOption[0]
  }
  jsonFileInput, err := os.Open( path )
  if err != nil {
    log.Println( "ConfImport (open) :", err )
    return false
  }
  defer jsonFileInput.Close()
  byteValue, err := ioutil.ReadAll(jsonFileInput)
  if err != nil {
    log.Println( "ConfImport (ioutil) :", err )
    return false
  }
  json.Unmarshal( byteValue, &GLOBAL_CONF )
  if GLOBAL_CONF.DelayCleaningContainers < 5 {
    GLOBAL_CONF.DelayCleaningContainers = 5
    WarningLogger.Println( "new value for delay cleaning containers : min 5 (seconds)" ) 
  }
  if GLOBAL_CONF.DelayCleaningContainers > 60 {
    GLOBAL_CONF.DelayCleaningContainers = 60 
    WarningLogger.Println( "new value for delay cleaning containers : max 60 (seconds)" ) 
  }
  return true
}

func (c *Conf) GetParam( key string ) string {
  if GLOBAL_CONF.Authorization != "" {
    GLOBAL_CONF_MUTEXT.RLock() 
    defer GLOBAL_CONF_MUTEXT.RUnlock() 
  }
  e := reflect.ValueOf( c ).Elem()
  r := e.FieldByName( key )
  if r.IsValid() {
    return r.Interface().(string)
  }
  return ""
}

func (c *Conf) GetRoute( key string ) (route *Route, err error) {
  if route, ok := c.Routes[key]; ok {
    return route, nil
  }
  return nil, errors.New( "unknow routes" )
}

func (c *Conf) Export( pathOption ...string ) bool {
  path := GLOBAL_CONF_PATH
  if len( pathOption ) < 0 {
    path = pathOption[0]
  }
  v, err := json.Marshal( c )
  if err != nil {
    log.Fatal( "export conf (Marshal) :", err )
    return false
  }
  jsonFileOutput, err := os.Create( path )
  defer jsonFileOutput.Close()
  if err != nil {
    log.Println( "export conf (open) :", err )
    return false
  }
  _, err = jsonFileOutput.Write( v )
  if err != nil {
    log.Println( "export conf (write) :", err )
    return false
  }
  return true
}

// -----------------------------------------------

type Route struct {
  Name string `json:"name"`
  Authorization string `json:"authorization"`
  Environment map[string]string `json:"env"`
  Image string `json:"image"`
  Timeout int `json:"timeout"`
  Retry int `json:"retry"`
  Delay int `json:"delay"`
  Port int `json:"port"`
  LastRequest time.Time 
  Id string
  IpAdress string
  Mutex sync.RWMutex
}

// -----------------------------------------------

func CreateLogger() {
  m := "!!! starting ; test log"
  DebugLogger = log.New( os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile )
  DebugLogger.Println( m )
  InfoLogger = log.New( os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile )
  InfoLogger.Println( m )
  WarningLogger = log.New( os.Stderr, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile )
  WarningLogger.Println( m )
  ErrorLogger = log.New( os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile )
  ErrorLogger.Println( m )
  PanicLogger = log.New( os.Stderr, "PANIC: ", log.Ldate|log.Ltime|log.Lshortfile )
  PanicLogger.Println( m )   
}

func CreateRegexUrl() {
  regex, err := regexp.Compile( "^([a-z0-9_-]+)" )
  if err != nil {
    os.Exit( ExitConfRegexUrlKo )
  }
  GLOBAL_REGEX_ROUTE_NAME = regex
}

func GetRootPath() (rootPath string, err error) {
  ex, err := os.Executable()
  if err != nil {
    WarningLogger.Println( "unable to get current path of executable : ", err )
    return "", errors.New( "unable to get current path of executable" )
  }
  rootPath = filepath.Dir( ex )
  return rootPath, nil
}

func CreateEnv() bool {
  rootPath, err := GetRootPath()
  if err != nil {
    return false
  }
  uiTmpDir := filepath.Join(
    rootPath,
    "./content",
  )
  pathTmpDir := filepath.Join(
    rootPath,
    "./tmp",
  )
  newMapRoutes := make( map[string]*Route ) 
  newMapEnvironmentRoute := make( map[string]string ) 
  newMapEnvironmentRoute["faass-example"] = "true" 
  newMapRoutes["example"] = &Route {    
      Name: "exampleContainer",
      Authorization: "Basic YWRtaW46YXplcnR5", 
      Environment: newMapEnvironmentRoute, 
      Image: "nginx", 
      Timeout : 250, 
      Retry: 3, 
      Delay: 8, 
      Port: 80, 
  } 
  newConf := Conf {
    Domain: "https://localhost",
    Authorization: "Basic YWRtaW46YXplcnR5", // admin:azerty 
    IncomingPort: 9090,
    UI: uiTmpDir, 
    TmpDir: pathTmpDir,
    Prefix: "lambda",
    Routes: newMapRoutes, 
  }
  if !newConf.Export() {
    ErrorLogger.Println( "Unable to create environment : conf" )
    return false
  }
  if err := os.Mkdir( uiTmpDir, os.ModePerm ); err != nil {
    WarningLogger.Println( "Unable to create environment : content dir (", err, "); pass" )
    return false
  }
  if err := os.Mkdir( pathTmpDir, os.ModePerm ); err != nil {
    WarningLogger.Println( "Unable to create environment : tmp dir (", err, "); pass" )
    return false
  }
  return true
}

func StartEnv() {
  confPath := flag.String( "conf", "./conf.json", "path to conf (JSON ; string)" ) 
  prepareEnv := flag.Bool( "prepare", false, "create environment (conf+dir ; bool)" )
  flag.Parse()
  GLOBAL_CONF_MUTEXT.Lock() 
  defer GLOBAL_CONF_MUTEXT.Unlock() 
  GLOBAL_CONF_PATH = string( *confPath )
  if *prepareEnv {
    if CreateEnv() {
      os.Exit( ExitOk )
    } else {
      os.Exit( ExitConfCreateKo )
    }
  }
  if !ConfImport() {
    PanicLogger.Println( "Unable to load configuration" )
    os.Exit( ExitConfLoadKo )
  }
  if GLOBAL_CONF.IncomingPort < 1 || GLOBAL_CONF.IncomingPort > 65535 {
    PanicLogger.Println(
      "Bad configuration : incorrect port '"+strconv.Itoa( GLOBAL_CONF.IncomingPort )+"'", 
    )
    os.Exit( ExitConfLoadKo )
  }
  rootPath, err := GetRootPath()
  if err != nil {
    PanicLogger.Println( "Unable to root path of executable" )
    os.Exit( ExitConfLoadKo )
  }
  GLOBAL_CONF.UI = filepath.Join(
    rootPath,
    "./content",
  )
  GLOBAL_CONF.TmpDir = filepath.Join(
    rootPath,
    "./tmp",
  )
}

// -----------------------------------------------

type Container interface {
  Create ( route Route ) ( state bool, err error )
  Check ( route Route ) ( state bool, err error )
  Remove ( route Route ) ( state bool, err error )
}
// -----------------------------------------------

type Containers struct {}

func ( container *Containers ) Run ( route *Route ) ( err error ) {
  if route.Id == "" {
    _, err := container.Create( route )
    if err != nil {
      return err
    }
  }
  state, err := container.Check( route )
  if err != nil {
    return err
  }
  if state == "running" {
    return nil
  }
  started, err := container.Start( route )
  cIpAdress, err := container.GetInfos(
    route,
    "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}",
  )
  if err != nil {
    return err
  }
  route.IpAdress = cIpAdress
  if err != nil || started == false {
    return err
  }
  for i := 0; i < route.Retry; i++ {
    time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
    state, err = container.Check( route )
    if err != nil {
      return err
    }
    if state == "running" {
      return nil
    }
  }
  return errors.New( "Container has failed to start in time" )
}

func ( container *Containers ) Create ( route *Route ) ( state string, err error ) {
  if route.Image == "" {
    return "failed", errors.New( "Image container has null value" )
  }
  if route.Name == "" {
    return "failed", errors.New( "Name container has null value" )
  }
  fileEnvPath := filepath.Join(
    GLOBAL_CONF.GetParam("TmpDir"),
    route.Name+".env", 
  )
  route.Mutex.Lock()
  defer route.Mutex.Unlock()
  fileEnv, err := os.Create( fileEnvPath )
  defer fileEnv.Close()
  if err != nil {
    ErrorLogger.Println( "unable to create container file env : ", err )
    return "failed", errors.New( "env file for container failed" )
  }
  for key, value := range route.Environment {
    fileEnv.WriteString( key+"="+value+"\n" )
  }
  pathContainerTmpDir := filepath.Join(
    GLOBAL_CONF.GetParam("TmpDir"),
    route.Name, 
  )
  if err := os.MkdirAll( pathContainerTmpDir, os.ModePerm ); err != nil {
    ErrorLogger.Println( "unable to create tmp dir for container : ", err )
    return "failed", errors.New( "tmp dir for container failed" )
  }
  cmd := exec.Command(
    "docker",
    "container",
    "create",
    "--label",
    "faass=true",
    "--mount",
    "type=bind,source="+pathContainerTmpDir+",target=/hostdir",
    "--hostname",
    route.Name,
    "--env-file",
    fileEnvPath,
    route.Image,
  )
  o, err := cmd.CombinedOutput()
  cId := strings.TrimSuffix( string( o ), "\n" )
  if err != nil {
    ErrorLogger.Println( "container create in error : ", err )
    return "undetermined", errors.New( cId )
  }
  route.Id = cId 
  cmd = exec.Command(
    "docker",
    "inspect",
    "-f",
    "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}",
    cId,
  )
  o, err = cmd.Output()
  cIP := strings.TrimSuffix( string( o ), "\n" )
  route.IpAdress = cIP
  return cId, nil
}

func ( container *Containers ) Check ( route *Route ) ( state string, err error ) {
  // docker container ls -a --filter 'status=created' --format "{{.ID}}" | xargs docker rm
  if route.Id == "" {
    return "undetermined", errors.New( "ID container has null string" )
  }
  route.Mutex.RLock()
  defer route.Mutex.RUnlock()
  cmd := exec.Command(
    "docker",
    "container",
    "inspect",
    "-f",
    "{{.State.Status}}",
    route.Id,
  )
  o, err := cmd.CombinedOutput()
  cState := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil {
    ErrorLogger.Println( "container check ", err )
    return "undetermined", errors.New( cState )
  }
  return cState, nil
}

func ( container *Containers ) Start ( route *Route ) ( state bool, err error ) {
  if route.Id == "" {
    return false, errors.New( "ID container has null string" )
  }
  route.Mutex.Lock()
  defer route.Mutex.Unlock()
  cmd := exec.Command(
    "docker",
    "container",
    "restart",
    route.Id,
  )
  o, err := cmd.CombinedOutput()
  cId := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil || cId != route.Id {
    return false, errors.New( cId )
  }
  return true, nil
}

func ( container *Containers ) Stop ( route *Route ) ( state bool, err error ) {
  if route.Id == "" {
    return false, errors.New( "ID container has null string" )
  }
  route.Mutex.Lock()
  defer route.Mutex.Unlock()
  cmd := exec.Command(
    "docker",
    "container",
    "stop",
    route.Id,
  )
  o, err := cmd.CombinedOutput()
  cId := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil || cId != route.Id {
    return false, errors.New( cId )
  }
  return true, nil
}

func ( container *Containers ) Remove ( route *Route ) ( state bool, err error ) {
   if route.Id == "" {
    return false, errors.New( "ID container has null string" )
  }
  route.Mutex.Lock()
  defer route.Mutex.Unlock()
  cmd := exec.Command(
    "docker",
    "container",
    "rm",
    route.Id,
  )
  o, err := cmd.CombinedOutput()
  cId := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil || cId != route.Id {
    return false, errors.New( cId )
  }
  route.Id = ""
  return true, nil
}

func ( container *Containers ) GetInfos ( route *Route, pattern string ) ( infos string, err error ) {
   if route.Id == "" {
    return "", errors.New( "ID container has null string" )
  }
  route.Mutex.RLock()
  defer route.Mutex.RUnlock()
  cmd := exec.Command(
    "docker",
    "container",
    "inspect",
    "-f",
    pattern,
    route.Id,
  )
  o, err := cmd.Output()
  cInfos := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil {
    return "", errors.New( route.Id )
  }
  return cInfos, nil
}

// -----------------------------------------------

type HTTPResponse struct {
  Code int 
  MessageError string
  Payload interface{}
  IOFile io.ReadCloser
}

func ( httpR *HTTPResponse ) httpRespond( w http.ResponseWriter ) bool { 
  if httpR.Code < 300 {
    if httpR.Payload != nil {
      HTTPResponse, err := json.Marshal( httpR.Payload ) 
      if err != nil { 
        w.WriteHeader( 500 ) 
        w.Header().Set( "Content-type", "application/problem+json" )
        ErrorLogger.Println( "API export conf (Marshal) :", err ) 
        httpR.Code = 500
        return false
      } 
      w.WriteHeader( httpR.Code ) 
      w.Header().Set( "Content-type", "application/json" ) 
      w.Write( HTTPResponse ) 
    } else if httpR.IOFile != nil { 
      w.WriteHeader( httpR.Code ) 
      io.Copy( w, httpR.IOFile )
    } else {
      w.WriteHeader( httpR.Code ) 
    }
    return true 
  } else { 
    w.WriteHeader( httpR.Code ) 
    w.Header().Set( "Content-type", "application/problem+json" )
    w.Write( 
      []byte( 
        fmt.Sprintf( 
          `{"message":"%v"}`, 
          httpR.MessageError, 
        ),
      ),
    )
    return true
  }
}

// -----------------------------------------------

func ApiHandler(w http.ResponseWriter, r *http.Request) { 
  if r.Header.Get( "Authorization" ) != GLOBAL_CONF.Authorization  { 
    HTTPResponse := HTTPResponse { 
      Code: 401, 
      MessageError: "you must be authentified", 
    }
    HTTPResponse.httpRespond( w ) 
    return 
  } 
  pathExtract := r.URL.Path[5:] // "/api/" = 5 signes 
  re := regexp.MustCompile(`^([a-z]+)(/([a-zA-Z0-9_-]+))?`)
  typeOf := "" 
  typeId := "" 
  if re.MatchString( pathExtract ) {
    parts := re.FindStringSubmatch( pathExtract )
    typeOf = parts[1]
    typeId = parts[3]
  }
  switch typeOf {
  case "conf":
    ApiHandlerConf( typeId, w, r )
  case "route":
    ApiHandlerRoute( typeId, w, r )
  default:
    w.WriteHeader( 404 ) 
  }
}

func ApiHandlerRoute(typeId string, w http.ResponseWriter, r *http.Request) {
  HTTPResponse := &HTTPResponse { 
    Code: 500,
    MessageError: "an unexpected error has occurred", 
  } 
  defer HTTPResponse.httpRespond( w ) 
  switch r.Method  {
  case "GET":
    GLOBAL_CONF_MUTEXT.RLock() 
    defer GLOBAL_CONF_MUTEXT.RUnlock() 
    route, _ := GLOBAL_CONF.GetRoute( typeId )
    if route == nil {
      HTTPResponse.Code = 404 
      HTTPResponse.MessageError = "unknow route"
      InfoLogger.Println( "Route ", typeId, "asked (non-existent)" )
      return 
    } 
    HTTPResponse.Code = 200 
    HTTPResponse.Payload = route 
    InfoLogger.Println( "Route ", typeId, "asked (existent)" ) 
  case "POST":
    body, err := ioutil.ReadAll( r.Body )
    if err != nil { 
      ErrorLogger.Println( "API import route (read body) :", err )
      HTTPResponse.Code = 500 
      HTTPResponse.MessageError = "the request's body is an invalid"
      return 
    } 
    var newRoute = Route {}
    err = json.Unmarshal( body, &newRoute ) 
    if err != nil { 
      ErrorLogger.Println( "API import route (parse body) :", err )
      HTTPResponse.Code = 400 
      HTTPResponse.MessageError = "the request's body is an invalid"
      return 
    } 
    GLOBAL_CONF_MUTEXT.Lock() 
    defer GLOBAL_CONF_MUTEXT.Unlock() 
    route, _ := GLOBAL_CONF.GetRoute( typeId )
    if route != nil {
      rId := route.Id 
      if rId != "" {
        _, err := GLOBAL_CONF.Containers.Stop( route )
        if err != nil {
          WarningLogger.Println( "Container ", route.Name, "(cId ", rId, ") not stopped - maybe he is still active ?" )
        } 
        InfoLogger.Println( "Container ", route.Name, "(cId ", rId, ") stopped" )
        time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
        _, err = GLOBAL_CONF.Containers.Remove( route )
        if err != nil {
          WarningLogger.Println( "Container ", route.Name, "(cId ", rId, ") not terminated" )
        } else {
          InfoLogger.Println( "Container ", route.Name, "(ex-cId ", rId, ") terminated" )
        }
      }
    } 
    GLOBAL_CONF.Routes[typeId] = &newRoute 
    InfoLogger.Println( "Route ", typeId, "updated" )
    HTTPResponse.Code = 200 
    HTTPResponse.Payload = nil 
  case "DELETE":
    GLOBAL_CONF_MUTEXT.Lock() 
    defer GLOBAL_CONF_MUTEXT.Unlock() 
    route, _ := GLOBAL_CONF.GetRoute( typeId )
    if route == nil {
      HTTPResponse.Code = 404 
      HTTPResponse.MessageError = "unknow route"
      return 
    } 
    rId := route.Id 
    if rId != "" {
      _, err := GLOBAL_CONF.Containers.Stop( route )
      if err != nil {
        WarningLogger.Println( "Container ", route.Name, "(cId ", rId, ") not stopped - maybe he is still active ?" )
      } 
      InfoLogger.Println( "Container ", route.Name, "(cId ", rId, ") stopped" )
      time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
      _, err = GLOBAL_CONF.Containers.Remove( route )
      if err != nil {
        WarningLogger.Println( "Container ", route.Name, "(cId ", rId, ") not terminated" )
      } else {
        InfoLogger.Println( "Container ", route.Name, "(ex-cId ", rId, ") terminated" )
      } 
    }
    delete( GLOBAL_CONF.Routes, typeId ) 
    InfoLogger.Println( "Route ", typeId, "removed" )
    HTTPResponse.Code = 200 
    HTTPResponse.Payload = nil 
  default:
    HTTPResponse.Code = 405 
    HTTPResponse.MessageError = "only GET, POST or DELETE verbs are allowed"
    InfoLogger.Println( "Route ", typeId, "action unknow (", r.Method, ")" ) 
  }
}

func ApiHandlerConf(_ string, w http.ResponseWriter, r *http.Request) {
  HTTPResponse := &HTTPResponse { 
    Code: 500,
    MessageError: "an unexpected error has occurred", 
  }
  defer HTTPResponse.httpRespond( w ) 
  switch r.Method  {
  case "GET": 
    HTTPResponse.Code = 200 
    HTTPResponse.Payload = GLOBAL_CONF 
    return 
  case "PATCH": 
    if contentType := r.Header.Get("Content-type"); contentType != "application/json" {
      HTTPResponse.Code = 400 
      HTTPResponse.MessageError = "you must have 'application/json' content-type header"
      return 
    } 
    body, err := ioutil.ReadAll( r.Body )
    if err != nil { 
      ErrorLogger.Println( "API import conf (read body) :", err )
      HTTPResponse.Code = 500 
      HTTPResponse.MessageError = "the request's body is an invalid"
      return 
    } 
    var f interface{} 
    err = json.Unmarshal( body, &f ) 
    if err != nil { 
      ErrorLogger.Println( "API import conf (parse body) :", err )
      HTTPResponse.Code = 400 
      HTTPResponse.MessageError = "the request's body is an invalid"
      return 
    } 
    o := f.( map[string]interface{} ) 
    for key, value := range o { 
      switch key {
      case "delay": 
        switch value.(type) {
        case float64:
          delay := int(value.(float64)) 
          if delay >= 5 && delay <= 60 {
            GLOBAL_CONF_MUTEXT.Lock()
            GLOBAL_CONF.DelayCleaningContainers = delay 
            GLOBAL_CONF_MUTEXT.Unlock()
            WarningLogger.Println( "Delay changed ; new value :", delay )
            continue 
          } else { 
            HTTPResponse.Code = 400 
            HTTPResponse.MessageError = "value of delay invalid : int between 5 and 60 (seconds)"
            return 
          } 
        default:
          HTTPResponse.Code = 400 
          HTTPResponse.MessageError = "type's value of delay invalid"
          return 
        } 
      default:
        HTTPResponse.Code = 501 
        HTTPResponse.MessageError = "at least one key is invalid"
        return
      }
    } 
    HTTPResponse.Code = 202
  default:
    HTTPResponse.Code = 400
    HTTPResponse.MessageError = "you must have GET or POST HTTP verbs"
    return 
  }
}

// -----------------------------------------------

func lambdaHandler(w http.ResponseWriter, r *http.Request) {
  HTTPResponse := HTTPResponse { 
    Code: 500, 
    MessageError: "an unexpected error found", 
  }
  defer HTTPResponse.httpRespond( w ) 
  url := r.URL.Path[8:] // "/lambda/" = 8 signes
  if GLOBAL_REGEX_ROUTE_NAME.MatchString( url ) != true {
    InfoLogger.Println( "bad desired url :", url )
    HTTPResponse.Code = 400
    HTTPResponse.MessageError = "bad desired url" 
    return
  }
  InfoLogger.Println( "known real desired url :", r.URL )
  rNameSize := utf8.RuneCountInString( GLOBAL_REGEX_ROUTE_NAME.FindStringSubmatch( url )[1] )
  rName := url[:rNameSize]
  rRest := url[rNameSize:]
  if rRest == "" {
    rRest += "/"
  }
  GLOBAL_CONF_MUTEXT.RLock()
  route, err := GLOBAL_CONF.GetRoute( rName )
  if err != nil {
    InfoLogger.Println( "unknow desired url :", rName, "(", err, ")" )
    HTTPResponse.Code = 404
    HTTPResponse.MessageError = "unknow desired url" 
    GLOBAL_CONF_MUTEXT.RUnlock()
    return
  } 
  if r.Header.Get( "Authorization" ) != route.Authorization  { 
    HTTPResponse.Code = 401
    HTTPResponse.MessageError = "you must be authentified" 
    GLOBAL_CONF_MUTEXT.RUnlock()
    return 
  } 
  InfoLogger.Println( "known desired url :", rName )
  route.LastRequest = time.Now()
  err = GLOBAL_CONF.Containers.Run( route )
  GLOBAL_CONF_MUTEXT.RUnlock()
  if err != nil {
    WarningLogger.Println( "unknow state of container for route :", rName, "(", err, ")" )
    HTTPResponse.Code = 503
    HTTPResponse.MessageError = "unknow state of container" 
    return
  }
  DebugLogger.Println( "running container for desired route :", route.IpAdress, "(cId", route.Id, ")" )
  if r.URL.RawQuery != "" {
    rRest += "?"+r.URL.RawQuery
  }
  if r.URL.RawFragment != "" {
    rRest += "#"+r.URL.RawFragment
  }
  dURL := fmt.Sprintf(
    "http://%s%s",
    route.IpAdress+":"+strconv.Itoa( route.Port ) ,
    rRest,
  )
  DebugLogger.Println( "new url for desired route :", dURL, "(cId", route.Id, ")" )
  proxyReq, err := http.NewRequest(
    r.Method,
    dURL,
    r.Body,
  )
  if err != nil {
    WarningLogger.Println( "bad gateway for container as route :", rName, "(", err, ")" )
    HTTPResponse.Code = 502
    HTTPResponse.MessageError = "bad gateway for container" 
    HTTPResponse.httpRespond( w ) 
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
    WarningLogger.Println( "request failed to container as route :", rName, "(", err, ")" )
    HTTPResponse.Code = 500
    HTTPResponse.MessageError = "request failed to container"
    return
  }
  DebugLogger.Println( "result of desired route :", proxyRes.StatusCode, "(cId", route.Id, ")" )
  wH := w.Header()
  for header, values := range proxyRes.Header {
    for _, value := range values {
      wH.Add(header, value)
    }
  }
  HTTPResponse.Code = proxyRes.StatusCode 
  HTTPResponse.IOFile = proxyRes.Body
}

// -----------------------------------------------

func CleanContainers( ctx context.Context, force bool ) {
  GLOBAL_WAIT_GROUP.Add( 1 )
  for {
    tt := time.After( time.Duration( GLOBAL_CONF.DelayCleaningContainers ) * time.Second )
    select {
    case <-tt:
      for routeName := range GLOBAL_CONF.Routes {
        route := GLOBAL_CONF.Routes[routeName]
        if route.Id != "" {
          routeDelayLastRequest := route.LastRequest.Add( time.Duration( route.Delay ) * time.Second )
          GLOBAL_CONF_MUTEXT.RLock()
          state, err := GLOBAL_CONF.Containers.Check( route ) 
          if err != nil {
            WarningLogger.Println( "Container ", route.Name, "(cId ", route.Id, ") : state unknow ; ", err )
          } else if state != "exited" && routeDelayLastRequest.Before( time.Now() ) {
            _, err := GLOBAL_CONF.Containers.Stop( route )
            if err != nil {
              WarningLogger.Println( "Container ", route.Name, "(cId ", route.Id, ") not stopped - maybe he is still active ?" )
            } else {
              InfoLogger.Println( "Container", route.Name, "(cId ", route.Id, ") stopped"  )
            }
          }
          GLOBAL_CONF_MUTEXT.RUnlock()
        }
      }
    case <-ctx.Done():
      GLOBAL_CONF_MUTEXT.RLock()
      defer GLOBAL_CONF_MUTEXT.RUnlock()
      defer GLOBAL_WAIT_GROUP.Done()
      for routeName := range GLOBAL_CONF.Routes {
        route := GLOBAL_CONF.Routes[routeName]
        rId := route.Id
        if rId != "" {
          _, err := GLOBAL_CONF.Containers.Stop( route )
          if err != nil {
            WarningLogger.Println( "Container ", route.Name, "(cId ", rId, ") not stopped - maybe he is still active ?" )
          } else {
            InfoLogger.Println( "Container ", route.Name, "(cId ", rId, ") stopped" )
            time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
            _, err := GLOBAL_CONF.Containers.Remove( route )
            if err != nil {
              WarningLogger.Println( "Container ", route.Name, "(cId ", rId, ") not terminated" )
            } else {
              InfoLogger.Println( "Container ", route.Name, "(ex-cId ", rId, ") terminated" )
            }
          }
        }
      }
      return
    }
  }
}
// -----------------------------------------------

func RunServer ( httpServer *http.Server ) {
  defer InfoLogger.Println( "Shutdown ListenAndServeTLS terminated" )
  err := httpServer.ListenAndServeTLS(
    "server.crt",
    "server.key",
  )
  if err != nil && err != http.ErrServerClosed {
    PanicLogger.Println( "ListenAndServe err :", err )
    os.Exit( ExitUndefined )
  }
}
// -----------------------------------------------

func main() { 

  CreateLogger()
  StartEnv()

  CreateRegexUrl()

  ctx := context.Background()
  ctx, cancel := context.WithCancel( context.Background() )

  muxer := http.NewServeMux()

  UIPath := GLOBAL_CONF.GetParam( "UI" )
  if UIPath != "" {
    InfoLogger.Println( "UI path found :", UIPath )
    muxer.Handle( "/", http.FileServer( http.Dir( UIPath ) ) )
  }
 
  muxer.HandleFunc( "/lambda/", lambdaHandler )

  if GLOBAL_CONF.Authorization != "" {
    InfoLogger.Println( "Authorization secret found ; API active" )
    muxer.HandleFunc( "/api/", ApiHandler )
  } else { 
    InfoLogger.Println( "Authorization secret not found ; API inactive" )
  } 

  httpServer := &http.Server{
    Addr: ":"+strconv.Itoa( GLOBAL_CONF.IncomingPort ),
    Handler:     muxer,
  }

  signalChan := make(chan os.Signal, 1)
  go CleanContainers( ctx, false )
  go RunServer( httpServer )
  signal.Notify(
    signalChan,
    syscall.SIGHUP,
    syscall.SIGINT,
    syscall.SIGQUIT,
  )
  <-signalChan
  InfoLogger.Println("interrupt received ; shutting down")

  if err := httpServer.Shutdown( ctx ); err != nil {
    PanicLogger.Printf("shutdown error: %v\n", err)
    defer os.Exit( ExitConfShuttingServerFailed )
  }

  cancel()

  GLOBAL_WAIT_GROUP.Wait()

  InfoLogger.Printf("process gracefully stopped")

}