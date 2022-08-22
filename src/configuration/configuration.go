package configuration

import(
  "errors"
  "os"
  "path/filepath"
  "log"
  "io/ioutil"
  "encoding/json"
  "strconv"
  // -----------
  "itinerary"
  "executors"
  "logger"
)

type Conf struct {
  Logger *logger.Logger `json:"-"`
  Containers executors.Containers `json:"-"`
  Domain string `json:"domain"`
  Authorization string `json:"authorization"`
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
  if c.DelayCleaningContainers < 5 {
    c.DelayCleaningContainers = 5
    message = "new value for delay cleaning containers : min 5 (seconds)"
  }
  if c.DelayCleaningContainers > 60 {
    c.DelayCleaningContainers = 60
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
    "./content",
  )
  pathTmpDir := filepath.Join(
    rootPath,
    "./tmp",
  )
  c.Domain = "https://localhost"
  c.Authorization = "Basic YWRtaW46YXplcnR5" // admin:azerty
  c.IncomingPort = 9090
  c.DelayCleaningContainers = 0
  c.UI = uiTmpDir
  c.TmpDir = pathTmpDir
  c.Prefix = "lambda"
  newMapRoutes := make( map[string]*itinerary.Route )
  newMapEnvironmentRoute := make( map[string]string )
  newMapEnvironmentRoute["faass-example"] = "true"
  newMapRoutes["example-service"] = &itinerary.Route {
      Name: "exampleService",
      IsService: true,
      Authorization: "Basic YWRtaW46YXplcnR5",
      Environment: newMapEnvironmentRoute,
      Image: "nginx",
      Timeout : 250,
      Retry: 3,
      Delay: 8,
      Port: 80,
  }
  newMapRoutes["example-function"] = &itinerary.Route {
      Name: "exampleFunction",
      IsService: false,
      ScriptPath: filepath.Join( pathTmpDir, "./example-function.py" ),
      ScriptCmd: []string{ "python3", "/function" },
      Environment: newMapEnvironmentRoute,
      Image: "python3",
      Timeout : 500,
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