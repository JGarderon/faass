package main
/*
  FaasS = Function as a (Simple) Service
  ---
  Created by Julien Garderon (Nothus)
  from August 01 to 19, 2022
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
  // -----------
  "logger"
  "configuration"
  ApiConfiguration "api/configuration"
  ApiFunction "api/functions"
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

func main() { 

  Logger.Init()
  StartEnv()

  CreateRegexUrl()

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
    muxer.HandleFunc( "/api/", ApiHandler )

    muxer.Handle( 
      "/api/configuration/", 
        ApiConfiguration.HandlerApi {
        Logger: &Logger, 
        ConfMutext: &GLOBAL_CONF_MUTEXT, 
        Conf: &GLOBAL_CONF, 
      }, 
    )

    muxer.Handle( 
      "/api/functions/", 
        ApiFunction.HandlerApi {
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
  go CleanContainers( ctx, false )
  go RunServer( httpServer )
  signal.Notify(
    signalChan,
    syscall.SIGHUP,
    syscall.SIGINT,
    syscall.SIGQUIT,
  )
  <-signalChan
  Logger.Info("interrupt received ; shutting down")

  if err := httpServer.Shutdown( ctx ); err != nil {
    Logger.Panic( "shutdown error: %v\n", err )
    defer os.Exit( ExitConfShuttingServerFailed )
  }

  cancel()

  GLOBAL_WAIT_GROUP.Wait()

  Logger.Info( "process gracefully stopped" )

}