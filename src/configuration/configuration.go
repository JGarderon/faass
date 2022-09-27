package configuration

import(
  "errors"
  "os"
  "path/filepath"
  "io/ioutil"
  "encoding/json"
  "strconv"
  "net"
  "strings"
  "fmt"
  // -----------
  "itinerary"
  "executors"
  "logger"
  "configuration/auth"
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
  ConfRefAuthorizationsDefault          = "default"
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
  ExitConfAuthCheckKo
  ExitConfRegexUrlKo
  ExitConfShuttingServerFailed
  ExitImageContainersPullFailed
)

// -----------------------------------------------

type Conf struct {
  Logger *logger.Logger `json:"-"`
  PathCmdContainer string `json:"pathcmdcontainer"`
  Containers executors.Containers `json:"-"`
  Domain string `json:"domain"`
  Authorizations map[string]auth.Authorization `json:"authorizations"`
  AuthorizationAPI string `json:"authapi"`
  AuthorizationAPIDefault string `json:"-"`
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
  if err := json.Unmarshal( byteValue, c ) ; err != nil {
    return errors.New( 
      fmt.Sprintf( "impossible to parse conf's file : %v", err ),
    ) 
  }
  return nil
}

func ( c *Conf ) ResolveAuth() error {
  c.Logger.Info( "startup of auth resolving" )
  if c.AuthorizationAPI != "" {
    if _, ok := c.Authorizations[c.AuthorizationAPI] ; !ok { 
      return errors.New( 
        fmt.Sprintf( "resolve auth failed ; API auth '%v' not exists", c.AuthorizationAPI ),
      )
    }
  }
  c.Logger.Debug( "API auth resolved" )
  c.AuthorizationAPIDefault = c.AuthorizationAPI
  c.AuthorizationAPI = c.Authorizations[c.AuthorizationAPI]
  for routeName, route := range c.Routes {
    if route.Authorization == "" {
      continue 
    }
    if _, ok := c.Authorizations[route.Authorization] ; !ok { 
      return errors.New( 
        fmt.Sprintf( 
          "resolve auth failed for route '%v' ; auth '%v' not exists", 
          routeName,
          route.Authorization,
        ),
      )
    } 
    c.Logger.Debugf( "route '%v' auth  resolved", routeName )
    route.AuthorizationDefault = route.Authorization
    route.Authorization = c.Authorizations[route.Authorization]
  }
  c.Logger.Info( "stop of auth resolving ; all success" )
  return nil
}

func ( c *Conf ) Check() ( err error ) {
  message := "" 
  if c.IncomingTLS != "" {
    r := strings.Split( c.IncomingTLS, ":" )
    if len( r ) != 2 {
      message = "tls has no ':' separator"
    } else {
      c.IncomingTLSCrt = r[0]
      c.IncomingTLSKey = r[1]
    }
  }
  if net.ParseIP( c.IncomingAdress ) == nil {
    message = "incomming adress is not an ip"
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
  }
  for name, route := range c.Routes {
    if err := route.Check(); err != nil {
      message = fmt.Sprintf( "bad route '%v' : %v", name, err )
      break
    }
  }
  if message != "" { 
    err = errors.New( message ) 
  }
  return err 
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
  c.Authorizations = map[string]auth.Authorization{ 
    ConfRefAuthorizationsDefault : ConfAuthorizationDefault, 
  } 
  c.AuthorizationAPI = ConfRefAuthorizationsDefault
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
      Authorization: ConfRefAuthorizationsDefault,
      Environment: newMapEnvironmentRoute,
      Image: "nginx",
      Timeout : ServiceTimeoutDefault,
      Retry: ServiceRetryDefault,
      Delay: ServiceDelayDefault,
      Port: ServicePortDefault,
  }
  newMapRoutes["example-shell"] = &itinerary.Route {
      Name: "exampleShell",
      TypeName: "shell", 
      Authorization: ConfRefAuthorizationsDefault,
      Environment: newMapEnvironmentRoute,
      Image: "",
      Timeout : ServiceTimeoutDefault,
      Delay: ServiceDelayDefault,
      ScriptPath: "/usr/bin/env",
      ScriptCmd: []string{},
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
  if err := c.Check() ; err != nil {
    return false 
  }
  return true
}

func ( c *Conf ) GetRoute( key string ) ( route *itinerary.Route, err error ) {
  if route, ok := c.Routes[key]; ok {
    return route, nil
  }
  return nil, errors.New( "unknow itinerary.Routes" )
}

func ( c *Conf ) Export( pathRoot string, reverseResolveAuth bool ) error {
  newConfExport := Conf{}
  newConfExport.PathCmdContainer = c.PathCmdContainer
  newConfExport.Domain = c.Domain
  authTmp := make( map[string]auth.Authorization )
  for key, value := range c.Authorizations {
    authTmp[key] = value
  }
  newConfExport.Authorizations = authTmp
  fmt.Println( "c.AuthorizationAPIDefault", c.AuthorizationAPI, "----" )
  fmt.Println( "c.AuthorizationAPIDefault", c.AuthorizationAPIDefault, "----" )
  if  reverseResolveAuth {
    newConfExport.AuthorizationAPI = c.AuthorizationAPIDefault
  } else {
    newConfExport.AuthorizationAPI = c.AuthorizationAPI
  }
  newConfExport.IncomingAdress = c.IncomingAdress
  newConfExport.IncomingPort = c.IncomingPort
  newConfExport.IncomingTLS = c.IncomingTLS
  newConfExport.DelayCleaningContainers = c.DelayCleaningContainers
  newConfExport.UI = c.UI
  newConfExport.TmpDir = c.TmpDir
  newConfExport.Prefix = c.Prefix
  routeTmp := make( map[string]*itinerary.Route ) 
  for key, value := range c.Routes {
    if newRoute, err := value.Export( reverseResolveAuth ) ; err != nil {
      return errors.New( 
        fmt.Sprintf( "export conf failed durint Route '%v' copying : %v", key, err ), 
      )
    } else { 
      routeTmp[key] = &newRoute
    }
  }
  newConfExport.Routes = routeTmp
  v, err := json.Marshal( newConfExport )
  if err != nil {
    return errors.New( 
      fmt.Sprintf( "export conf failed durint Marshal step : %v", err ), 
    )
  }
  jsonFileOutput, err := os.Create( pathRoot )
  defer jsonFileOutput.Close()
  if err != nil {
    return errors.New( 
      fmt.Sprintf( "export conf failed durint opening step : %v", err ), 
    )
  }
  _, err = jsonFileOutput.Write( v )
  if err != nil {
    return errors.New( 
      fmt.Sprintf( "export conf failed durint writing step : %v", err ), 
    )
  }
  return nil
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
    "Authorizations": map[string]interface{} { 
      "default": map[string]auth.Authorization{ 
        ConfRefAuthorizationsDefault : ConfAuthorizationDefault, 
      }, 
      "type": "string", 
      "realtype": "range(auth)", 
      "edit": false, 
      "title": "Reference to content of header Authorization",
      "help" : "", 
      "value": c.Authorizations,
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
