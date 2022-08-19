package logger 

import (
  "log"
  "os"
)

type Logger struct {
  Debug       func(v ...any)
  Debugf      func(f string, v ...any)
  Info        func(v ...any)
  Infof       func(f string, v ...any)
  Warning     func(v ...any)
  Warningf    func(f string, v ...any)
  Error       func(v ...any)
  Errorf      func(f string, v ...any)
  Panic       func(v ...any)
  Panicf      func(f string, v ...any)
}

func ( logger *Logger ) Init() {
  logger.Debug      = log.New( os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile ).Println
  logger.Debugf     = log.New( os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile ).Printf
  logger.Info       = log.New( os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile ).Println
  logger.Infof      = log.New( os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile ).Printf
  logger.Warning    = log.New( os.Stderr, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile ).Println
  logger.Warningf   = log.New( os.Stderr, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile ).Printf
  logger.Error      = log.New( os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile ).Println
  logger.Errorf     = log.New( os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile ).Printf
  logger.Panic      = log.New( os.Stderr, "PANIC: ", log.Ldate|log.Ltime|log.Lshortfile ).Println
  logger.Panicf     = log.New( os.Stderr, "PANIC: ", log.Ldate|log.Ltime|log.Lshortfile ).Printf
}

func ( logger *Logger ) Test ( m string ) bool {
  logger.Debug( m )
  logger.Info( m )
  logger.Warning( m )
  logger.Error( m )
  logger.Panic( m ) 
  return true
}