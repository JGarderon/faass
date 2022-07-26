package utils

import (
  "os"
  "path/filepath"
  "regexp"
  "errors"
  "flag"
  "time"
  "context"
  "fmt"
  "sync"
  "os/exec"
  // -----------
  "logger"
  "configuration"
)

// -----------------------------------------------

func CreateRegexUrl() *regexp.Regexp {
  regex, err := regexp.Compile( "^([a-z0-9_-]+)" )
  if err != nil {
    os.Exit( configuration.ExitConfRegexUrlKo )
  }
  return regex 
}

func GetRootPath() (rootPath string, err error) {
  ex, err := os.Executable()
  if err != nil {
    // Logger.Warningf( "unable to get current path of executable : %s", err )
    return "", errors.New( "unable to get current path of executable" )
  }
  rootPath = filepath.Dir( ex )
  return rootPath, nil
}

func CreateEnv( pathExport string ) ( bool, string ) {
  rootPath, err := GetRootPath()
  if err != nil {
    return false, fmt.Sprintf( "unable to get root path : %v", err )
  }
  newConf := configuration.Conf{} 
  if !newConf.PopulateDefaults(rootPath) {
    return false, fmt.Sprintf( "Unable to export environment's conf" )
  }
  if err := newConf.Export( pathExport, false ) ; err != nil {
    return false, fmt.Sprintf( "Unable to export environment's conf :", err )
  }
  if err := os.Mkdir( newConf.UI, os.ModePerm ); err != nil {
    return false, fmt.Sprintf( "Unable to create environment for UI contents \"%v\" : %v ; pass", newConf.UI, err )
  }
  if err := os.Mkdir( newConf.TmpDir, os.ModePerm ); err != nil {
    return false, fmt.Sprintf( "Unable to create environment for temporary contents \"%v\" : %v ; pass", newConf.TmpDir, err )
  }
  return true, ""
}

func StartEnv( globalConfMutex *sync.RWMutex, globalConfPath *string, globalConf *configuration.Conf, logger *logger.Logger ) {
  testLogger := flag.String( "testlogger", "", "test logger (value of print ; string)" ) 
  confPath := flag.String( "conf", "./conf.json", "path to conf (JSON ; string)" ) 
  prepareEnv := flag.Bool( "prepare", false, "create environment (conf+dir ; bool)" )
  pullImageContainer := flag.Bool( "pulling", false, "pull image's containers (bool)" )
  pullImageContainerOnly := flag.Bool( "pulling-only", false, "pull image's containers and exit (bool)" )
  flag.Parse()
  if *testLogger != "" {
    logger.Test( *testLogger ) 
  }
  globalConfMutex.Lock() 
  defer globalConfMutex.Unlock() 
  globalConfPath = confPath
  if *prepareEnv {
    if state, mError := CreateEnv( *globalConfPath ) ; state {
      os.Exit( configuration.ExitOk )
    } else {
      logger.Warning( mError ) 
      os.Exit( configuration.ExitConfCreateKo )
    }
  }
  err := configuration.Import( *globalConfPath, globalConf )  
  if err != nil {
    logger.Panicf( "unable to load configuration with error : %v", err )
    os.Exit( configuration.ExitConfLoadKo )
  } 
  if err := globalConf.Check() ; err != nil {
    logger.Panicf( "check of conf failed : %v", err ) 
    os.Exit( configuration.ExitConfCheckKo )
  }
  globalConf.Logger = logger
  globalConf.Containers.PathCmd = globalConf.PathCmdContainer
  globalConf.Containers.Logger = logger
  if err := globalConf.ResolveAuth() ; err != nil {
    logger.Panicf( "check of conf (auth part's) failed : %v", err ) 
    os.Exit( configuration.ExitConfAuthCheckKo )
  }
  if *pullImageContainerOnly || *pullImageContainer {
    if err := PullImageContainers( globalConf, logger ) ; err != nil {
      logger.Panicf( "image's container pulling failed : %v", err ) 
      os.Exit( configuration.ExitImageContainersPullFailed )
    }
  } else { 
    logger.Warningf( "no pulling images'containers" ) 
  }
  if *pullImageContainerOnly {
    logger.Warningf( "only pulling images'containers wanted" ) 
    os.Exit( configuration.ExitOk )
  }
}

// -----------------------------------------------

func PullImageContainers( globalConf *configuration.Conf, logger *logger.Logger ) ( err error ) {
  for routeName, route := range globalConf.Routes {
    if route.Image == "" { 
      logger.Infof( "route '%v' haven't image's container", routeName ) 
      continue
    }
    logger.Infof( "image's container '%v' pulling started (%v)", routeName, route.Image ) 
    args := []string{ 
        "image", "pull",
        route.Image,
    }   
    cmd := exec.Command( globalConf.PathCmdContainer, args... )
    if err := cmd.Run() ; err != nil { 
      err = errors.New( 
        fmt.Sprintf( "image's container '%v' (%v) pulling terminated (with error)", routeName, route.Image ), 
      ) 
      return err 
    } else { 
      logger.Infof( "image's container '%v' pulling terminated (without error)", routeName ) 
    }
  }
  return err 
}

// -----------------------------------------------

func CleanContainers( ctx context.Context, force bool, globalConfMutex *sync.RWMutex, globalConf *configuration.Conf, globalWaitGroup *sync.WaitGroup, logger *logger.Logger ) {
  globalWaitGroup.Add( 1 )
  for {
    tt := time.After( time.Duration( globalConf.DelayCleaningContainers ) * time.Second )
    select {
    case <-tt:
      for routeName := range globalConf.Routes {
        route := globalConf.Routes[routeName]
        if route.Id != "" {
          routeDelayLastRequest := route.LastRequest.Add( time.Duration( route.Delay ) * time.Second )
          globalConfMutex.RLock()
          state, err := globalConf.Containers.Check( route ) 
          if err != nil {
            logger.Warning( "Container ", route.Name, "(cId ", route.Id, ") : state unknow ; ", err )
          } else if state != "exited" && routeDelayLastRequest.Before( time.Now() ) {
            _, err := globalConf.Containers.Stop( route )
            if err != nil {
              logger.Warning( "Container ", route.Name, "(cId ", route.Id, ") not stopped - maybe he is still active ?" )
            } else {
              logger.Info( "Container", route.Name, "(cId ", route.Id, ") stopped"  )
            }
          }
          globalConfMutex.RUnlock()
        }
      }
    case <-ctx.Done():
      globalConfMutex.RLock()
      defer globalConfMutex.RUnlock()
      defer globalWaitGroup.Done()
      for routeName := range globalConf.Routes {
        route := globalConf.Routes[routeName]
        rId := route.Id
        if rId != "" {
          _, err := globalConf.Containers.Stop( route )
          if err != nil {
            logger.Warning( "Container ", route.Name, "(cId ", rId, ") not stopped - maybe he is still active ?" )
          } else {
            logger.Info( "Container ", route.Name, "(cId ", rId, ") stopped" )
            time.Sleep( time.Duration( route.Timeout ) * time.Millisecond )
            _, err := globalConf.Containers.Remove( route )
            if err != nil {
              logger.Warning( "Container ", route.Name, "(cId ", rId, ") not terminated" )
            } else {
              logger.Info( "Container ", route.Name, "(ex-cId ", rId, ") terminated" )
            }
          }
        }
      }
      return
    }
  }
}
