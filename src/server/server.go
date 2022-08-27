package server

import ( 
  "net/http"
  "os"
  // -----------
  "configuration"
  "logger"
)

// -----------------------------------------------

func Run ( logger *logger.Logger, httpServer *http.Server ) {
  defer logger.Warning( "Shutdown ListenAndServeTLS terminated" )
  err := httpServer.ListenAndServeTLS(
    "server.crt",
    "server.key",
  )
  if err != nil && err != http.ErrServerClosed {
    logger.Panicf( "General internal server error : %v", err )
    os.Exit( configuration.ExitUndefined )
  }
}

