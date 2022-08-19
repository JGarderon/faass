package main

import (
  "os"
  "path/filepath"
  "regexp"
  "errors"
  "flag"
  "strconv"
  "time"
  "context"
  // -----------
  "itinerary"
  "configuration"
)

// -----------------------------------------------

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
    Logger.Warning( "unable to get current path of executable : ", err )
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
  newConf := configuration.Conf {
    Domain: "https://localhost",
    Authorization: "Basic YWRtaW46YXplcnR5", // admin:azerty 
    IncomingPort: 9090,
    UI: uiTmpDir, 
    TmpDir: pathTmpDir,
    Prefix: "lambda",
    Routes: newMapRoutes, 
  }
  if !newConf.Export( GLOBAL_CONF_PATH ) {
    Logger.Error( "Unable to create environment : conf" )
    return false
  }
  if err := os.Mkdir( uiTmpDir, os.ModePerm ); err != nil {
    Logger.Warning( "Unable to create environment : content dir (", err, "); pass" )
    return false
  }
  if err := os.Mkdir( pathTmpDir, os.ModePerm ); err != nil {
    Logger.Warning( "Unable to create environment : tmp dir (", err, "); pass" )
    return false
  }
  return true
}

func StartEnv() {
  testLogger := flag.String( "testlogger", "", "test logger (value of print ; string)" ) 
  confPath := flag.String( "conf", "./conf.json", "path to conf (JSON ; string)" ) 
  prepareEnv := flag.Bool( "prepare", false, "create environment (conf+dir ; bool)" )
  flag.Parse()
  if *testLogger != "" {
    Logger.Test( *testLogger ) 
  }
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
  GLOBAL_CONF.Logger = &Logger
  GLOBAL_CONF.Containers.Logger = &Logger
  if !GLOBAL_CONF.ConfImport( GLOBAL_CONF_PATH ) {
    Logger.Panic( "Unable to load configuration" )
    os.Exit( ExitConfLoadKo )
  }
  if GLOBAL_CONF.IncomingPort < 1 || GLOBAL_CONF.IncomingPort > 65535 {
    Logger.Panic(
      "Bad configuration : incorrect port '"+strconv.Itoa( GLOBAL_CONF.IncomingPort )+"'", 
    )
    os.Exit( ExitConfLoadKo )
  }
  rootPath, err := GetRootPath()
  if err != nil {
    Logger.Panic( "Unable to root path of executable" )
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
            Logger.Warning( "Container ", route.Name, "(cId ", route.Id, ") : state unknow ; ", err )
          } else if state != "exited" && routeDelayLastRequest.Before( time.Now() ) {
            _, err := GLOBAL_CONF.Containers.Stop( route )
            if err != nil {
              Logger.Warning( "Container ", route.Name, "(cId ", route.Id, ") not stopped - maybe he is still active ?" )
            } else {
              Logger.Info( "Container", route.Name, "(cId ", route.Id, ") stopped"  )
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
            Logger.Warning( "Container ", route.Name, "(cId ", rId, ") not stopped - maybe he is still active ?" )
          } else {
            Logger.Info( "Container ", route.Name, "(cId ", rId, ") stopped" )
            time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
            _, err := GLOBAL_CONF.Containers.Remove( route )
            if err != nil {
              Logger.Warning( "Container ", route.Name, "(cId ", rId, ") not terminated" )
            } else {
              Logger.Info( "Container ", route.Name, "(ex-cId ", rId, ") terminated" )
            }
          }
        }
      }
      return
    }
  }
}
