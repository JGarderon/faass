package logger 

import (
  "log"
  "os"
)

type Logger struct {
  debugRealLogger       *log.Logger
  infoRealLogger        *log.Logger
  warningRealLogger     *log.Logger
  errorRealLogger       *log.Logger
  panicRealLogger       *log.Logger
}

func ( logger *Logger ) Debug ( v ...interface{} ) {
  logger.debugRealLogger.Println( v... ) 
}

func ( logger *Logger ) Debugf ( f string, v ...interface{} ) {
  logger.debugRealLogger.Printf( f, v... ) 
}

func ( logger *Logger ) Info ( v ...interface{} ) {
  logger.infoRealLogger.Println( v... ) 
}

func ( logger *Logger ) Infof ( f string, v ...interface{} ) {
  logger.infoRealLogger.Printf( f, v... ) 
}

func ( logger *Logger ) Error ( v ...interface{} ) {
  logger.errorRealLogger.Println( v... ) 
}

func ( logger *Logger ) Errorf ( f string, v ...interface{} ) {
  logger.errorRealLogger.Printf( f, v... ) 
}

func ( logger *Logger ) Warning ( v ...interface{} ) {
  logger.warningRealLogger.Println( v... ) 
}

func ( logger *Logger ) Warningf ( f string, v ...interface{} ) {
  logger.warningRealLogger.Printf( f, v... ) 
}

func ( logger *Logger ) Panic ( v ...interface{} ) {
  logger.panicRealLogger.Println( v... ) 
}

func ( logger *Logger ) Panicf ( f string, v ...interface{} ) {
  logger.panicRealLogger.Printf( f, v... ) 
}

func ( logger *Logger ) Init() {
  logger.debugRealLogger      = log.New( os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile )
  logger.infoRealLogger       = log.New( os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile )
  logger.warningRealLogger    = log.New( os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile )
  logger.errorRealLogger      = log.New( os.Stdout, "IERROR: ", log.Ldate|log.Ltime|log.Lshortfile )
  logger.panicRealLogger      = log.New( os.Stdout, "PANIC: ", log.Ldate|log.Ltime|log.Lshortfile )
}

func ( logger *Logger ) Test ( m string ) bool {
  logger.Debug( m )
  logger.Info( m )
  logger.Warning( m )
  logger.Error( m )
  logger.Panic( m ) 
  return true
}