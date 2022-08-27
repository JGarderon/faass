package configuration

import(
  "errors"
  "os"
  "path/filepath"
  "log"
  "io/ioutil"
  "encoding/json"
  "strconv"
  "net"
  // -----------
  "itinerary"
  "executors"
  "logger"
)

// -----------------------------------------------

var ConfDirTmp string                   = "./tmp"
var ConfDirContent string               = "./content"

// -----------------------------------------------

var ConfPrefix string                   = "lambda"

// -----------------------------------------------

const (
  ConfIncomingPortDefault               = 9090
  ConfDelayCleaningContainersDefault    = 60
  ConfDelayCleaningContainersMin        = 5
  ConfDelayCleaningContainersMax        = 3600

  FunctionTimeoutDefault                = 1000

  ServiceTimeoutDefault                 = 250
  ServiceRetryDefault                   = 3
  ServiceDelayDefault                   = 8
  ServicePortDefault                    = 80
)

// -----------------------------------------------

const (
  ExitOk = iota           // toujours '0'
  ExitUndefined           // toujours '1'
  ExitConfCreateKo
  ExitConfLoadKo
  ExitConfCheckKo
  ExitConfRegexUrlKo
  ExitConfShuttingServerFailed
)

// -----------------------------------------------

type Conf struct {
  Logger *logger.Logger `json:"-"`
  Containers executors.Containers `json:"-"`
  Domain string `json:"domain"`
  Authorization string `json:"authorization"`
  IncomingAdress string `json:"adress"`
  IncomingPort int `json:"listen"`
  DelayCleaningContainers int `json:"delay"`
  UI string `json:"ui"`
  TmpDir string `json:"tmp"`
  Prefix string `json:"prefix"`
  Routes map[string]*itinerary.Route `json:"routes"`
}

func Import( pathRoot string, c *Conf ) error {
  jsonFileInput, err := os.Open( pathRoot )
  if err != nil {
    return errors.New( "impossible to open conf's file" )
  }
  defer jsonFileInput.Close()
  byteValue, err := ioutil.ReadAll(jsonFileInput)
  if err != nil {
    return errors.New( "impossible to read conf's file" )
  }
  if json.Unmarshal( byteValue, c ) != nil {
    return errors.New( "impossible to parse conf's file" )
  }
  return nil
}

func ( c *Conf ) Check() ( message string, state bool ) {
  state = true
  if net.ParseIP( c.IncomingAdress ) == nil {
    message = "incomming adress is not an ip"
    state = false 
  }
  if c.DelayCleaningContainers < ConfDelayCleaningContainersMin {
    c.DelayCleaningContainers = ConfDelayCleaningContainersMin
    message = "new value for delay cleaning containers : min 5 (seconds)"
  }
  if c.DelayCleaningContainers > ConfDelayCleaningContainersMax {
    c.DelayCleaningContainers = ConfDelayCleaningContainersMax
    message = "new value for delay cleaning containers : max 60 (seconds)"
  }
  if c.IncomingPort < 1 || c.IncomingPort > 65535 {
    message = "bad configuration : incorrect port '"+strconv.Itoa( c.IncomingPort )+"'"
    state = false
  }
  return message, state
}

func ( c *Conf ) PopulateDefaults( rootPath string ) bool {
  uiTmpDir := filepath.Join(
    rootPath,
    ConfDirContent,
  )
  pathTmpDir := filepath.Join(
    rootPath,
    ConfDirTmp,
  )
  c.Domain = "https://localhost"
  c.Authorization = "Basic YWRtaW46YXplcnR5" // admin:azerty
  c.IncomingAdress = "0.0.0.0"
  c.IncomingPort = ConfIncomingPortDefault
  c.DelayCleaningContainers = ConfDelayCleaningContainersDefault
  c.UI = uiTmpDir
  c.TmpDir = pathTmpDir
  c.Prefix = ConfPrefix
  newMapRoutes := make( map[string]*itinerary.Route )
  newMapEnvironmentRoute := make( map[string]string )
  newMapEnvironmentRoute["faass-example"] = "true"
  newMapRoutes["example-service"] = &itinerary.Route {
      Name: "exampleService",
      IsService: true,
      Authorization: "Basic YWRtaW46YXplcnR5",
      Environment: newMapEnvironmentRoute,
      Image: "nginx",
      Timeout : ServiceTimeoutDefault,
      Retry: ServiceRetryDefault,
      Delay: ServiceDelayDefault,
      Port: ServicePortDefault,
  }
  newMapRoutes["example-function"] = &itinerary.Route {
      Name: "exampleFunction",
      IsService: false,
      ScriptPath: filepath.Join( pathTmpDir, "./example-function.py" ),
      ScriptCmd: []string{ "python3", "/function" },
      Environment: newMapEnvironmentRoute,
      Image: "python:3",
      Timeout : FunctionTimeoutDefault,
  }
  c.Routes = newMapRoutes
  return true
}

func ( c *Conf ) GetRoute( key string ) ( route *itinerary.Route, err error ) {
  if route, ok := c.Routes[key]; ok {
    return route, nil
  }
  return nil, errors.New( "unknow itinerary.Routes" )
}

func ( c *Conf ) Export( pathRoot string ) bool {
  v, err := json.Marshal( c )
  if err != nil {
    log.Fatal( "export conf (Marshal) :", err )
    return false
  }
  jsonFileOutput, err := os.Create( pathRoot )
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