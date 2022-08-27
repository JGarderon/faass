package main

import (
  "os"
  "path/filepath"
  "regexp"
  "errors"
  "flag"
  "time"
  "context"
  // -----------
  "configuration"
)

// -----------------------------------------------

func CreateRegexUrl() {
  regex, err := regexp.Compile( "^([a-z0-9_-]+)" )
  if err != nil {
    os.Exit( configuration.ExitConfRegexUrlKo )
  }
  GLOBAL_REGEX_ROUTE_NAME = regex
}

func GetRootPath() (rootPath string, err error) {
  ex, err := os.Executable()
  if err != nil {
    Logger.Warningf( "unable to get current path of executable : %s", err )
    return "", errors.New( "unable to get current path of executable" )
  }
  rootPath = filepath.Dir( ex )
  return rootPath, nil
}

func CreateEnv() bool {
  rootPath, err := GetRootPath()
  if err != nil {
    Logger.Warningf( "unable to get root path : %v", err )
    return false
  }
  newConf := configuration.Conf{} 
  if !newConf.PopulateDefaults(rootPath) {
    Logger.Error( "Unable to create environment : conf" )
    return false
  }
  if !newConf.Export( GLOBAL_CONF_PATH ) {
    Logger.Error( "Unable to export environment : conf" )
    return false
  }
  if err := os.Mkdir( newConf.UI, os.ModePerm ); err != nil {
    Logger.Warningf( "Unable to create environment for UI contents \"%v\" : %v ; pass", newConf.UI, err )
    return false
  }
  if err := os.Mkdir( newConf.TmpDir, os.ModePerm ); err != nil {
    Logger.Warningf( "Unable to create environment for temporary contents \"%v\" : %v ; pass", newConf.TmpDir, err )
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
      os.Exit( configuration.ExitOk )
    } else {
      os.Exit( configuration.ExitConfCreateKo )
    }
  }
  err := configuration.Import( GLOBAL_CONF_PATH, &GLOBAL_CONF )  
  if err != nil {
    Logger.Panicf( "unable to load configuration with error : %v", err )
    os.Exit( configuration.ExitConfLoadKo )
  } 
  if message, state := GLOBAL_CONF.Check() ; !state {
    Logger.Panicf( "check of conf failed : %v", message ) 
    os.Exit( configuration.ExitConfCheckKo )
  }
  GLOBAL_CONF.Logger = &Logger
  GLOBAL_CONF.Containers.Logger = &Logger
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
