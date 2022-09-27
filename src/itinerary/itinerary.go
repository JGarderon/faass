package itinerary

import( 
  "time"
  "sync"
  "path/filepath"
  "os"
  "errors"
)

const (
  RouteTypeFunction int   = iota
  RouteTypeService        
  RouteTypeShell          
)

type Route struct {
  Name string `json:"name"`
  TypeName string `json:"type"`
  ScriptPath string `json:"script"`
  ScriptCmd []string `json:"cmd"`
  Authorization string `json:"authorization"`
  AuthorizationDefault string `json:"-"`
  Environment map[string]string `json:"env"`
  Image string `json:"image"`
  Timeout int `json:"timeout"`
  Retry int `json:"retry"`
  Delay int `json:"delay"`
  Port int `json:"port"`
  LastRequest time.Time `json:"-"`
  Id string `json:"-"`
  IpAdress string `json:"-"`
  Mutex sync.RWMutex `json:"-"`
  TypeNum int `json:"-"`
}

func ( route *Route ) Export( reverseResolveAuth bool ) ( newRouteCopied Route, error error ) {
  newRouteCopied.Name = route.Name
  newRouteCopied.TypeName = route.TypeName
  newRouteCopied.ScriptPath = route.ScriptPath
  var scriptCmdTmp []string 
  newRouteCopied.ScriptCmd =  append( scriptCmdTmp, route.ScriptCmd... )
  if reverseResolveAuth { 
    newRouteCopied.Authorization = route.AuthorizationDefault
  } else {
    newRouteCopied.Authorization = route.Authorization
  }
  envTmp := make( map[string]string )
  for key, value := range route.Environment {
    envTmp[key] = value 
  }
  newRouteCopied.Environment = envTmp
  newRouteCopied.Image = route.Image
  newRouteCopied.Timeout = route.Timeout
  newRouteCopied.Retry = route.Retry
  newRouteCopied.Delay = route.Delay
  newRouteCopied.Port = route.Port
  return newRouteCopied, nil
}

func ( route *Route ) Check() ( error error ) {
  switch ( route.TypeName ) {
  case "function":
    route.TypeNum = RouteTypeFunction
  case "service":
    route.TypeNum = RouteTypeService
  case "shell":
    route.TypeNum = RouteTypeShell
  default:
    error = errors.New( "type of route invalid" ) 
  }
  return error
}

func ( route *Route ) CreateFileEnv( tmpDir string ) ( fileEnvPath string, err error ) {
  fileEnvPath = filepath.Join(
    tmpDir,
    route.Name+".env", 
  )
  if _,err := os.Stat( fileEnvPath ); err == nil {
    return fileEnvPath, nil 
  } 
  fileEnv, err := os.Create( fileEnvPath )
  if err != nil {
    return "", errors.New( "env file for container failed" )
  }
  for key, value := range route.Environment {
    fileEnv.WriteString( key+"="+value+"\n" )
  }
  fileEnv.Close()
  return fileEnvPath, nil
}