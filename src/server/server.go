package server

import ( 
  "net/http"
  "os"
  // -----------
  "configuration"
  "logger"
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

