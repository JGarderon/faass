package itinerary

import( 
  "time"
  "sync"
  "path/filepath"
  "os"
  "errors"
)

type Route struct {
  Name string `json:"name"`
  IsService bool `json:"service"`
  ScriptPath string `json:"script"`
  ScriptCmd []string `json:"cmd"`
  Authorization string `json:"authorization"`
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