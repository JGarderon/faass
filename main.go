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
  ApiConfiguration "api/configuration"
  ApiFunctions "api/functions"
  ApiServices "api/services"
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

  muxer := http.NewServeMux()

  UIPath := GLOBAL_CONF.UI 
  if UIPath != "" {
    Logger.Info( "UI path found :", UIPath )
    muxer.Handle( "/", http.FileServer( http.Dir( UIPath ) ) )
  }
 
  muxer.HandleFunc( "/lambda/", lambdaHandler )

  if GLOBAL_CONF.Authorization != "" {
    Logger.Info( "Authorization secret found ; API active" )
    muxer.Handle( 
      "/api/configuration", 
        ApiConfiguration.HandlerApi {
        Logger: &Logger, 
        ConfMutext: &GLOBAL_CONF_MUTEXT, 
        Conf: &GLOBAL_CONF, 
      }, 
    )
    muxer.Handle( 
      "/api/functions/", 
        ApiFunctions.HandlerApi {
        Logger: &Logger, 
        ConfMutext: &GLOBAL_CONF_MUTEXT, 
        Conf: &GLOBAL_CONF, 
      }, 
    )
    muxer.Handle( 
      "/api/services/", 
        ApiServices.HandlerApi {
        Logger: &Logger, 
        ConfMutext: &GLOBAL_CONF_MUTEXT, 
        Conf: &GLOBAL_CONF, 
      }, 
    )
  } else { 
    Logger.Info( "Authorization secret not found ; API inactive" )
  } 

  httpServer := &http.Server{
    Addr: GLOBAL_CONF.IncomingAdress+":"+strconv.Itoa( GLOBAL_CONF.IncomingPort ),
    Handler:     muxer,
  }

  signalChan := make(chan os.Signal, 1)
  go utils.CleanContainers( 
    ctx, 
    false,
    &GLOBAL_CONF_MUTEXT,
    &GLOBAL_CONF, 
    &GLOBAL_WAIT_GROUP, 
    &Logger,
  )
  go server.Run( &Logger, httpServer )
  signal.Notify(
    signalChan,
    syscall.SIGHUP,
    syscall.SIGINT,
    syscall.SIGQUIT,
    syscall.SIGTERM,
  )
  <-signalChan
  Logger.Info("interrupt received ; shutting down")

  if err := httpServer.Shutdown( ctx ); err != nil {
    Logger.Panic( "shutdown error: %v\n", err )
    defer os.Exit( configuration.ExitConfShuttingServerFailed )
  }

  cancel()

  GLOBAL_WAIT_GROUP.Wait()

  Logger.Info( "process gracefully stopped" )

}