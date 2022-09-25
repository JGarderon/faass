package server

import ( 
  "net/http"
  "os"
  "sync"
  "regexp"
  // -----------
  "configuration"
  "logger"
  "server/lambda"
  ApiConfiguration "api/configuration"
  ApiFunctions "api/functions"
  ApiServices "api/services"
)

// -----------------------------------------------

func Run ( conf *configuration.Conf, logger *logger.Logger, httpServer *http.Server ) {
  defer logger.Warning( "Shutdown ListenAndServeTLS terminated" )
  exitWithError := false 
  if conf.IncomingTLS != "" {
    err := httpServer.ListenAndServeTLS(
      conf.IncomingTLSCrt,
      conf.IncomingTLSKey,
    ) 
    if err != nil && err != http.ErrServerClosed {
      logger.Panicf( "General internal TLS server error : %v", err )
      exitWithError = true
    }
  } else {
    err := httpServer.ListenAndServe()
    if err != nil && err != http.ErrServerClosed {
      logger.Panicf( "General internal server error : %v", err )
      exitWithError = true
    }
  }
  if exitWithError {
    os.Exit( configuration.ExitUndefined )
  }
}

func CreateServeMux( c *configuration.Conf, m *sync.RWMutex, l *logger.Logger, r *regexp.Regexp ) *http.ServeMux {
  m.RLock()
  defer m.RUnlock()
  muxer := http.NewServeMux()
  UIPath := c.UI 
  if UIPath != "" {
    l.Info( "UI path found :", UIPath )
    muxer.Handle( "/", http.FileServer( http.Dir( UIPath ) ) )
  }
  muxer.Handle( 
    "/lambda/", 
    lambda.HandlerLambda {
      GlobalRouteRegex: r,
      Logger: l, 
      ConfMutext: m, 
      Conf: c, 
    }, 
  )
  if c.AuthorizationAPI != "" {
    l.Info( "Authorization secret API found ; API active" )
    muxer.Handle( 
      "/api/configuration", 
      ApiConfiguration.HandlerApi {
        Logger: l, 
        ConfMutext: m, 
        Conf: c, 
      }, 
    )
    muxer.Handle( 
      "/api/functions/", 
      ApiFunctions.HandlerApi {
        Logger: l, 
        ConfMutext: m, 
        Conf: c, 
      }, 
    )
    muxer.Handle( 
      "/api/services/", 
      ApiServices.HandlerApi {
        Logger: l, 
        ConfMutext: m, 
        Conf: c, 
      }, 
    )
  } else { 
    l.Info( "Authorization secret API not found ; API inactive" )
  } 
  return muxer 
}

