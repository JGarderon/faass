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
  "strings"
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
  ConfPathCmdContainerDefault           = "/usr/bin/docker"
  ConfDomainDefault                     = "https://localhost"
  ConfIncomingAdressDefault             = "0.0.0.0"
  ConfIncomingPortDefault               = 9090
  ConfIncomingTLSDefault                = ""
  ConfDelayCleaningContainersDefault    = 60
  ConfDelayCleaningContainersMin        = 5
  ConfDelayCleaningContainersMax        = 3600
  ConfAuthorizationDefault              = "Basic YWRtaW46YXplcnR5" // admin:azerty 
  ConfPrefixDefault                     = "lambda"

  FunctionTimeoutDefault                = 1500

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
  PathCmdContainer string `json:"pathcmdcontainer"`
  Containers executors.Containers `json:"-"`
  Domain string `json:"domain"`
  Authorization string `json:"authorization"`
  IncomingAdress string `json:"adress"`
  IncomingPort int `json:"listen"`
  IncomingTLS string `json:"tls"`
  IncomingTLSCrt string `json:"-"`
  IncomingTLSKey string `json:"-"`
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
  if c.IncomingTLS != "" {
    r := strings.Split( c.IncomingTLS, ":" )
    if len( r ) != 2 {
      message = "tls has no ':' separator"
      state = false 
    } else {
      c.IncomingTLSCrt = r[0]
      c.IncomingTLSKey = r[1]
    }
  }
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
  c.PathCmdContainer = ConfPathCmdContainerDefault
  c.Domain = ConfDomainDefault
  c.Authorization = ConfAuthorizationDefault 
  c.IncomingAdress = ConfIncomingAdressDefault
  c.IncomingPort = ConfIncomingPortDefault
  c.IncomingTLS = ConfIncomingTLSDefault 
  c.DelayCleaningContainers = ConfDelayCleaningContainersDefault
  c.UI = uiTmpDir
  c.TmpDir = pathTmpDir
  c.Prefix = ConfPrefix
  newMapRoutes := make( map[string]*itinerary.Route )
  newMapEnvironmentRoute := make( map[string]string )
  newMapEnvironmentRoute["faass-example"] = "true"
  newMapRoutes["example-service"] = &itinerary.Route {
      Name: "exampleService",
      TypeName: "service", 
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
      TypeName: "function", 
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

func ( c *Conf ) GetHead() map[string]map[string]interface{} {
  return map[string]map[string]interface{} { 
    "PathCmdContainer": map[string]interface{} { 
      "default": ConfPathCmdContainerDefault, 
      "type": "string", 
      "realtype": "string", 
      "edit": false, 
      "title": "Path of command for executor's container",
      "help" : "", 
      "value": c.PathCmdContainer,
    },
    "Domain": map[string]interface{} { 
      "default": ConfDomainDefault, 
      "type": "string", 
      "realtype": "string", 
      "edit": false, 
      "title": "Listening domain",
      "help" : "", 
      "value": c.Domain,
    },
    "Authorization": map[string]interface{} { 
      "default": ConfAuthorizationDefault, 
      "type": "string", 
      "realtype": "string", 
      "edit": false, 
      "title": "Content of header Authorization",
      "help" : "", 
      "value": c.Authorization,
    },
    "IncomingAdress": map[string]interface{} { 
      "default": ConfIncomingAdressDefault, 
      "type": "string", 
      "realtype": "ip", 
      "edit": false, 
      "title": "Adress of bind",
      "help" : "\"0.0.0.0\" for all interfaces", 
      "value": c.IncomingAdress,
    },
    "IncomingPort": map[string]interface{} { 
      "default": ConfIncomingPortDefault, 
      "type": "number", 
      "realtype": "range(1,65535)", 
      "edit": false, 
      "title": "Port of bind",
      "help" : "Valid range : 1 to 65535", 
      "value": c.IncomingPort,
    },
    "IncomingTLS": map[string]interface{} { 
      "default": ConfIncomingTLSDefault, 
      "type": "string", 
      "realtype": "string", 
      "edit": false,
      "title": "Tuple of paths for certificat and key TLS",
      "help" : "Separator between two parts \":\"", 
      "value": c.IncomingTLS,
    },
    "DelayCleaningContainers": map[string]interface{} { 
      "default": ConfDelayCleaningContainersDefault, 
      "type": "number", 
      "realtype": "range(5,60)", 
      "edit": true, 
      "title": "Delay",
      "help" : "", 
      "value": c.DelayCleaningContainers,
    },
    "UI": map[string]interface{} { 
      "default": nil, 
      "type": "string", 
      "realtype": "path", 
      "edit": false, 
      "title": "Distant path for UI's content",
      "help" : "Can be relative or absolute root ; no default value (system-dependent)", 
      "value": c.UI,
    },
    "TmpDir": map[string]interface{} { 
      "default": nil, 
      "type": "string", 
      "realtype": "path", 
      "edit": false, 
      "title": "Distant path for temporary files",
      "help" : "Can be relative or absolute root ; no default value (system-dependent)", 
      "value": c.TmpDir,
    },
    "Prefix": map[string]interface{} { 
      "default": ConfPrefixDefault, 
      "type": "string", 
      "realtype": "string", 
      "edit": false, 
      "title": "Prefix for URI",
      "help" : "Must be a valid string", 
      "value": c.Prefix,
    },
  }
}
