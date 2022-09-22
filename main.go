package main
/*
  FaasS = Function as a (Simple) Service
  ---
  Created by Julien Garderon (Nothus)
  from August 01 to September 06, 2022
  MIT Licence
  ---
  This is a POC - Proof of Concept -, based on the idea of the OpenFaas project
  /!\ NOT INTENDED FOR PRODUCTION, only dev /!\
*/

import (
  "os/signal"
  "net/http"
  "context"
  "strconv"
  "syscall"
  "regexp"
  "sync"
  "os"
  // "fmt"
  // -----------
  "logger"
  "configuration"
  "configuration/utils"
  "server"
)

// -----------------------------------------------

var GLOBAL_CONF_PATH string
var GLOBAL_CONF configuration.Conf
var GLOBAL_CONF_MUTEXT sync.RWMutex 

var GLOBAL_REGEX_ROUTE_NAME *regexp.Regexp

var GLOBAL_WAIT_GROUP sync.WaitGroup

// -----------------------------------------------

var Logger logger.Logger

// -----------------------------------------------

func main() { 

  Logger.Init()
  utils.StartEnv( 
    &GLOBAL_CONF_MUTEXT,
    &GLOBAL_CONF_PATH, 
    &GLOBAL_CONF, 
    &Logger, 
  )

  GLOBAL_REGEX_ROUTE_NAME = utils.CreateRegexUrl()

  ctx := context.Background()
  ctx, cancel := context.WithCancel( context.Background() )

  go utils.CleanContainers( 
    ctx, 
    false,
    &GLOBAL_CONF_MUTEXT,
    &GLOBAL_CONF, 
    &GLOBAL_WAIT_GROUP, 
    &Logger,
  )
  
  signalChan := make( chan os.Signal, 1 )
  signal.Notify(
    signalChan,
    syscall.SIGINT,
    syscall.SIGQUIT,
    syscall.SIGTERM,
  )
  restartChan := make( chan os.Signal, 1 )
  signal.Notify(
    restartChan,
    syscall.SIGHUP,
  )
  continueServer := true
  for continueServer == true {
    Logger.Info("server start")
    httpServer := &http.Server{
      Addr:     GLOBAL_CONF.IncomingAdress+":"+strconv.Itoa( GLOBAL_CONF.IncomingPort ),
      Handler:  server.CreateServeMux(
        &GLOBAL_CONF,
        &GLOBAL_CONF_MUTEXT, 
        &Logger,
        GLOBAL_REGEX_ROUTE_NAME, 
      ),
    }
    go server.Run( &GLOBAL_CONF, &Logger, httpServer )
    select {
      case <-signalChan: 
        if err := httpServer.Shutdown( ctx ); err != nil {
          Logger.Panic( "shutdown error: %v\n", err )
          defer os.Exit( configuration.ExitConfShuttingServerFailed )
        }
        Logger.Info("interrupt received ; shutting down")
        continueServer = false
      case <-restartChan: 
        if err := httpServer.Shutdown( ctx ); err != nil {
          Logger.Panic( "restart failed ; shutdown error: %v\n", err )
          defer os.Exit( configuration.ExitConfShuttingServerFailed )
        }
        Logger.Info("restart received")
    }
  }

  cancel()

  GLOBAL_WAIT_GROUP.Wait()

  Logger.Info( "process gracefully stopped" )

}